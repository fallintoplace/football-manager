package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	iceberg "github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	"github.com/apache/iceberg-go/catalog/rest"
	"github.com/apache/iceberg-go/table"
	aws "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type MatchdayPayload struct {
	Season  int           `json:"season"`
	Round   int           `json:"round"`
	Clubs   []Club        `json:"clubs"`
	Players []Player      `json:"players"`
	Results []MatchResult `json:"results"`
}

type Club struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
}

type Player struct {
	ID       string `json:"id"`
	ClubID   string `json:"clubId"`
	Name     string `json:"name"`
	Position string `json:"position"`
}

type MatchResult struct {
	FixtureID     string             `json:"fixtureId"`
	Round         int                `json:"round"`
	HomeID        string             `json:"homeId"`
	AwayID        string             `json:"awayId"`
	HomeGoals     int                `json:"homeGoals"`
	AwayGoals     int                `json:"awayGoals"`
	Events        []MatchEvent       `json:"events"`
	Trace         []MatchFrame       `json:"trace"`
	Metrics       MatchMetrics       `json:"metrics"`
	PlayerRatings map[string]float64 `json:"playerRatings"`
}

type MatchEvent struct {
	Minute     int      `json:"minute"`
	Type       string   `json:"type"`
	TeamID     string   `json:"teamId"`
	PlayerName string   `json:"playerName"`
	Text       string   `json:"text"`
	XG         *float64 `json:"xg"`
}

type MatchFrame struct {
	Minute           float64       `json:"minute"`
	Phase            string        `json:"phase"`
	PossessingTeamID string        `json:"possessingTeamId"`
	Ball             TraceBall     `json:"ball"`
	Players          []TracePlayer `json:"players"`
}

type TraceBall struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type TracePlayer struct {
	ID       string  `json:"id"`
	TeamID   string  `json:"teamId"`
	Name     string  `json:"name"`
	Position string  `json:"position"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	TargetX  float64 `json:"targetX"`
	TargetY  float64 `json:"targetY"`
	Intent   string  `json:"intent"`
}

type MatchMetrics struct {
	HomeXG            float64 `json:"homeXg"`
	AwayXG            float64 `json:"awayXg"`
	HomeShots         int     `json:"homeShots"`
	AwayShots         int     `json:"awayShots"`
	HomeShotsOnTarget int     `json:"homeShotsOnTarget"`
	AwayShotsOnTarget int     `json:"awayShotsOnTarget"`
	HomePossession    float64 `json:"homePossession"`
	HomePressure      float64 `json:"homePressure"`
	AwayPressure      float64 `json:"awayPressure"`
	HomeTerritory     float64 `json:"homeTerritory"`
}

type AnalyticsPlayer struct {
	PlayerID      string  `json:"playerId"`
	PlayerName    string  `json:"playerName"`
	Matches       uint64  `json:"matches"`
	AverageRating float64 `json:"averageRating"`
}

type AnalyticsSummary struct {
	Matches           uint64            `json:"matches"`
	Events            uint64            `json:"events"`
	Frames            uint64            `json:"frames"`
	AverageXG         float64           `json:"averageXg"`
	AveragePossession float64           `json:"averagePossession"`
	Players           []AnalyticsPlayer `json:"players"`
	Source            string            `json:"source"`
}

type Store struct {
	clickhouse driver.Conn
	iceberg    *icebergWriter
	mu         sync.Mutex
}

type icebergWriter struct {
	baseURL  string
	endpoint string
	username string
	password string
	mu       sync.Mutex
	catalog  *rest.Catalog
}

func main() {
	store, err := newStore()
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", store.handleHealth)
	mux.HandleFunc("/ingest", store.handleIngest)
	mux.HandleFunc("/api/analytics/summary", store.handleSummary)

	server := &http.Server{
		Addr:              env("ANALYTICS_ADDR", ":8787"),
		Handler:           cors(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("touchline analytics listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func newStore() (*Store, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{env("CLICKHOUSE_ADDR", "localhost:9002")},
		Auth: clickhouse.Auth{
			Database: env("CLICKHOUSE_DATABASE", "default"),
			Username: env("CLICKHOUSE_USERNAME", "default"),
			Password: env("CLICKHOUSE_PASSWORD", ""),
		},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("open ClickHouse: %w", err)
	}

	return &Store{
		clickhouse: conn,
		iceberg: &icebergWriter{
			baseURL:  env("ICEBERG_REST_URL", "http://localhost:8181"),
			endpoint: env("AWS_S3_ENDPOINT", "http://localhost:9000"),
			username: env("AWS_ACCESS_KEY_ID", "admin"),
			password: env("AWS_SECRET_ACCESS_KEY", "password"),
		},
	}, nil
}

func (s *Store) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	clickhouseReady := s.clickhouse.Ping(ctx) == nil
	icebergReady := s.iceberg.ready(ctx) == nil
	writeJSON(w, http.StatusOK, map[string]bool{
		"clickhouse": clickhouseReady,
		"iceberg":    icebergReady,
	})
}

func (s *Store) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}

	var payload MatchdayPayload
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<20))
	if err := decoder.Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid matchday payload"})
		return
	}
	if len(payload.Results) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "matchday has no results"})
		return
	}

	if err := s.ingest(r.Context(), payload); err != nil {
		log.Printf("ingest failed: %v", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "analytics stores unavailable"})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": true, "matches": len(payload.Results)})
}

func (s *Store) handleSummary(w http.ResponseWriter, r *http.Request) {
	clubID := r.URL.Query().Get("club_id")
	if clubID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "club_id is required"})
		return
	}

	summary, err := s.summary(r.Context(), clubID)
	if err != nil {
		log.Printf("summary failed: %v", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "analytics store unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Store) ingest(ctx context.Context, payload MatchdayPayload) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	if err := ensureClickHouseSchema(ctx, s.clickhouse); err != nil {
		return err
	}
	if err := s.iceberg.ready(ctx); err != nil {
		return err
	}

	for _, result := range payload.Results {
		exists, err := s.hasMatch(ctx, result.FixtureID)
		if err != nil {
			return err
		}
		if exists {
			return nil
		}
	}

	if err := insertClickHouse(ctx, s.clickhouse, payload); err != nil {
		return err
	}
	if err := s.iceberg.append(ctx, payload); err != nil {
		return err
	}
	return nil
}

func (s *Store) hasMatch(ctx context.Context, matchID string) (bool, error) {
	var count uint64
	if err := s.clickhouse.QueryRow(ctx, "SELECT count() FROM touchline_matches WHERE match_id = ?", matchID).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) summary(ctx context.Context, clubID string) (AnalyticsSummary, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := ensureClickHouseSchema(ctx, s.clickhouse); err != nil {
		return AnalyticsSummary{}, err
	}

	var summary AnalyticsSummary
	if err := s.clickhouse.QueryRow(ctx, `
		SELECT count(), coalesce(avg(home_xg + away_xg), 0), coalesce(avg(home_possession), 0)
		FROM touchline_matches
		WHERE home_id = ? OR away_id = ?`, clubID, clubID).Scan(&summary.Matches, &summary.AverageXG, &summary.AveragePossession); err != nil {
		return AnalyticsSummary{}, err
	}
	if err := s.clickhouse.QueryRow(ctx, `
		SELECT count()
		FROM touchline_match_events
		WHERE match_id IN (SELECT match_id FROM touchline_matches WHERE home_id = ? OR away_id = ?)`, clubID, clubID).Scan(&summary.Events); err != nil {
		return AnalyticsSummary{}, err
	}
	if err := s.clickhouse.QueryRow(ctx, `
		SELECT count()
		FROM touchline_player_frames
		WHERE match_id IN (SELECT match_id FROM touchline_matches WHERE home_id = ? OR away_id = ?)`, clubID, clubID).Scan(&summary.Frames); err != nil {
		return AnalyticsSummary{}, err
	}

	rows, err := s.clickhouse.Query(ctx, `
		SELECT player_id, any(player_name), count(), avg(rating)
		FROM touchline_player_ratings
		WHERE club_id = ?
		GROUP BY player_id
		ORDER BY avg(rating) DESC
		LIMIT 8`, clubID)
	if err != nil {
		return AnalyticsSummary{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var player AnalyticsPlayer
		if err := rows.Scan(&player.PlayerID, &player.PlayerName, &player.Matches, &player.AverageRating); err != nil {
			return AnalyticsSummary{}, err
		}
		summary.Players = append(summary.Players, player)
	}
	if err := rows.Err(); err != nil {
		return AnalyticsSummary{}, err
	}
	summary.Source = "ClickHouse + Iceberg"
	return summary, nil
}

func ensureClickHouseSchema(ctx context.Context, conn driver.Conn) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS touchline_clubs (
			season UInt32,
			club_id String,
			name String,
			short_name String,
			ingested_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = ReplacingMergeTree(ingested_at) ORDER BY (season, club_id)`,
		`CREATE TABLE IF NOT EXISTS touchline_matches (
			match_id String,
			season UInt32,
			round UInt32,
			home_id String,
			away_id String,
			home_goals UInt32,
			away_goals UInt32,
			home_xg Float32,
			away_xg Float32,
			home_shots UInt32,
			away_shots UInt32,
			home_shots_on_target UInt32,
			away_shots_on_target UInt32,
			home_possession Float32,
			home_pressure Float32,
			away_pressure Float32,
			home_territory Float32,
			result String,
			ingested_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = MergeTree ORDER BY (season, round, match_id)`,
		`CREATE TABLE IF NOT EXISTS touchline_match_events (
			event_key String,
			match_id String,
			season UInt32,
			round UInt32,
			minute UInt32,
			event_type LowCardinality(String),
			team_id String,
			player_id String,
			player_name String,
			text String,
			xg Float32,
			ingested_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = MergeTree ORDER BY (season, match_id, minute, event_key)`,
		`CREATE TABLE IF NOT EXISTS touchline_player_frames (
			frame_key String,
			match_id String,
			season UInt32,
			round UInt32,
			frame_index UInt32,
			minute Float32,
			phase LowCardinality(String),
			possessing_team_id String,
			ball_x Float32,
			ball_y Float32,
			player_id String,
			team_id String,
			player_name String,
			position LowCardinality(String),
			player_x Float32,
			player_y Float32,
			target_x Float32,
			target_y Float32,
			intent LowCardinality(String),
			ingested_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = MergeTree ORDER BY (season, match_id, frame_index, player_id)`,
		`CREATE TABLE IF NOT EXISTS touchline_player_ratings (
			rating_key String,
			match_id String,
			season UInt32,
			round UInt32,
			player_id String,
			club_id String,
			player_name String,
			rating Float32,
			opponent_id String,
			ingested_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = MergeTree ORDER BY (season, player_id, match_id)`,
	}
	for _, statement := range statements {
		if err := conn.Exec(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func insertClickHouse(ctx context.Context, conn driver.Conn, payload MatchdayPayload) error {
	if len(payload.Clubs) > 0 {
		clubs, err := conn.PrepareBatch(ctx, `INSERT INTO touchline_clubs (
			season, club_id, name, short_name
		)`)
		if err != nil {
			return err
		}
		for _, club := range payload.Clubs {
			if err := clubs.Append(payload.Season, club.ID, club.Name, club.ShortName); err != nil {
				return err
			}
		}
		if err := clubs.Send(); err != nil {
			return err
		}
	}

	matches, err := conn.PrepareBatch(ctx, `INSERT INTO touchline_matches (
		match_id, season, round, home_id, away_id, home_goals, away_goals, home_xg, away_xg,
		home_shots, away_shots, home_shots_on_target, away_shots_on_target, home_possession,
		home_pressure, away_pressure, home_territory, result
	)`)
	if err != nil {
		return err
	}
	for _, result := range payload.Results {
		if err := matches.Append(
			result.FixtureID, payload.Season, result.Round, result.HomeID, result.AwayID,
			result.HomeGoals, result.AwayGoals, result.Metrics.HomeXG, result.Metrics.AwayXG,
			result.Metrics.HomeShots, result.Metrics.AwayShots, result.Metrics.HomeShotsOnTarget,
			result.Metrics.AwayShotsOnTarget, result.Metrics.HomePossession, result.Metrics.HomePressure,
			result.Metrics.AwayPressure, result.Metrics.HomeTerritory, matchResult(result),
		); err != nil {
			return err
		}
	}
	if err := matches.Send(); err != nil {
		return err
	}

	players := playerIndex(payload.Players)
	events, err := conn.PrepareBatch(ctx, `INSERT INTO touchline_match_events (
		event_key, match_id, season, round, minute, event_type, team_id, player_id, player_name, text, xg
	)`)
	if err != nil {
		return err
	}
	for _, result := range payload.Results {
		for index, event := range result.Events {
			xg := float64(0)
			if event.XG != nil {
				xg = *event.XG
			}
			if err := events.Append(
				fmt.Sprintf("%s:event:%d", result.FixtureID, index), result.FixtureID, payload.Season, result.Round,
				event.Minute, event.Type, event.TeamID, players[event.TeamID+"\x00"+event.PlayerName], event.PlayerName,
				event.Text, xg,
			); err != nil {
				return err
			}
		}
	}
	if err := events.Send(); err != nil {
		return err
	}

	frames, err := conn.PrepareBatch(ctx, `INSERT INTO touchline_player_frames (
		frame_key, match_id, season, round, frame_index, minute, phase, possessing_team_id, ball_x, ball_y,
		player_id, team_id, player_name, position, player_x, player_y, target_x, target_y, intent
	)`)
	if err != nil {
		return err
	}
	for _, result := range payload.Results {
		for frameIndex, frame := range result.Trace {
			for _, player := range frame.Players {
				if err := frames.Append(
					fmt.Sprintf("%s:frame:%d:%s", result.FixtureID, frameIndex, player.ID), result.FixtureID,
					payload.Season, result.Round, frameIndex, frame.Minute, frame.Phase, frame.PossessingTeamID,
					frame.Ball.X, frame.Ball.Y, player.ID, player.TeamID, player.Name, player.Position, player.X,
					player.Y, player.TargetX, player.TargetY, player.Intent,
				); err != nil {
					return err
				}
			}
		}
	}
	if err := frames.Send(); err != nil {
		return err
	}

	ratings, err := conn.PrepareBatch(ctx, `INSERT INTO touchline_player_ratings (
		rating_key, match_id, season, round, player_id, club_id, player_name, rating, opponent_id
	)`)
	if err != nil {
		return err
	}
	for _, result := range payload.Results {
		for playerID, rating := range result.PlayerRatings {
			player, ok := playerByID(payload.Players, playerID)
			if !ok {
				continue
			}
			opponentID := result.AwayID
			if player.ClubID == result.AwayID {
				opponentID = result.HomeID
			}
			if err := ratings.Append(
				fmt.Sprintf("%s:rating:%s", result.FixtureID, playerID), result.FixtureID, payload.Season,
				result.Round, playerID, player.ClubID, player.Name, rating, opponentID,
			); err != nil {
				return err
			}
		}
	}
	return ratings.Send()
}

func (w *icebergWriter) ready(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.ensureCatalog(ctx)
}

func (w *icebergWriter) ensureCatalog(ctx context.Context) error {
	if w.catalog != nil {
		return nil
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(env("AWS_REGION", "us-east-1")))
	if err != nil {
		return err
	}
	cfg.Credentials = aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(w.username, w.password, ""))
	if w.endpoint != "" {
		_ = os.Setenv("AWS_S3_ENDPOINT", w.endpoint)
	}
	cat, err := rest.NewCatalog(ctx, "rest", w.baseURL, rest.WithAwsConfig(cfg))
	if err != nil {
		return err
	}
	w.catalog = cat
	return nil
}

func (w *icebergWriter) append(ctx context.Context, payload MatchdayPayload) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.ensureCatalog(ctx); err != nil {
		return err
	}
	if err := w.catalog.CreateNamespace(ctx, catalog.ToIdentifier("touchline"), nil); err != nil && !errors.Is(err, catalog.ErrNamespaceAlreadyExists) {
		return err
	}

	appendFns := []struct {
		name   string
		schema *iceberg.Schema
		table  arrow.Table
	}{
		{name: "matches", schema: matchesIcebergSchema(), table: matchesArrowTable(payload)},
		{name: "match_events", schema: eventsIcebergSchema(), table: eventsArrowTable(payload)},
		{name: "player_frames", schema: framesIcebergSchema(), table: framesArrowTable(payload)},
		{name: "player_ratings", schema: ratingsIcebergSchema(), table: ratingsArrowTable(payload)},
	}
	for _, item := range appendFns {
		if item.table == nil || item.table.NumRows() == 0 {
			if item.table != nil {
				item.table.Release()
			}
			continue
		}
		tbl, err := w.ensureTable(ctx, item.name, item.schema)
		if err != nil {
			item.table.Release()
			return err
		}
		_, err = tbl.AppendTable(ctx, item.table, 1024, iceberg.Properties{"source": "touchline"})
		item.table.Release()
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *icebergWriter) ensureTable(ctx context.Context, name string, schema *iceberg.Schema) (*table.Table, error) {
	ident := catalog.ToIdentifier("touchline", name)
	tbl, err := w.catalog.LoadTable(ctx, ident)
	if err == nil {
		return tbl, nil
	}
	if !errors.Is(err, catalog.ErrNoSuchTable) {
		return nil, err
	}
	return w.catalog.CreateTable(ctx, ident, schema)
}

func matchesArrowTable(payload MatchdayPayload) arrow.Table {
	schema := arrowSchema([]arrow.Field{
		stringField("match_id"), intField("season"), intField("round"), stringField("home_id"), stringField("away_id"),
		intField("home_goals"), intField("away_goals"), floatField("home_xg"), floatField("away_xg"),
		intField("home_shots"), intField("away_shots"), intField("home_shots_on_target"), intField("away_shots_on_target"),
		floatField("home_possession"), floatField("home_pressure"), floatField("away_pressure"), floatField("home_territory"),
		stringField("result"),
	})
	builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	for _, result := range payload.Results {
		builder.Field(0).(*array.StringBuilder).Append(result.FixtureID)
		builder.Field(1).(*array.Int64Builder).Append(int64(payload.Season))
		builder.Field(2).(*array.Int64Builder).Append(int64(result.Round))
		builder.Field(3).(*array.StringBuilder).Append(result.HomeID)
		builder.Field(4).(*array.StringBuilder).Append(result.AwayID)
		builder.Field(5).(*array.Int64Builder).Append(int64(result.HomeGoals))
		builder.Field(6).(*array.Int64Builder).Append(int64(result.AwayGoals))
		builder.Field(7).(*array.Float64Builder).Append(result.Metrics.HomeXG)
		builder.Field(8).(*array.Float64Builder).Append(result.Metrics.AwayXG)
		builder.Field(9).(*array.Int64Builder).Append(int64(result.Metrics.HomeShots))
		builder.Field(10).(*array.Int64Builder).Append(int64(result.Metrics.AwayShots))
		builder.Field(11).(*array.Int64Builder).Append(int64(result.Metrics.HomeShotsOnTarget))
		builder.Field(12).(*array.Int64Builder).Append(int64(result.Metrics.AwayShotsOnTarget))
		builder.Field(13).(*array.Float64Builder).Append(result.Metrics.HomePossession)
		builder.Field(14).(*array.Float64Builder).Append(result.Metrics.HomePressure)
		builder.Field(15).(*array.Float64Builder).Append(result.Metrics.AwayPressure)
		builder.Field(16).(*array.Float64Builder).Append(result.Metrics.HomeTerritory)
		builder.Field(17).(*array.StringBuilder).Append(matchResult(result))
	}
	return finishArrowTable(schema, builder)
}

func eventsArrowTable(payload MatchdayPayload) arrow.Table {
	schema := arrowSchema([]arrow.Field{
		stringField("event_key"), stringField("match_id"), intField("season"), intField("round"), intField("minute"),
		stringField("event_type"), stringField("team_id"), stringField("player_id"), stringField("player_name"),
		stringField("text"), floatField("xg"),
	})
	builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	players := playerIndex(payload.Players)
	for _, result := range payload.Results {
		for index, event := range result.Events {
			xg := float64(0)
			if event.XG != nil {
				xg = *event.XG
			}
			builder.Field(0).(*array.StringBuilder).Append(fmt.Sprintf("%s:event:%d", result.FixtureID, index))
			builder.Field(1).(*array.StringBuilder).Append(result.FixtureID)
			builder.Field(2).(*array.Int64Builder).Append(int64(payload.Season))
			builder.Field(3).(*array.Int64Builder).Append(int64(result.Round))
			builder.Field(4).(*array.Int64Builder).Append(int64(event.Minute))
			builder.Field(5).(*array.StringBuilder).Append(event.Type)
			builder.Field(6).(*array.StringBuilder).Append(event.TeamID)
			builder.Field(7).(*array.StringBuilder).Append(players[event.TeamID+"\x00"+event.PlayerName])
			builder.Field(8).(*array.StringBuilder).Append(event.PlayerName)
			builder.Field(9).(*array.StringBuilder).Append(event.Text)
			builder.Field(10).(*array.Float64Builder).Append(xg)
		}
	}
	return finishArrowTable(schema, builder)
}

func framesArrowTable(payload MatchdayPayload) arrow.Table {
	schema := arrowSchema([]arrow.Field{
		stringField("frame_key"), stringField("match_id"), intField("season"), intField("round"), intField("frame_index"),
		floatField("minute"), stringField("phase"), stringField("possessing_team_id"), floatField("ball_x"), floatField("ball_y"),
		stringField("player_id"), stringField("team_id"), stringField("player_name"), stringField("position"), floatField("player_x"),
		floatField("player_y"), floatField("target_x"), floatField("target_y"), stringField("intent"),
	})
	builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	for _, result := range payload.Results {
		for frameIndex, frame := range result.Trace {
			for _, player := range frame.Players {
				builder.Field(0).(*array.StringBuilder).Append(fmt.Sprintf("%s:frame:%d:%s", result.FixtureID, frameIndex, player.ID))
				builder.Field(1).(*array.StringBuilder).Append(result.FixtureID)
				builder.Field(2).(*array.Int64Builder).Append(int64(payload.Season))
				builder.Field(3).(*array.Int64Builder).Append(int64(result.Round))
				builder.Field(4).(*array.Int64Builder).Append(int64(frameIndex))
				builder.Field(5).(*array.Float64Builder).Append(frame.Minute)
				builder.Field(6).(*array.StringBuilder).Append(frame.Phase)
				builder.Field(7).(*array.StringBuilder).Append(frame.PossessingTeamID)
				builder.Field(8).(*array.Float64Builder).Append(frame.Ball.X)
				builder.Field(9).(*array.Float64Builder).Append(frame.Ball.Y)
				builder.Field(10).(*array.StringBuilder).Append(player.ID)
				builder.Field(11).(*array.StringBuilder).Append(player.TeamID)
				builder.Field(12).(*array.StringBuilder).Append(player.Name)
				builder.Field(13).(*array.StringBuilder).Append(player.Position)
				builder.Field(14).(*array.Float64Builder).Append(player.X)
				builder.Field(15).(*array.Float64Builder).Append(player.Y)
				builder.Field(16).(*array.Float64Builder).Append(player.TargetX)
				builder.Field(17).(*array.Float64Builder).Append(player.TargetY)
				builder.Field(18).(*array.StringBuilder).Append(player.Intent)
			}
		}
	}
	return finishArrowTable(schema, builder)
}

func ratingsArrowTable(payload MatchdayPayload) arrow.Table {
	schema := arrowSchema([]arrow.Field{
		stringField("rating_key"), stringField("match_id"), intField("season"), intField("round"), stringField("player_id"),
		stringField("club_id"), stringField("player_name"), floatField("rating"), stringField("opponent_id"),
	})
	builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	for _, result := range payload.Results {
		for playerID, rating := range result.PlayerRatings {
			player, ok := playerByID(payload.Players, playerID)
			if !ok {
				continue
			}
			opponentID := result.AwayID
			if player.ClubID == result.AwayID {
				opponentID = result.HomeID
			}
			builder.Field(0).(*array.StringBuilder).Append(fmt.Sprintf("%s:rating:%s", result.FixtureID, playerID))
			builder.Field(1).(*array.StringBuilder).Append(result.FixtureID)
			builder.Field(2).(*array.Int64Builder).Append(int64(payload.Season))
			builder.Field(3).(*array.Int64Builder).Append(int64(result.Round))
			builder.Field(4).(*array.StringBuilder).Append(playerID)
			builder.Field(5).(*array.StringBuilder).Append(player.ClubID)
			builder.Field(6).(*array.StringBuilder).Append(player.Name)
			builder.Field(7).(*array.Float64Builder).Append(rating)
			builder.Field(8).(*array.StringBuilder).Append(opponentID)
		}
	}
	return finishArrowTable(schema, builder)
}

func finishArrowTable(schema *arrow.Schema, builder *array.RecordBuilder) arrow.Table {
	record := builder.NewRecordBatch()
	builder.Release()
	tbl := array.NewTableFromRecords(schema, []arrow.RecordBatch{record})
	record.Release()
	return tbl
}

func matchesIcebergSchema() *iceberg.Schema {
	return icebergSchema([]iceberg.NestedField{
		{ID: 1, Name: "match_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 2, Name: "season", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 3, Name: "round", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 4, Name: "home_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 5, Name: "away_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 6, Name: "home_goals", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 7, Name: "away_goals", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 8, Name: "home_xg", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 9, Name: "away_xg", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 10, Name: "home_shots", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 11, Name: "away_shots", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 12, Name: "home_shots_on_target", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 13, Name: "away_shots_on_target", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 14, Name: "home_possession", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 15, Name: "home_pressure", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 16, Name: "away_pressure", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 17, Name: "home_territory", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 18, Name: "result", Type: iceberg.PrimitiveTypes.String, Required: true},
	})
}

func eventsIcebergSchema() *iceberg.Schema {
	return icebergSchema([]iceberg.NestedField{
		{ID: 1, Name: "event_key", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 2, Name: "match_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 3, Name: "season", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 4, Name: "round", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 5, Name: "minute", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 6, Name: "event_type", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 7, Name: "team_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 8, Name: "player_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 9, Name: "player_name", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 10, Name: "text", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 11, Name: "xg", Type: iceberg.PrimitiveTypes.Float64, Required: true},
	})
}

func framesIcebergSchema() *iceberg.Schema {
	return icebergSchema([]iceberg.NestedField{
		{ID: 1, Name: "frame_key", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 2, Name: "match_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 3, Name: "season", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 4, Name: "round", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 5, Name: "frame_index", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 6, Name: "minute", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 7, Name: "phase", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 8, Name: "possessing_team_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 9, Name: "ball_x", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 10, Name: "ball_y", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 11, Name: "player_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 12, Name: "team_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 13, Name: "player_name", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 14, Name: "position", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 15, Name: "player_x", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 16, Name: "player_y", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 17, Name: "target_x", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 18, Name: "target_y", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 19, Name: "intent", Type: iceberg.PrimitiveTypes.String, Required: true},
	})
}

func ratingsIcebergSchema() *iceberg.Schema {
	return icebergSchema([]iceberg.NestedField{
		{ID: 1, Name: "rating_key", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 2, Name: "match_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 3, Name: "season", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 4, Name: "round", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 5, Name: "player_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 6, Name: "club_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 7, Name: "player_name", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 8, Name: "rating", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 9, Name: "opponent_id", Type: iceberg.PrimitiveTypes.String, Required: true},
	})
}

func icebergSchema(fields []iceberg.NestedField) *iceberg.Schema {
	return iceberg.NewSchema(1, fields...)
}

func arrowSchema(fields []arrow.Field) *arrow.Schema {
	return arrow.NewSchema(fields, nil)
}

func stringField(name string) arrow.Field {
	return arrow.Field{Name: name, Type: arrow.BinaryTypes.String, Nullable: false}
}

func intField(name string) arrow.Field {
	return arrow.Field{Name: name, Type: arrow.PrimitiveTypes.Int64, Nullable: false}
}

func floatField(name string) arrow.Field {
	return arrow.Field{Name: name, Type: arrow.PrimitiveTypes.Float64, Nullable: false}
}

func playerIndex(players []Player) map[string]string {
	index := make(map[string]string, len(players))
	for _, player := range players {
		index[player.ClubID+"\x00"+player.Name] = player.ID
	}
	return index
}

func playerByID(players []Player, id string) (Player, bool) {
	for _, player := range players {
		if player.ID == id {
			return player, true
		}
	}
	return Player{}, false
}

func matchResult(result MatchResult) string {
	if result.HomeGoals > result.AwayGoals {
		return "home_win"
	}
	if result.HomeGoals < result.AwayGoals {
		return "away_win"
	}
	return "draw"
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
