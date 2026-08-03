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
	_ "github.com/apache/iceberg-go/io/gocloud"
	"github.com/apache/iceberg-go/table"
	aws "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type MatchdayPayload struct {
	RunID           string             `json:"runId"`
	Season          int                `json:"season"`
	Round           int                `json:"round"`
	Clubs           []Club             `json:"clubs"`
	Players         []Player           `json:"players"`
	Standings       []StandingSnapshot `json:"standings"`
	PlayerSnapshots []PlayerSnapshot   `json:"playerSnapshots"`
	Results         []MatchResult      `json:"results"`
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

type StandingSnapshot struct {
	ClubID         string `json:"clubId"`
	Rank           int    `json:"rank"`
	Played         int    `json:"played"`
	Won            int    `json:"won"`
	Drawn          int    `json:"drawn"`
	Lost           int    `json:"lost"`
	GoalsFor       int    `json:"goalsFor"`
	GoalsAgainst   int    `json:"goalsAgainst"`
	GoalDifference int    `json:"goalDifference"`
	Points         int    `json:"points"`
	Form           string `json:"form"`
}

type PlayerSnapshot struct {
	PlayerID      string  `json:"playerId"`
	ClubID        string  `json:"clubId"`
	PlayerName    string  `json:"playerName"`
	Position      string  `json:"position"`
	Age           int     `json:"age"`
	Overall       int     `json:"overall"`
	Potential     int     `json:"potential"`
	Form          int     `json:"form"`
	Morale        int     `json:"morale"`
	Fitness       int     `json:"fitness"`
	Value         float64 `json:"value"`
	AverageRating float64 `json:"averageRating"`
	Appeared      bool    `json:"appeared"`
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

type AnalyticsTimelinePoint struct {
	Round          int32   `json:"round"`
	Rank           int32   `json:"rank"`
	Played         int32   `json:"played"`
	Won            int32   `json:"won"`
	Drawn          int32   `json:"drawn"`
	Lost           int32   `json:"lost"`
	GoalsFor       int32   `json:"goalsFor"`
	GoalsAgainst   int32   `json:"goalsAgainst"`
	GoalDifference int32   `json:"goalDifference"`
	Points         int32   `json:"points"`
	Form           string  `json:"form"`
	XGFor          float64 `json:"xgFor"`
	XGAgainst      float64 `json:"xgAgainst"`
	Possession     float64 `json:"possession"`
	ShotsFor       int32   `json:"shotsFor"`
	ShotsAgainst   int32   `json:"shotsAgainst"`
	Pressure       float64 `json:"pressure"`
	Territory      float64 `json:"territory"`
	Events         uint64  `json:"events"`
	Frames         uint64  `json:"frames"`
}

type AnalyticsTimelineStanding struct {
	Round          int32  `json:"round"`
	ClubID         string `json:"clubId"`
	Rank           int32  `json:"rank"`
	Played         int32  `json:"played"`
	Won            int32  `json:"won"`
	Drawn          int32  `json:"drawn"`
	Lost           int32  `json:"lost"`
	GoalsFor       int32  `json:"goalsFor"`
	GoalsAgainst   int32  `json:"goalsAgainst"`
	GoalDifference int32  `json:"goalDifference"`
	Points         int32  `json:"points"`
	Form           string `json:"form"`
}

type AnalyticsDevelopmentPlayer struct {
	PlayerID       string `json:"playerId"`
	PlayerName     string `json:"playerName"`
	Position       string `json:"position"`
	OpeningOverall int32  `json:"openingOverall"`
	Overall        int32  `json:"overall"`
	Potential      int32  `json:"potential"`
	Form           int32  `json:"form"`
	Fitness        int32  `json:"fitness"`
	Change         int32  `json:"change"`
}

type AnalyticsTimeline struct {
	Points  []AnalyticsTimelinePoint     `json:"points"`
	Table   []AnalyticsTimelineStanding  `json:"table"`
	Players []AnalyticsDevelopmentPlayer `json:"players"`
	Source  string                       `json:"source"`
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
	mux.HandleFunc("/api/analytics/timeline", store.handleTimeline)

	server := &http.Server{
		Addr:              env("ANALYTICS_ADDR", ":8787"),
		Handler:           cors(mux, env("ANALYTICS_ALLOWED_ORIGIN", "http://localhost:5173")),
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
	if err := validatePayload(payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
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

	summary, err := s.summary(r.Context(), clubID, r.URL.Query().Get("run_id"))
	if err != nil {
		log.Printf("summary failed: %v", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "analytics store unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Store) handleTimeline(w http.ResponseWriter, r *http.Request) {
	clubID := r.URL.Query().Get("club_id")
	runID := r.URL.Query().Get("run_id")
	if clubID == "" || runID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "club_id and run_id are required"})
		return
	}

	timeline, err := s.timeline(r.Context(), clubID, runID)
	if err != nil {
		log.Printf("timeline failed: %v", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "analytics store unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, timeline)
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

	clickHouseResults := make([]MatchResult, 0, len(payload.Results))
	icebergResults := make([]MatchResult, 0, len(payload.Results))
	for _, result := range payload.Results {
		ingested, err := s.hasIngestLog(ctx, payload.RunID, result.FixtureID)
		if err != nil {
			return err
		}
		if ingested {
			continue
		}
		icebergResults = append(icebergResults, result)
		complete, err := s.hasCompleteMatch(ctx, payload.RunID, result)
		if err != nil {
			return err
		}
		if !complete {
			clickHouseResults = append(clickHouseResults, result)
		}
	}
	if len(icebergResults) == 0 {
		return nil
	}

	if len(clickHouseResults) > 0 {
		clickHousePayload := payload
		clickHousePayload.Results = clickHouseResults
		for _, result := range clickHouseResults {
			if err := clearClickHouseMatch(ctx, s.clickhouse, payload.RunID, result.FixtureID); err != nil {
				return err
			}
		}
		if err := insertClickHouse(ctx, s.clickhouse, clickHousePayload); err != nil {
			return err
		}
	}
	icebergPayload := payload
	icebergPayload.Results = icebergResults
	if err := s.iceberg.append(ctx, icebergPayload); err != nil {
		return err
	}
	if err := s.markIngested(ctx, payload.RunID, icebergResults); err != nil {
		return err
	}
	return nil
}

func (s *Store) hasIngestLog(ctx context.Context, runID string, matchID string) (bool, error) {
	var count uint64
	if err := s.clickhouse.QueryRow(ctx, "SELECT count() FROM touchline_ingest_log WHERE run_id = ? AND match_id = ?", runID, matchID).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) hasCompleteMatch(ctx context.Context, runID string, result MatchResult) (bool, error) {
	var matches, events, frames, ratings uint64
	if err := s.clickhouse.QueryRow(ctx, `
		SELECT
			(SELECT count() FROM touchline_matches_v2 WHERE run_id = ? AND match_id = ?),
			(SELECT count() FROM touchline_match_events_v2 WHERE run_id = ? AND match_id = ?),
			(SELECT count() FROM touchline_player_frames_v2 WHERE run_id = ? AND match_id = ?),
			(SELECT count() FROM touchline_player_ratings_v2 WHERE run_id = ? AND match_id = ?)`,
		runID, result.FixtureID, runID, result.FixtureID, runID, result.FixtureID, runID, result.FixtureID,
	).Scan(&matches, &events, &frames, &ratings); err != nil {
		return false, err
	}
	expectedFrames := 0
	for _, frame := range result.Trace {
		expectedFrames += len(frame.Players)
	}
	return matches == 1 && events == uint64(len(result.Events)) && frames == uint64(expectedFrames) && ratings == uint64(len(result.PlayerRatings)), nil
}

func clearClickHouseMatch(ctx context.Context, conn driver.Conn, runID string, matchID string) error {
	for _, tableName := range []string{"touchline_matches_v2", "touchline_match_events_v2", "touchline_player_frames_v2", "touchline_player_ratings_v2"} {
		statement := fmt.Sprintf("ALTER TABLE %s DELETE WHERE run_id = ? AND match_id = ? SETTINGS mutations_sync = 2", tableName)
		if err := conn.Exec(ctx, statement, runID, matchID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) markIngested(ctx context.Context, runID string, results []MatchResult) error {
	logBatch, err := s.clickhouse.PrepareBatch(ctx, `INSERT INTO touchline_ingest_log (run_id, match_id)`)
	if err != nil {
		return err
	}
	for _, result := range results {
		if err := logBatch.Append(runID, result.FixtureID); err != nil {
			return err
		}
	}
	return logBatch.Send()
}

func (s *Store) summary(ctx context.Context, clubID string, runID string) (AnalyticsSummary, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := ensureClickHouseSchema(ctx, s.clickhouse); err != nil {
		return AnalyticsSummary{}, err
	}

	var summary AnalyticsSummary
	summary.Players = []AnalyticsPlayer{}
	if err := s.clickhouse.QueryRow(ctx, `
		SELECT count(), if(count() = 0, 0, avg(if(home_id = ?, home_xg, away_xg))), if(count() = 0, 0, avg(if(home_id = ?, home_possession, 100 - home_possession)))
		FROM touchline_matches_v2
		WHERE (home_id = ? OR away_id = ?) AND (? = '' OR run_id = ?)`, clubID, clubID, clubID, clubID, runID, runID).Scan(&summary.Matches, &summary.AverageXG, &summary.AveragePossession); err != nil {
		return AnalyticsSummary{}, err
	}
	if err := s.clickhouse.QueryRow(ctx, `
		SELECT count()
		FROM touchline_match_events_v2
		WHERE (? = '' OR run_id = ?)
		  AND match_id IN (SELECT match_id FROM touchline_matches_v2 WHERE (home_id = ? OR away_id = ?) AND (? = '' OR run_id = ?))`, runID, runID, clubID, clubID, runID, runID).Scan(&summary.Events); err != nil {
		return AnalyticsSummary{}, err
	}
	if err := s.clickhouse.QueryRow(ctx, `
		SELECT uniqExact(tuple(run_id, match_id, frame_index))
		FROM touchline_player_frames_v2
		WHERE (? = '' OR run_id = ?)
		  AND match_id IN (SELECT match_id FROM touchline_matches_v2 WHERE (home_id = ? OR away_id = ?) AND (? = '' OR run_id = ?))`, runID, runID, clubID, clubID, runID, runID).Scan(&summary.Frames); err != nil {
		return AnalyticsSummary{}, err
	}

	rows, err := s.clickhouse.Query(ctx, `
		SELECT player_id, any(player_name), count(), avg(rating)
		FROM touchline_player_ratings_v2
		WHERE club_id = ? AND (? = '' OR run_id = ?)
		GROUP BY player_id
		ORDER BY avg(rating) DESC
		LIMIT 8`, clubID, runID, runID)
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

func (s *Store) timeline(ctx context.Context, clubID string, runID string) (AnalyticsTimeline, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := ensureClickHouseSchema(ctx, s.clickhouse); err != nil {
		return AnalyticsTimeline{}, err
	}

	timeline := AnalyticsTimeline{
		Points:  []AnalyticsTimelinePoint{},
		Table:   []AnalyticsTimelineStanding{},
		Players: []AnalyticsDevelopmentPlayer{},
		Source:  "ClickHouse + Iceberg",
	}
	pointRows, err := s.clickhouse.Query(ctx, `
		WITH event_counts AS
		(
			SELECT e.run_id, e.season, e.round, count() AS events
			FROM touchline_match_events_v2 AS e
			INNER JOIN touchline_matches_v2 AS m
				ON e.run_id = m.run_id AND e.season = m.season AND e.match_id = m.match_id
			WHERE e.run_id = ? AND (m.home_id = ? OR m.away_id = ?)
			GROUP BY e.run_id, e.season, e.round
		), frame_counts AS
		(
			SELECT f.run_id, f.season, f.round, uniqExact(tuple(f.run_id, f.match_id, f.frame_index)) AS frames
			FROM touchline_player_frames_v2 AS f
			INNER JOIN touchline_matches_v2 AS m
				ON f.run_id = m.run_id AND f.season = m.season AND f.match_id = m.match_id
			WHERE f.run_id = ? AND (m.home_id = ? OR m.away_id = ?)
			GROUP BY f.run_id, f.season, f.round
		)
		SELECT
			toInt32(s.round), toInt32(s.rank), toInt32(s.played), toInt32(s.won), toInt32(s.drawn), toInt32(s.lost),
			toInt32(s.goals_for), toInt32(s.goals_against), toInt32(s.goal_difference), toInt32(s.points), s.form,
			round(coalesce(any(if(m.home_id = s.club_id, toFloat64(m.home_xg), toFloat64(m.away_xg))), 0.0), 2),
			round(coalesce(any(if(m.home_id = s.club_id, toFloat64(m.away_xg), toFloat64(m.home_xg))), 0.0), 2),
			round(coalesce(any(if(m.home_id = s.club_id, toFloat64(m.home_possession), 100.0 - toFloat64(m.home_possession))), 0.0), 1),
			toInt32(coalesce(any(if(m.home_id = s.club_id, m.home_shots, m.away_shots)), 0)),
			toInt32(coalesce(any(if(m.home_id = s.club_id, m.away_shots, m.home_shots)), 0)),
			round(coalesce(any(if(m.home_id = s.club_id, toFloat64(m.home_pressure), toFloat64(m.away_pressure))), 0.0), 1),
			round(coalesce(any(if(m.home_id = s.club_id, toFloat64(m.home_territory), 100.0 - toFloat64(m.home_territory))), 0.0), 1),
			coalesce(event_counts.events, toUInt64(0)), coalesce(frame_counts.frames, toUInt64(0))
		FROM (SELECT * FROM touchline_standings FINAL) AS s
		LEFT JOIN touchline_matches_v2 AS m
			ON m.run_id = s.run_id AND m.season = s.season AND m.round = s.round
			AND (m.home_id = s.club_id OR m.away_id = s.club_id)
		LEFT JOIN event_counts
			ON event_counts.run_id = s.run_id AND event_counts.season = s.season AND event_counts.round = s.round
		LEFT JOIN frame_counts
			ON frame_counts.run_id = s.run_id AND frame_counts.season = s.season AND frame_counts.round = s.round
		WHERE s.run_id = ? AND s.club_id = ?
		GROUP BY
			s.round, s.rank, s.played, s.won, s.drawn, s.lost, s.goals_for, s.goals_against,
			s.goal_difference, s.points, s.form, event_counts.events, frame_counts.frames
		ORDER BY s.round`, runID, clubID, clubID, runID, clubID, clubID, runID, clubID)
	if err != nil {
		return AnalyticsTimeline{}, err
	}
	for pointRows.Next() {
		var point AnalyticsTimelinePoint
		if err := pointRows.Scan(
			&point.Round, &point.Rank, &point.Played, &point.Won, &point.Drawn, &point.Lost,
			&point.GoalsFor, &point.GoalsAgainst, &point.GoalDifference, &point.Points, &point.Form,
			&point.XGFor, &point.XGAgainst, &point.Possession, &point.ShotsFor, &point.ShotsAgainst,
			&point.Pressure, &point.Territory, &point.Events, &point.Frames,
		); err != nil {
			pointRows.Close()
			return AnalyticsTimeline{}, err
		}
		timeline.Points = append(timeline.Points, point)
	}
	if err := pointRows.Err(); err != nil {
		pointRows.Close()
		return AnalyticsTimeline{}, err
	}
	pointRows.Close()

	tableRows, err := s.clickhouse.Query(ctx, `
		SELECT toInt32(round), club_id, toInt32(rank), toInt32(played), toInt32(won), toInt32(drawn), toInt32(lost),
			toInt32(goals_for), toInt32(goals_against), toInt32(goal_difference), toInt32(points), form
		FROM touchline_standings FINAL
		WHERE run_id = ?
		ORDER BY round, rank`, runID)
	if err != nil {
		return AnalyticsTimeline{}, err
	}
	for tableRows.Next() {
		var standing AnalyticsTimelineStanding
		if err := tableRows.Scan(
			&standing.Round, &standing.ClubID, &standing.Rank, &standing.Played, &standing.Won, &standing.Drawn,
			&standing.Lost, &standing.GoalsFor, &standing.GoalsAgainst, &standing.GoalDifference, &standing.Points,
			&standing.Form,
		); err != nil {
			tableRows.Close()
			return AnalyticsTimeline{}, err
		}
		timeline.Table = append(timeline.Table, standing)
	}
	if err := tableRows.Err(); err != nil {
		tableRows.Close()
		return AnalyticsTimeline{}, err
	}
	tableRows.Close()

	playerRows, err := s.clickhouse.Query(ctx, `
		SELECT
			player_id, any(player_name), any(position), toInt32(argMin(overall, round)), toInt32(argMax(overall, round)),
			toInt32(argMax(potential, round)), toInt32(argMax(form, round)), toInt32(argMax(fitness, round)),
			toInt32(argMax(overall, round) - argMin(overall, round))
		FROM touchline_player_snapshots FINAL
		WHERE run_id = ? AND club_id = ?
		GROUP BY player_id
		ORDER BY argMax(overall, round) - argMin(overall, round) DESC, argMax(overall, round) DESC
		LIMIT 8`, runID, clubID)
	if err != nil {
		return AnalyticsTimeline{}, err
	}
	for playerRows.Next() {
		var player AnalyticsDevelopmentPlayer
		if err := playerRows.Scan(
			&player.PlayerID, &player.PlayerName, &player.Position, &player.OpeningOverall, &player.Overall,
			&player.Potential, &player.Form, &player.Fitness, &player.Change,
		); err != nil {
			playerRows.Close()
			return AnalyticsTimeline{}, err
		}
		timeline.Players = append(timeline.Players, player)
	}
	if err := playerRows.Err(); err != nil {
		playerRows.Close()
		return AnalyticsTimeline{}, err
	}
	playerRows.Close()

	return timeline, nil
}

func ensureClickHouseSchema(ctx context.Context, conn driver.Conn) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS touchline_clubs_v2 (
			run_id String,
			season UInt32,
			club_id String,
			name String,
			short_name String,
			ingested_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = ReplacingMergeTree(ingested_at) ORDER BY (run_id, season, club_id)`,
		`CREATE TABLE IF NOT EXISTS touchline_matches_v2 (
			run_id String,
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
		) ENGINE = MergeTree ORDER BY (run_id, season, round, match_id)`,
		`CREATE TABLE IF NOT EXISTS touchline_match_events_v2 (
			run_id String,
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
		) ENGINE = MergeTree ORDER BY (run_id, season, match_id, minute, event_key)`,
		`CREATE TABLE IF NOT EXISTS touchline_player_frames_v2 (
			run_id String,
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
		) ENGINE = MergeTree ORDER BY (run_id, season, match_id, frame_index, player_id)`,
		`CREATE TABLE IF NOT EXISTS touchline_player_ratings_v2 (
			run_id String,
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
		) ENGINE = MergeTree ORDER BY (run_id, season, player_id, match_id)`,
		`CREATE TABLE IF NOT EXISTS touchline_standings (
			run_id String,
			season UInt32,
			round UInt32,
			club_id String,
			rank UInt32,
			played UInt32,
			won UInt32,
			drawn UInt32,
			lost UInt32,
			goals_for Int32,
			goals_against Int32,
			goal_difference Int32,
			points UInt32,
			form String,
			ingested_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = ReplacingMergeTree(ingested_at) ORDER BY (run_id, season, round, club_id)`,
		`CREATE TABLE IF NOT EXISTS touchline_player_snapshots (
			run_id String,
			season UInt32,
			round UInt32,
			player_id String,
			club_id String,
			player_name String,
			position LowCardinality(String),
			age UInt32,
			overall UInt32,
			potential UInt32,
			form UInt32,
			morale UInt32,
			fitness UInt32,
			value Float32,
			average_rating Float32,
			appeared Bool,
			ingested_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = ReplacingMergeTree(ingested_at) ORDER BY (run_id, season, round, player_id)`,
		`CREATE TABLE IF NOT EXISTS touchline_ingest_log (
			run_id String,
			match_id String,
			ingested_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = ReplacingMergeTree(ingested_at) ORDER BY (run_id, match_id)`,
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
		clubs, err := conn.PrepareBatch(ctx, `INSERT INTO touchline_clubs_v2 (
			run_id, season, club_id, name, short_name
		)`)
		if err != nil {
			return err
		}
		for _, club := range payload.Clubs {
			if err := clubs.Append(payload.RunID, payload.Season, club.ID, club.Name, club.ShortName); err != nil {
				return err
			}
		}
		if err := clubs.Send(); err != nil {
			return err
		}
	}

	standings, err := conn.PrepareBatch(ctx, `INSERT INTO touchline_standings (
		run_id, season, round, club_id, rank, played, won, drawn, lost, goals_for, goals_against,
		goal_difference, points, form
	)`)
	if err != nil {
		return err
	}
	for _, standing := range payload.Standings {
		if err := standings.Append(
			payload.RunID, payload.Season, payload.Round, standing.ClubID, standing.Rank, standing.Played,
			standing.Won, standing.Drawn, standing.Lost, standing.GoalsFor, standing.GoalsAgainst,
			standing.GoalDifference, standing.Points, standing.Form,
		); err != nil {
			return err
		}
	}
	if err := standings.Send(); err != nil {
		return err
	}

	playerSnapshots, err := conn.PrepareBatch(ctx, `INSERT INTO touchline_player_snapshots (
		run_id, season, round, player_id, club_id, player_name, position, age, overall, potential,
		form, morale, fitness, value, average_rating, appeared
	)`)
	if err != nil {
		return err
	}
	for _, snapshot := range payload.PlayerSnapshots {
		if err := playerSnapshots.Append(
			payload.RunID, payload.Season, payload.Round, snapshot.PlayerID, snapshot.ClubID, snapshot.PlayerName,
			snapshot.Position, snapshot.Age, snapshot.Overall, snapshot.Potential, snapshot.Form, snapshot.Morale,
			snapshot.Fitness, snapshot.Value, snapshot.AverageRating, snapshot.Appeared,
		); err != nil {
			return err
		}
	}
	if err := playerSnapshots.Send(); err != nil {
		return err
	}

	matches, err := conn.PrepareBatch(ctx, `INSERT INTO touchline_matches_v2 (
		run_id, match_id, season, round, home_id, away_id, home_goals, away_goals, home_xg, away_xg,
		home_shots, away_shots, home_shots_on_target, away_shots_on_target, home_possession,
		home_pressure, away_pressure, home_territory, result
	)`)
	if err != nil {
		return err
	}
	for _, result := range payload.Results {
		if err := matches.Append(
			payload.RunID, result.FixtureID, payload.Season, result.Round, result.HomeID, result.AwayID,
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
	events, err := conn.PrepareBatch(ctx, `INSERT INTO touchline_match_events_v2 (
		run_id, event_key, match_id, season, round, minute, event_type, team_id, player_id, player_name, text, xg
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
				payload.RunID, fmt.Sprintf("%s:event:%d", result.FixtureID, index), result.FixtureID, payload.Season, result.Round,
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

	frames, err := conn.PrepareBatch(ctx, `INSERT INTO touchline_player_frames_v2 (
		run_id, frame_key, match_id, season, round, frame_index, minute, phase, possessing_team_id, ball_x, ball_y,
		player_id, team_id, player_name, position, player_x, player_y, target_x, target_y, intent
	)`)
	if err != nil {
		return err
	}
	for _, result := range payload.Results {
		for frameIndex, frame := range result.Trace {
			for _, player := range frame.Players {
				if err := frames.Append(
					payload.RunID, fmt.Sprintf("%s:frame:%d:%s", result.FixtureID, frameIndex, player.ID), result.FixtureID,
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

	ratings, err := conn.PrepareBatch(ctx, `INSERT INTO touchline_player_ratings_v2 (
		run_id, rating_key, match_id, season, round, player_id, club_id, player_name, rating, opponent_id
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
				payload.RunID, fmt.Sprintf("%s:rating:%s", result.FixtureID, playerID), result.FixtureID, payload.Season,
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
		{name: "standings", schema: standingsIcebergSchema(), table: standingsArrowTable(payload)},
		{name: "player_snapshots", schema: playerSnapshotsIcebergSchema(), table: playerSnapshotsArrowTable(payload)},
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
		_, err = tbl.OverwriteTable(
			ctx,
			item.table,
			1024,
			iceberg.Properties{"source": "touchline"},
			table.WithOverwriteFilter(iceberg.NewAnd(
				iceberg.EqualTo(iceberg.Reference("run_id"), payload.RunID),
				iceberg.EqualTo(iceberg.Reference("season"), int64(payload.Season)),
				iceberg.EqualTo(iceberg.Reference("round"), int64(payload.Round)),
			)),
		)
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
		if _, ok := tbl.Schema().FindFieldByName("run_id"); ok {
			return tbl, nil
		}
		txn := tbl.NewTransaction()
		if err := txn.UpdateSchema(true, true).AddColumn([]string{"run_id"}, iceberg.PrimitiveTypes.String, "", false, iceberg.StringLiteral("")).Commit(); err != nil {
			return nil, err
		}
		return txn.Commit(ctx)
	}
	if !errors.Is(err, catalog.ErrNoSuchTable) {
		return nil, err
	}
	return w.catalog.CreateTable(ctx, ident, schema)
}

func matchesArrowTable(payload MatchdayPayload) arrow.Table {
	schema := arrowSchema([]arrow.Field{
		stringField("run_id"), stringField("match_id"), intField("season"), intField("round"), stringField("home_id"), stringField("away_id"),
		intField("home_goals"), intField("away_goals"), floatField("home_xg"), floatField("away_xg"),
		intField("home_shots"), intField("away_shots"), intField("home_shots_on_target"), intField("away_shots_on_target"),
		floatField("home_possession"), floatField("home_pressure"), floatField("away_pressure"), floatField("home_territory"),
		stringField("result"),
	})
	builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	for _, result := range payload.Results {
		builder.Field(0).(*array.StringBuilder).Append(payload.RunID)
		builder.Field(1).(*array.StringBuilder).Append(result.FixtureID)
		builder.Field(2).(*array.Int64Builder).Append(int64(payload.Season))
		builder.Field(3).(*array.Int64Builder).Append(int64(result.Round))
		builder.Field(4).(*array.StringBuilder).Append(result.HomeID)
		builder.Field(5).(*array.StringBuilder).Append(result.AwayID)
		builder.Field(6).(*array.Int64Builder).Append(int64(result.HomeGoals))
		builder.Field(7).(*array.Int64Builder).Append(int64(result.AwayGoals))
		builder.Field(8).(*array.Float64Builder).Append(result.Metrics.HomeXG)
		builder.Field(9).(*array.Float64Builder).Append(result.Metrics.AwayXG)
		builder.Field(10).(*array.Int64Builder).Append(int64(result.Metrics.HomeShots))
		builder.Field(11).(*array.Int64Builder).Append(int64(result.Metrics.AwayShots))
		builder.Field(12).(*array.Int64Builder).Append(int64(result.Metrics.HomeShotsOnTarget))
		builder.Field(13).(*array.Int64Builder).Append(int64(result.Metrics.AwayShotsOnTarget))
		builder.Field(14).(*array.Float64Builder).Append(result.Metrics.HomePossession)
		builder.Field(15).(*array.Float64Builder).Append(result.Metrics.HomePressure)
		builder.Field(16).(*array.Float64Builder).Append(result.Metrics.AwayPressure)
		builder.Field(17).(*array.Float64Builder).Append(result.Metrics.HomeTerritory)
		builder.Field(18).(*array.StringBuilder).Append(matchResult(result))
	}
	return finishArrowTable(schema, builder)
}

func eventsArrowTable(payload MatchdayPayload) arrow.Table {
	schema := arrowSchema([]arrow.Field{
		stringField("run_id"), stringField("event_key"), stringField("match_id"), intField("season"), intField("round"), intField("minute"),
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
			builder.Field(0).(*array.StringBuilder).Append(payload.RunID)
			builder.Field(1).(*array.StringBuilder).Append(fmt.Sprintf("%s:event:%d", result.FixtureID, index))
			builder.Field(2).(*array.StringBuilder).Append(result.FixtureID)
			builder.Field(3).(*array.Int64Builder).Append(int64(payload.Season))
			builder.Field(4).(*array.Int64Builder).Append(int64(result.Round))
			builder.Field(5).(*array.Int64Builder).Append(int64(event.Minute))
			builder.Field(6).(*array.StringBuilder).Append(event.Type)
			builder.Field(7).(*array.StringBuilder).Append(event.TeamID)
			builder.Field(8).(*array.StringBuilder).Append(players[event.TeamID+"\x00"+event.PlayerName])
			builder.Field(9).(*array.StringBuilder).Append(event.PlayerName)
			builder.Field(10).(*array.StringBuilder).Append(event.Text)
			builder.Field(11).(*array.Float64Builder).Append(xg)
		}
	}
	return finishArrowTable(schema, builder)
}

func framesArrowTable(payload MatchdayPayload) arrow.Table {
	schema := arrowSchema([]arrow.Field{
		stringField("run_id"), stringField("frame_key"), stringField("match_id"), intField("season"), intField("round"), intField("frame_index"),
		floatField("minute"), stringField("phase"), stringField("possessing_team_id"), floatField("ball_x"), floatField("ball_y"),
		stringField("player_id"), stringField("team_id"), stringField("player_name"), stringField("position"), floatField("player_x"),
		floatField("player_y"), floatField("target_x"), floatField("target_y"), stringField("intent"),
	})
	builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	for _, result := range payload.Results {
		for frameIndex, frame := range result.Trace {
			for _, player := range frame.Players {
				builder.Field(0).(*array.StringBuilder).Append(payload.RunID)
				builder.Field(1).(*array.StringBuilder).Append(fmt.Sprintf("%s:frame:%d:%s", result.FixtureID, frameIndex, player.ID))
				builder.Field(2).(*array.StringBuilder).Append(result.FixtureID)
				builder.Field(3).(*array.Int64Builder).Append(int64(payload.Season))
				builder.Field(4).(*array.Int64Builder).Append(int64(result.Round))
				builder.Field(5).(*array.Int64Builder).Append(int64(frameIndex))
				builder.Field(6).(*array.Float64Builder).Append(frame.Minute)
				builder.Field(7).(*array.StringBuilder).Append(frame.Phase)
				builder.Field(8).(*array.StringBuilder).Append(frame.PossessingTeamID)
				builder.Field(9).(*array.Float64Builder).Append(frame.Ball.X)
				builder.Field(10).(*array.Float64Builder).Append(frame.Ball.Y)
				builder.Field(11).(*array.StringBuilder).Append(player.ID)
				builder.Field(12).(*array.StringBuilder).Append(player.TeamID)
				builder.Field(13).(*array.StringBuilder).Append(player.Name)
				builder.Field(14).(*array.StringBuilder).Append(player.Position)
				builder.Field(15).(*array.Float64Builder).Append(player.X)
				builder.Field(16).(*array.Float64Builder).Append(player.Y)
				builder.Field(17).(*array.Float64Builder).Append(player.TargetX)
				builder.Field(18).(*array.Float64Builder).Append(player.TargetY)
				builder.Field(19).(*array.StringBuilder).Append(player.Intent)
			}
		}
	}
	return finishArrowTable(schema, builder)
}

func ratingsArrowTable(payload MatchdayPayload) arrow.Table {
	schema := arrowSchema([]arrow.Field{
		stringField("run_id"), stringField("rating_key"), stringField("match_id"), intField("season"), intField("round"), stringField("player_id"),
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
			builder.Field(0).(*array.StringBuilder).Append(payload.RunID)
			builder.Field(1).(*array.StringBuilder).Append(fmt.Sprintf("%s:rating:%s", result.FixtureID, playerID))
			builder.Field(2).(*array.StringBuilder).Append(result.FixtureID)
			builder.Field(3).(*array.Int64Builder).Append(int64(payload.Season))
			builder.Field(4).(*array.Int64Builder).Append(int64(result.Round))
			builder.Field(5).(*array.StringBuilder).Append(playerID)
			builder.Field(6).(*array.StringBuilder).Append(player.ClubID)
			builder.Field(7).(*array.StringBuilder).Append(player.Name)
			builder.Field(8).(*array.Float64Builder).Append(rating)
			builder.Field(9).(*array.StringBuilder).Append(opponentID)
		}
	}
	return finishArrowTable(schema, builder)
}

func standingsArrowTable(payload MatchdayPayload) arrow.Table {
	schema := arrowSchema([]arrow.Field{
		stringField("run_id"), intField("season"), intField("round"), stringField("club_id"), intField("rank"),
		intField("played"), intField("won"), intField("drawn"), intField("lost"), intField("goals_for"),
		intField("goals_against"), intField("goal_difference"), intField("points"), stringField("form"),
	})
	builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	for _, standing := range payload.Standings {
		builder.Field(0).(*array.StringBuilder).Append(payload.RunID)
		builder.Field(1).(*array.Int64Builder).Append(int64(payload.Season))
		builder.Field(2).(*array.Int64Builder).Append(int64(payload.Round))
		builder.Field(3).(*array.StringBuilder).Append(standing.ClubID)
		builder.Field(4).(*array.Int64Builder).Append(int64(standing.Rank))
		builder.Field(5).(*array.Int64Builder).Append(int64(standing.Played))
		builder.Field(6).(*array.Int64Builder).Append(int64(standing.Won))
		builder.Field(7).(*array.Int64Builder).Append(int64(standing.Drawn))
		builder.Field(8).(*array.Int64Builder).Append(int64(standing.Lost))
		builder.Field(9).(*array.Int64Builder).Append(int64(standing.GoalsFor))
		builder.Field(10).(*array.Int64Builder).Append(int64(standing.GoalsAgainst))
		builder.Field(11).(*array.Int64Builder).Append(int64(standing.GoalDifference))
		builder.Field(12).(*array.Int64Builder).Append(int64(standing.Points))
		builder.Field(13).(*array.StringBuilder).Append(standing.Form)
	}
	return finishArrowTable(schema, builder)
}

func playerSnapshotsArrowTable(payload MatchdayPayload) arrow.Table {
	schema := arrowSchema([]arrow.Field{
		stringField("run_id"), intField("season"), intField("round"), stringField("player_id"), stringField("club_id"),
		stringField("player_name"), stringField("position"), intField("age"), intField("overall"), intField("potential"),
		intField("form"), intField("morale"), intField("fitness"), floatField("value"), floatField("average_rating"),
		intField("appeared"),
	})
	builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	for _, snapshot := range payload.PlayerSnapshots {
		builder.Field(0).(*array.StringBuilder).Append(payload.RunID)
		builder.Field(1).(*array.Int64Builder).Append(int64(payload.Season))
		builder.Field(2).(*array.Int64Builder).Append(int64(payload.Round))
		builder.Field(3).(*array.StringBuilder).Append(snapshot.PlayerID)
		builder.Field(4).(*array.StringBuilder).Append(snapshot.ClubID)
		builder.Field(5).(*array.StringBuilder).Append(snapshot.PlayerName)
		builder.Field(6).(*array.StringBuilder).Append(snapshot.Position)
		builder.Field(7).(*array.Int64Builder).Append(int64(snapshot.Age))
		builder.Field(8).(*array.Int64Builder).Append(int64(snapshot.Overall))
		builder.Field(9).(*array.Int64Builder).Append(int64(snapshot.Potential))
		builder.Field(10).(*array.Int64Builder).Append(int64(snapshot.Form))
		builder.Field(11).(*array.Int64Builder).Append(int64(snapshot.Morale))
		builder.Field(12).(*array.Int64Builder).Append(int64(snapshot.Fitness))
		builder.Field(13).(*array.Float64Builder).Append(snapshot.Value)
		builder.Field(14).(*array.Float64Builder).Append(snapshot.AverageRating)
		appeared := int64(0)
		if snapshot.Appeared {
			appeared = 1
		}
		builder.Field(15).(*array.Int64Builder).Append(appeared)
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
		{ID: 1, Name: "run_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 2, Name: "match_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 3, Name: "season", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 4, Name: "round", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 5, Name: "home_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 6, Name: "away_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 7, Name: "home_goals", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 8, Name: "away_goals", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 9, Name: "home_xg", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 10, Name: "away_xg", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 11, Name: "home_shots", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 12, Name: "away_shots", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 13, Name: "home_shots_on_target", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 14, Name: "away_shots_on_target", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 15, Name: "home_possession", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 16, Name: "home_pressure", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 17, Name: "away_pressure", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 18, Name: "home_territory", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 19, Name: "result", Type: iceberg.PrimitiveTypes.String, Required: true},
	})
}

func eventsIcebergSchema() *iceberg.Schema {
	return icebergSchema([]iceberg.NestedField{
		{ID: 1, Name: "run_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 2, Name: "event_key", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 3, Name: "match_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 4, Name: "season", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 5, Name: "round", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 6, Name: "minute", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 7, Name: "event_type", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 8, Name: "team_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 9, Name: "player_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 10, Name: "player_name", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 11, Name: "text", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 12, Name: "xg", Type: iceberg.PrimitiveTypes.Float64, Required: true},
	})
}

func framesIcebergSchema() *iceberg.Schema {
	return icebergSchema([]iceberg.NestedField{
		{ID: 1, Name: "run_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 2, Name: "frame_key", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 3, Name: "match_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 4, Name: "season", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 5, Name: "round", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 6, Name: "frame_index", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 7, Name: "minute", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 8, Name: "phase", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 9, Name: "possessing_team_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 10, Name: "ball_x", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 11, Name: "ball_y", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 12, Name: "player_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 13, Name: "team_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 14, Name: "player_name", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 15, Name: "position", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 16, Name: "player_x", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 17, Name: "player_y", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 18, Name: "target_x", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 19, Name: "target_y", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 20, Name: "intent", Type: iceberg.PrimitiveTypes.String, Required: true},
	})
}

func ratingsIcebergSchema() *iceberg.Schema {
	return icebergSchema([]iceberg.NestedField{
		{ID: 1, Name: "run_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 2, Name: "rating_key", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 3, Name: "match_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 4, Name: "season", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 5, Name: "round", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 6, Name: "player_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 7, Name: "club_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 8, Name: "player_name", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 9, Name: "rating", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 10, Name: "opponent_id", Type: iceberg.PrimitiveTypes.String, Required: true},
	})
}

func standingsIcebergSchema() *iceberg.Schema {
	return icebergSchema([]iceberg.NestedField{
		{ID: 1, Name: "run_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 2, Name: "season", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 3, Name: "round", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 4, Name: "club_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 5, Name: "rank", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 6, Name: "played", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 7, Name: "won", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 8, Name: "drawn", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 9, Name: "lost", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 10, Name: "goals_for", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 11, Name: "goals_against", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 12, Name: "goal_difference", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 13, Name: "points", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 14, Name: "form", Type: iceberg.PrimitiveTypes.String, Required: true},
	})
}

func playerSnapshotsIcebergSchema() *iceberg.Schema {
	return icebergSchema([]iceberg.NestedField{
		{ID: 1, Name: "run_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 2, Name: "season", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 3, Name: "round", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 4, Name: "player_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 5, Name: "club_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 6, Name: "player_name", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 7, Name: "position", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 8, Name: "age", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 9, Name: "overall", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 10, Name: "potential", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 11, Name: "form", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 12, Name: "morale", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 13, Name: "fitness", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 14, Name: "value", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 15, Name: "average_rating", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 16, Name: "appeared", Type: iceberg.PrimitiveTypes.Int64, Required: true},
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

func validatePayload(payload MatchdayPayload) error {
	if strings.TrimSpace(payload.RunID) == "" {
		return errors.New("runId is required")
	}
	if len(payload.RunID) > 160 {
		return errors.New("runId is too long")
	}
	if payload.Season < 1 || payload.Season > 1000 {
		return errors.New("season is out of range")
	}
	if payload.Round < 0 || payload.Round > 100 {
		return errors.New("round is out of range")
	}
	if len(payload.Results) > 10 || len(payload.Clubs) > 20 || len(payload.Players) > 500 || len(payload.Standings) > 20 || len(payload.PlayerSnapshots) > 500 {
		return errors.New("matchday arrays exceed the local league limits")
	}
	for _, result := range payload.Results {
		if strings.TrimSpace(result.FixtureID) == "" || strings.TrimSpace(result.HomeID) == "" || strings.TrimSpace(result.AwayID) == "" {
			return errors.New("result identifiers are required")
		}
		if len(result.Events) > 128 || len(result.Trace) > 600 || len(result.PlayerRatings) > 40 {
			return errors.New("result arrays exceed the replay limits")
		}
		for _, frame := range result.Trace {
			if len(frame.Players) > 22 {
				return errors.New("a replay frame has too many players")
			}
		}
	}
	return nil
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

func cors(next http.Handler, allowedOrigin string) http.Handler {
	allowedOrigins := make(map[string]struct{})
	for _, origin := range strings.Split(allowedOrigin, ",") {
		if trimmed := strings.TrimSpace(origin); trimmed != "" {
			allowedOrigins[trimmed] = struct{}{}
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if _, ok := allowedOrigins[origin]; ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
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
