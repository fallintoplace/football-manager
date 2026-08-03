package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	iceberg "github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	"github.com/apache/iceberg-go/table"
)

type RealDataPayload struct {
	Source        string      `json:"source"`
	SourceVersion string      `json:"sourceVersion"`
	Competition   string      `json:"competition"`
	Season        int         `json:"season"`
	SeasonLabel   string      `json:"seasonLabel"`
	Matches       []RealMatch `json:"matches"`
}

type RealMatch struct {
	SourceMatchID string           `json:"sourceMatchId"`
	MatchDate     string           `json:"matchDate"`
	HomeTeamID    string           `json:"homeTeamId"`
	HomeTeamName  string           `json:"homeTeamName"`
	AwayTeamID    string           `json:"awayTeamId"`
	AwayTeamName  string           `json:"awayTeamName"`
	HomeScore     int              `json:"homeScore"`
	AwayScore     int              `json:"awayScore"`
	Actions       []RealDataAction `json:"actions"`
}

type RealDataAction struct {
	SourceActionID      string         `json:"sourceActionId"`
	PossessionID        string         `json:"possessionId"`
	SequenceID          string         `json:"sequenceId"`
	Period              int            `json:"period"`
	Second              int            `json:"second"`
	TeamID              string         `json:"teamId"`
	TeamName            string         `json:"teamName"`
	PlayerID            string         `json:"playerId"`
	PlayerName          string         `json:"playerName"`
	RecipientPlayerID   string         `json:"recipientPlayerId"`
	RecipientPlayerName string         `json:"recipientPlayerName"`
	ActionType          string         `json:"actionType"`
	Outcome             string         `json:"outcome"`
	StartX              float64        `json:"startX"`
	StartY              float64        `json:"startY"`
	EndX                *float64       `json:"endX"`
	EndY                *float64       `json:"endY"`
	Qualifiers          map[string]any `json:"qualifiers"`
}

type RealMatchSummary struct {
	Source        string  `json:"source"`
	SourceMatchID string  `json:"sourceMatchId"`
	Competition   string  `json:"competition"`
	Season        int     `json:"season"`
	SeasonLabel   string  `json:"seasonLabel"`
	MatchDate     string  `json:"matchDate"`
	HomeTeamName  string  `json:"homeTeamName"`
	HomeScore     int     `json:"homeScore"`
	AwayTeamName  string  `json:"awayTeamName"`
	AwayScore     int     `json:"awayScore"`
	Actions       uint64  `json:"actions"`
	Players       uint64  `json:"players"`
	Possessions   uint64  `json:"possessions"`
	XG            float64 `json:"xg"`
}

type RealShot struct {
	TeamName   string  `json:"teamName"`
	PlayerName string  `json:"playerName"`
	Second     uint64  `json:"second"`
	StartX     float64 `json:"startX"`
	StartY     float64 `json:"startY"`
	XG         float64 `json:"xg"`
	Outcome    string  `json:"outcome"`
}

type RealPassNetworkLink struct {
	TeamName       string  `json:"teamName"`
	Passer         string  `json:"passer"`
	Receiver       string  `json:"receiver"`
	Attempts       uint64  `json:"attempts"`
	Completions    uint64  `json:"completions"`
	CompletionRate float64 `json:"completionRate"`
}

type RealPlayerProfile struct {
	PlayerID         string  `json:"playerId"`
	PlayerName       string  `json:"playerName"`
	TeamName         string  `json:"teamName"`
	Actions          uint64  `json:"actions"`
	Passes           uint64  `json:"passes"`
	CompletedPasses  uint64  `json:"completedPasses"`
	CompletionRate   float64 `json:"completionRate"`
	Carries          uint64  `json:"carries"`
	Shots            uint64  `json:"shots"`
	XG               float64 `json:"xg"`
	DefensiveActions uint64  `json:"defensiveActions"`
}

type RealMatchExplorer struct {
	Match          RealMatchSummary      `json:"match"`
	Shots          []RealShot            `json:"shots"`
	PassNetwork    []RealPassNetworkLink `json:"passNetwork"`
	PlayerProfiles []RealPlayerProfile   `json:"playerProfiles"`
}

func (s *Store) handleRealDataImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}

	var payload RealDataPayload
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<20))
	if err := decoder.Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid real-data payload"})
		return
	}
	if err := validateRealDataPayload(payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := s.ingestRealData(r.Context(), payload); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "real-data stores unavailable"})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"accepted": true,
		"source":   payload.Source,
		"matches":  len(payload.Matches),
		"actions":  countRealActions(payload),
	})
}

func (s *Store) handleRealDataMatches(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}

	source := r.URL.Query().Get("source")
	if source == "" {
		source = "statsbomb"
	}
	if len(source) > 80 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "source is too long"})
		return
	}
	season := 0
	if rawSeason := r.URL.Query().Get("season"); rawSeason != "" {
		parsed, err := strconv.Atoi(rawSeason)
		if err != nil || parsed < 1900 || parsed > 2200 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "season is invalid"})
			return
		}
		season = parsed
	}

	matches, err := s.realDataMatches(r.Context(), source, season)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "real-data store unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, matches)
}

func (s *Store) handleRealDataMatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}

	source := r.URL.Query().Get("source")
	matchID := r.URL.Query().Get("source_match_id")
	if source == "" || matchID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "source and source_match_id are required"})
		return
	}
	if len(source) > 80 || len(matchID) > 120 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "source or source_match_id is too long"})
		return
	}

	explorer, err := s.realMatchExplorer(r.Context(), source, matchID)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "real-data store unavailable"})
		return
	}
	if explorer.Match.SourceMatchID == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "real match not found"})
		return
	}
	writeJSON(w, http.StatusOK, explorer)
}

func (s *Store) realDataMatches(ctx context.Context, source string, season int) ([]RealMatchSummary, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := ensureRealDataSchema(ctx, s.clickhouse); err != nil {
		return nil, err
	}
	rows, err := s.clickhouse.Query(ctx, `
		SELECT
			m.source,
			m.source_match_id,
			m.competition,
			m.season,
			m.season_label,
			m.match_date,
			m.home_team_name,
			m.home_score,
			m.away_team_name,
			m.away_score,
			ifNull(a.actions, 0),
			ifNull(a.players, 0),
			ifNull(a.possessions, 0),
			ifNull(a.xg, 0)
		FROM touchline_real_matches_v1 AS m
		LEFT JOIN (
			SELECT source, source_match_id, count() AS actions, uniqExact(player_id) AS players,
				uniqExact(possession_id) AS possessions,
				sum(toFloat64OrZero(JSONExtractRaw(qualifiers, 'shotXg'))) AS xg
			FROM touchline_real_actions_v1
			GROUP BY source, source_match_id
		) AS a ON a.source = m.source AND a.source_match_id = m.source_match_id
		WHERE m.source = ? AND (? = 0 OR m.season = ?)
		ORDER BY m.match_date, m.source_match_id`, source, season, season)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	matches := make([]RealMatchSummary, 0)
	for rows.Next() {
		var match RealMatchSummary
		var season uint32
		var homeScore int32
		var awayScore int32
		if err := rows.Scan(
			&match.Source, &match.SourceMatchID, &match.Competition, &season, &match.SeasonLabel,
			&match.MatchDate, &match.HomeTeamName, &homeScore, &match.AwayTeamName, &awayScore,
			&match.Actions, &match.Players, &match.Possessions, &match.XG,
		); err != nil {
			return nil, err
		}
		match.Season = int(season)
		match.HomeScore = int(homeScore)
		match.AwayScore = int(awayScore)
		matches = append(matches, match)
	}
	return matches, rows.Err()
}

func (s *Store) realMatchExplorer(ctx context.Context, source string, matchID string) (RealMatchExplorer, error) {
	matches, err := s.realDataMatches(ctx, source, 0)
	if err != nil {
		return RealMatchExplorer{}, err
	}
	explorer := RealMatchExplorer{
		Shots:          []RealShot{},
		PassNetwork:    []RealPassNetworkLink{},
		PlayerProfiles: []RealPlayerProfile{},
	}
	for _, match := range matches {
		if match.SourceMatchID == matchID {
			explorer.Match = match
			break
		}
	}
	if explorer.Match.SourceMatchID == "" {
		return explorer, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	shots, err := s.clickhouse.Query(ctx, `
		SELECT team_name, player_name, second, start_x, start_y,
			toFloat64OrZero(JSONExtractRaw(qualifiers, 'shotXg')), outcome
		FROM touchline_real_actions_v1
		WHERE source = ? AND source_match_id = ? AND action_type = 'shot'
		ORDER BY second`, source, matchID)
	if err != nil {
		return RealMatchExplorer{}, err
	}
	for shots.Next() {
		var shot RealShot
		var second uint32
		var startX float32
		var startY float32
		if err := shots.Scan(&shot.TeamName, &shot.PlayerName, &second, &startX, &startY, &shot.XG, &shot.Outcome); err != nil {
			shots.Close()
			return RealMatchExplorer{}, err
		}
		shot.Second = uint64(second)
		shot.StartX = float64(startX)
		shot.StartY = float64(startY)
		explorer.Shots = append(explorer.Shots, shot)
	}
	if err := shots.Err(); err != nil {
		shots.Close()
		return RealMatchExplorer{}, err
	}
	shots.Close()

	network, err := s.clickhouse.Query(ctx, `
		SELECT team_name, player_name, recipient_player_name, count(), countIf(outcome = 'successful'),
			round(toFloat64(countIf(outcome = 'successful')) / greatest(toFloat64(count()), 1) * 100, 1)
		FROM touchline_real_actions_v1
		WHERE source = ? AND source_match_id = ? AND action_type = 'pass' AND recipient_player_id != ''
		GROUP BY team_name, player_name, recipient_player_name
		ORDER BY countIf(outcome = 'successful') DESC, count() DESC
		LIMIT 20`, source, matchID)
	if err != nil {
		return RealMatchExplorer{}, err
	}
	for network.Next() {
		var link RealPassNetworkLink
		if err := network.Scan(&link.TeamName, &link.Passer, &link.Receiver, &link.Attempts, &link.Completions, &link.CompletionRate); err != nil {
			network.Close()
			return RealMatchExplorer{}, err
		}
		explorer.PassNetwork = append(explorer.PassNetwork, link)
	}
	if err := network.Err(); err != nil {
		network.Close()
		return RealMatchExplorer{}, err
	}
	network.Close()

	profiles, err := s.clickhouse.Query(ctx, `
		SELECT player_id, any(player_name), any(team_name), count(), countIf(action_type = 'pass'),
			countIf(action_type = 'pass' AND outcome = 'successful'), countIf(action_type = 'carry'),
			countIf(action_type = 'shot'), sum(toFloat64OrZero(JSONExtractRaw(qualifiers, 'shotXg'))),
			countIf(action_type IN ('tackle', 'interception', 'duel', 'block', 'clearance', 'recovery'))
		FROM touchline_real_actions_v1
		WHERE source = ? AND source_match_id = ?
		GROUP BY player_id
		ORDER BY count() DESC
		LIMIT 40`, source, matchID)
	if err != nil {
		return RealMatchExplorer{}, err
	}
	for profiles.Next() {
		var profile RealPlayerProfile
		if err := profiles.Scan(
			&profile.PlayerID, &profile.PlayerName, &profile.TeamName, &profile.Actions, &profile.Passes,
			&profile.CompletedPasses, &profile.Carries, &profile.Shots, &profile.XG, &profile.DefensiveActions,
		); err != nil {
			profiles.Close()
			return RealMatchExplorer{}, err
		}
		if profile.Passes > 0 {
			profile.CompletionRate = float64(profile.CompletedPasses) / float64(profile.Passes) * 100
		}
		explorer.PlayerProfiles = append(explorer.PlayerProfiles, profile)
	}
	if err := profiles.Err(); err != nil {
		profiles.Close()
		return RealMatchExplorer{}, err
	}
	profiles.Close()
	return explorer, nil
}

func validateRealDataPayload(payload RealDataPayload) error {
	if payload.Source == "" || len(payload.Source) > 80 {
		return fmt.Errorf("source is required and must be at most 80 characters")
	}
	if payload.SourceVersion == "" || len(payload.SourceVersion) > 120 {
		return fmt.Errorf("sourceVersion is required and must be at most 120 characters")
	}
	if payload.Competition == "" || len(payload.Competition) > 120 {
		return fmt.Errorf("competition is required and must be at most 120 characters")
	}
	if payload.Season < 1900 || payload.Season > 2200 {
		return fmt.Errorf("season is outside the supported range")
	}
	if len(payload.Matches) == 0 || len(payload.Matches) > 20 {
		return fmt.Errorf("real-data import must contain between 1 and 20 matches")
	}
	totalActions := 0
	for _, match := range payload.Matches {
		if match.SourceMatchID == "" || len(match.SourceMatchID) > 120 {
			return fmt.Errorf("sourceMatchId is required and must be at most 120 characters")
		}
		if match.MatchDate == "" || match.HomeTeamID == "" || match.AwayTeamID == "" {
			return fmt.Errorf("real match metadata is incomplete")
		}
		if len(match.Actions) > 20000 {
			return fmt.Errorf("real match action count exceeds the safety limit")
		}
		totalActions += len(match.Actions)
		for _, action := range match.Actions {
			if action.SourceActionID == "" || action.TeamID == "" || action.PlayerID == "" || action.ActionType == "" {
				return fmt.Errorf("real action metadata is incomplete")
			}
			if action.StartX < 0 || action.StartX > 100 || action.StartY < 0 || action.StartY > 100 {
				return fmt.Errorf("real action coordinates must be normalized to 0-100")
			}
		}
	}
	if totalActions > 250000 {
		return fmt.Errorf("real-data import exceeds the total action safety limit")
	}
	return nil
}

func countRealActions(payload RealDataPayload) int {
	total := 0
	for _, match := range payload.Matches {
		total += len(match.Actions)
	}
	return total
}

func (s *Store) ingestRealData(ctx context.Context, payload RealDataPayload) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	if err := ensureRealDataSchema(ctx, s.clickhouse); err != nil {
		return err
	}

	for _, match := range payload.Matches {
		for _, tableName := range []string{"touchline_real_matches_v1", "touchline_real_actions_v1"} {
			statement := fmt.Sprintf("ALTER TABLE %s DELETE WHERE source = ? AND source_match_id = ? SETTINGS mutations_sync = 2", tableName)
			if err := s.clickhouse.Exec(ctx, statement, payload.Source, match.SourceMatchID); err != nil {
				return err
			}
		}
	}
	if err := insertRealClickHouse(ctx, s.clickhouse, payload); err != nil {
		return err
	}
	return s.iceberg.appendRealData(ctx, payload)
}

func ensureRealDataSchema(ctx context.Context, conn driver.Conn) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS touchline_real_matches_v1 (
			source LowCardinality(String),
			source_match_id String,
			competition String,
			season UInt32,
			season_label String,
			match_date String,
			home_team_id String,
			home_team_name String,
			away_team_id String,
			away_team_name String,
			home_score Int32,
			away_score Int32,
			imported_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = MergeTree ORDER BY (source, season, match_date, source_match_id)`,
		`CREATE TABLE IF NOT EXISTS touchline_real_actions_v1 (
			source LowCardinality(String),
			source_match_id String,
			competition String,
			season UInt32,
			season_label String,
			match_date String,
			source_action_id String,
			possession_id String,
			sequence_id String,
			period UInt8,
			second UInt32,
			team_id String,
			team_name String,
			player_id String,
			player_name String,
			recipient_player_id String,
			recipient_player_name String,
			action_type LowCardinality(String),
			outcome LowCardinality(String),
			start_x Float32,
			start_y Float32,
			end_x Float32,
			end_y Float32,
			qualifiers String,
			imported_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = MergeTree ORDER BY (source, season, source_match_id, second, source_action_id)`,
	}
	for _, statement := range statements {
		if err := conn.Exec(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func insertRealClickHouse(ctx context.Context, conn driver.Conn, payload RealDataPayload) error {
	matches, err := conn.PrepareBatch(ctx, `INSERT INTO touchline_real_matches_v1 (
		source, source_match_id, competition, season, season_label, match_date,
		home_team_id, home_team_name, away_team_id, away_team_name, home_score, away_score
	)`)
	if err != nil {
		return err
	}
	for _, match := range payload.Matches {
		if err := matches.Append(
			payload.Source, match.SourceMatchID, payload.Competition, payload.Season, payload.SeasonLabel, match.MatchDate,
			match.HomeTeamID, match.HomeTeamName, match.AwayTeamID, match.AwayTeamName, match.HomeScore, match.AwayScore,
		); err != nil {
			return err
		}
	}
	if err := matches.Send(); err != nil {
		return err
	}

	actions, err := conn.PrepareBatch(ctx, `INSERT INTO touchline_real_actions_v1 (
		source, source_match_id, competition, season, season_label, match_date,
		source_action_id, possession_id, sequence_id, period, second, team_id, team_name,
		player_id, player_name, recipient_player_id, recipient_player_name, action_type, outcome,
		start_x, start_y, end_x, end_y, qualifiers
	)`)
	if err != nil {
		return err
	}
	for _, match := range payload.Matches {
		for _, action := range match.Actions {
			qualifiers, err := json.Marshal(action.Qualifiers)
			if err != nil {
				return err
			}
			endX, endY := float64(0), float64(0)
			if action.EndX != nil {
				endX = *action.EndX
			}
			if action.EndY != nil {
				endY = *action.EndY
			}
			if err := actions.Append(
				payload.Source, match.SourceMatchID, payload.Competition, payload.Season, payload.SeasonLabel, match.MatchDate,
				action.SourceActionID, action.PossessionID, action.SequenceID, action.Period, action.Second,
				action.TeamID, action.TeamName, action.PlayerID, action.PlayerName, action.RecipientPlayerID,
				action.RecipientPlayerName, action.ActionType, action.Outcome, action.StartX, action.StartY,
				endX, endY, string(qualifiers),
			); err != nil {
				return err
			}
		}
	}
	return actions.Send()
}

func realMatchesIcebergSchema() *iceberg.Schema {
	return icebergSchema([]iceberg.NestedField{
		{ID: 1, Name: "source", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 2, Name: "source_match_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 3, Name: "competition", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 4, Name: "season", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 5, Name: "season_label", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 6, Name: "match_date", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 7, Name: "home_team_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 8, Name: "home_team_name", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 9, Name: "away_team_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 10, Name: "away_team_name", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 11, Name: "home_score", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 12, Name: "away_score", Type: iceberg.PrimitiveTypes.Int64, Required: true},
	})
}

func realActionsIcebergSchema() *iceberg.Schema {
	return icebergSchema([]iceberg.NestedField{
		{ID: 1, Name: "source", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 2, Name: "source_match_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 3, Name: "competition", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 4, Name: "season", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 5, Name: "season_label", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 6, Name: "match_date", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 7, Name: "source_action_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 8, Name: "possession_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 9, Name: "sequence_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 10, Name: "period", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 11, Name: "second", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 12, Name: "team_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 13, Name: "team_name", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 14, Name: "player_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 15, Name: "player_name", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 16, Name: "recipient_player_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 17, Name: "recipient_player_name", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 18, Name: "action_type", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 19, Name: "outcome", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 20, Name: "start_x", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 21, Name: "start_y", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 22, Name: "end_x", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 23, Name: "end_y", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 24, Name: "qualifiers", Type: iceberg.PrimitiveTypes.String, Required: true},
	})
}

func realMatchesArrowTable(payload RealDataPayload, matches []RealMatch) arrow.Table {
	schema := arrowSchema([]arrow.Field{
		stringField("source"), stringField("source_match_id"), stringField("competition"), intField("season"), stringField("season_label"), stringField("match_date"),
		stringField("home_team_id"), stringField("home_team_name"), stringField("away_team_id"), stringField("away_team_name"), intField("home_score"), intField("away_score"),
	})
	builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	for _, match := range matches {
		builder.Field(0).(*array.StringBuilder).Append(payload.Source)
		builder.Field(1).(*array.StringBuilder).Append(match.SourceMatchID)
		builder.Field(2).(*array.StringBuilder).Append(payload.Competition)
		builder.Field(3).(*array.Int64Builder).Append(int64(payload.Season))
		builder.Field(4).(*array.StringBuilder).Append(payload.SeasonLabel)
		builder.Field(5).(*array.StringBuilder).Append(match.MatchDate)
		builder.Field(6).(*array.StringBuilder).Append(match.HomeTeamID)
		builder.Field(7).(*array.StringBuilder).Append(match.HomeTeamName)
		builder.Field(8).(*array.StringBuilder).Append(match.AwayTeamID)
		builder.Field(9).(*array.StringBuilder).Append(match.AwayTeamName)
		builder.Field(10).(*array.Int64Builder).Append(int64(match.HomeScore))
		builder.Field(11).(*array.Int64Builder).Append(int64(match.AwayScore))
	}
	return finishArrowTable(schema, builder)
}

func realActionsArrowTable(payload RealDataPayload, match RealMatch) arrow.Table {
	schema := arrowSchema([]arrow.Field{
		stringField("source"), stringField("source_match_id"), stringField("competition"), intField("season"), stringField("season_label"), stringField("match_date"),
		stringField("source_action_id"), stringField("possession_id"), stringField("sequence_id"), intField("period"), intField("second"),
		stringField("team_id"), stringField("team_name"), stringField("player_id"), stringField("player_name"), stringField("recipient_player_id"), stringField("recipient_player_name"),
		stringField("action_type"), stringField("outcome"), floatField("start_x"), floatField("start_y"), floatField("end_x"), floatField("end_y"), stringField("qualifiers"),
	})
	builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	for _, action := range match.Actions {
		qualifiers, err := json.Marshal(action.Qualifiers)
		if err != nil {
			continue
		}
		endX, endY := float64(0), float64(0)
		if action.EndX != nil {
			endX = *action.EndX
		}
		if action.EndY != nil {
			endY = *action.EndY
		}
		builder.Field(0).(*array.StringBuilder).Append(payload.Source)
		builder.Field(1).(*array.StringBuilder).Append(match.SourceMatchID)
		builder.Field(2).(*array.StringBuilder).Append(payload.Competition)
		builder.Field(3).(*array.Int64Builder).Append(int64(payload.Season))
		builder.Field(4).(*array.StringBuilder).Append(payload.SeasonLabel)
		builder.Field(5).(*array.StringBuilder).Append(match.MatchDate)
		builder.Field(6).(*array.StringBuilder).Append(action.SourceActionID)
		builder.Field(7).(*array.StringBuilder).Append(action.PossessionID)
		builder.Field(8).(*array.StringBuilder).Append(action.SequenceID)
		builder.Field(9).(*array.Int64Builder).Append(int64(action.Period))
		builder.Field(10).(*array.Int64Builder).Append(int64(action.Second))
		builder.Field(11).(*array.StringBuilder).Append(action.TeamID)
		builder.Field(12).(*array.StringBuilder).Append(action.TeamName)
		builder.Field(13).(*array.StringBuilder).Append(action.PlayerID)
		builder.Field(14).(*array.StringBuilder).Append(action.PlayerName)
		builder.Field(15).(*array.StringBuilder).Append(action.RecipientPlayerID)
		builder.Field(16).(*array.StringBuilder).Append(action.RecipientPlayerName)
		builder.Field(17).(*array.StringBuilder).Append(action.ActionType)
		builder.Field(18).(*array.StringBuilder).Append(action.Outcome)
		builder.Field(19).(*array.Float64Builder).Append(action.StartX)
		builder.Field(20).(*array.Float64Builder).Append(action.StartY)
		builder.Field(21).(*array.Float64Builder).Append(endX)
		builder.Field(22).(*array.Float64Builder).Append(endY)
		builder.Field(23).(*array.StringBuilder).Append(string(qualifiers))
	}
	return finishArrowTable(schema, builder)
}

func (w *icebergWriter) appendRealData(ctx context.Context, payload RealDataPayload) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.ensureCatalog(ctx); err != nil {
		return err
	}
	if err := w.catalog.CreateNamespace(ctx, catalog.ToIdentifier("touchline"), nil); err != nil && !errors.Is(err, catalog.ErrNamespaceAlreadyExists) {
		return err
	}

	for _, match := range payload.Matches {
		items := []struct {
			name   string
			schema *iceberg.Schema
			table  arrow.Table
		}{
			{name: "real_matches", schema: realMatchesIcebergSchema(), table: realMatchesArrowTable(payload, []RealMatch{match})},
			{name: "real_actions", schema: realActionsIcebergSchema(), table: realActionsArrowTable(payload, match)},
		}
		for _, item := range items {
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
				iceberg.Properties{"source": payload.Source, "source_version": payload.SourceVersion},
				table.WithOverwriteFilter(iceberg.NewAnd(
					iceberg.EqualTo(iceberg.Reference("source"), payload.Source),
					iceberg.EqualTo(iceberg.Reference("source_match_id"), match.SourceMatchID),
				)),
			)
			item.table.Release()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
