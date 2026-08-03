package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
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
	FixtureID     string                      `json:"fixtureId"`
	Round         int                         `json:"round"`
	HomeID        string                      `json:"homeId"`
	AwayID        string                      `json:"awayId"`
	HomeGoals     int                         `json:"homeGoals"`
	AwayGoals     int                         `json:"awayGoals"`
	Events        []MatchEvent                `json:"events"`
	Actions       []MatchAction               `json:"actions"`
	Trace         []MatchFrame                `json:"trace"`
	Metrics       MatchMetrics                `json:"metrics"`
	Tactics       MatchTactics                `json:"tactics"`
	PlayerStats   map[string]PlayerMatchStats `json:"playerStats"`
	PlayerRatings map[string]float64          `json:"playerRatings"`
}

type MatchEvent struct {
	Minute     int      `json:"minute"`
	Type       string   `json:"type"`
	TeamID     string   `json:"teamId"`
	PlayerName string   `json:"playerName"`
	Text       string   `json:"text"`
	XG         *float64 `json:"xg"`
}

type MatchAction struct {
	ID                string         `json:"id"`
	MatchID           string         `json:"matchId"`
	SequenceID        string         `json:"sequenceId"`
	PossessionID      string         `json:"possessionId"`
	Period            int            `json:"period"`
	Second            int            `json:"second"`
	TeamID            string         `json:"teamId"`
	PlayerID          string         `json:"playerId"`
	RecipientPlayerID string         `json:"recipientPlayerId"`
	Type              string         `json:"type"`
	Outcome           string         `json:"outcome"`
	StartX            float64        `json:"startX"`
	StartY            float64        `json:"startY"`
	EndX              *float64       `json:"endX"`
	EndY              *float64       `json:"endY"`
	Qualifiers        map[string]any `json:"qualifiers"`
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
	HomePossessions   int     `json:"homePossessions"`
	AwayPossessions   int     `json:"awayPossessions"`
	HomeFinalThird    int     `json:"homeFinalThirdEntries"`
	AwayFinalThird    int     `json:"awayFinalThirdEntries"`
	HomeBoxEntries    int     `json:"homeBoxEntries"`
	AwayBoxEntries    int     `json:"awayBoxEntries"`
	HomePressWins     int     `json:"homePressWins"`
	AwayPressWins     int     `json:"awayPressWins"`
	HomeBuildUpFails  int     `json:"homeBuildUpFails"`
	AwayBuildUpFails  int     `json:"awayBuildUpFails"`
	HomeMidfieldWins  int     `json:"homeMidfieldWins"`
	AwayMidfieldWins  int     `json:"awayMidfieldWins"`
	HomeLineBreaks    int     `json:"homeLineBreaks"`
	AwayLineBreaks    int     `json:"awayLineBreaks"`
	HomeBallsBehind   int     `json:"homeBallsBehind"`
	AwayBallsBehind   int     `json:"awayBallsBehind"`
	HomeCounters      int     `json:"homeCounters"`
	AwayCounters      int     `json:"awayCounters"`
	HomeSaves         int     `json:"homeSaves"`
	AwaySaves         int     `json:"awaySaves"`
	HomeCards         int     `json:"homeCards"`
	AwayCards         int     `json:"awayCards"`
	HomeFatigueLosses int     `json:"homeLateFatigueLosses"`
	AwayFatigueLosses int     `json:"awayLateFatigueLosses"`
}

type TacticSnapshot struct {
	Formation     string `json:"formation"`
	Mentality     string `json:"mentality"`
	Pressing      int    `json:"pressing"`
	Tempo         int    `json:"tempo"`
	DefensiveLine int    `json:"defensiveLine"`
}

type MatchTactics struct {
	Home TacticSnapshot `json:"home"`
	Away TacticSnapshot `json:"away"`
}

type PlayerMatchStats struct {
	Started       bool    `json:"started"`
	MinutesPlayed int     `json:"minutesPlayed"`
	Goals         int     `json:"goals"`
	Shots         int     `json:"shots"`
	XG            float64 `json:"xg"`
}

type AnalyticsRun struct {
	RunID           string `json:"runId"`
	CareerID        string `json:"careerId"`
	Season          uint32 `json:"season"`
	LastRound       uint32 `json:"lastRound"`
	RoundsCompleted uint32 `json:"roundsCompleted"`
	MatchesInRound  uint32 `json:"matchesInRound"`
	ClubsExpected   uint32 `json:"clubsExpected"`
	Status          string `json:"status"`
	SchemaVersion   uint32 `json:"schemaVersion"`
	UpdatedAt       string `json:"updatedAt"`
}

type AnalyticsSeasonComparison struct {
	CareerID          string  `json:"careerId"`
	RunID             string  `json:"runId"`
	Season            uint32  `json:"season"`
	Status            string  `json:"status"`
	LastRound         uint32  `json:"lastRound"`
	Rank              uint32  `json:"rank"`
	Played            uint32  `json:"played"`
	Won               uint32  `json:"won"`
	Drawn             uint32  `json:"drawn"`
	Lost              uint32  `json:"lost"`
	Points            uint32  `json:"points"`
	GoalsFor          int32   `json:"goalsFor"`
	GoalsAgainst      int32   `json:"goalsAgainst"`
	Matches           uint64  `json:"matches"`
	XGFor             float64 `json:"xgFor"`
	XGAgainst         float64 `json:"xgAgainst"`
	AverageRating     float64 `json:"averageRating"`
	AveragePossession float64 `json:"averagePossession"`
	AveragePressure   float64 `json:"averagePressure"`
	AveragePressWins  float64 `json:"averagePressWins"`
	AverageBoxEntries float64 `json:"averageBoxEntries"`
	PlayerMinutes     uint64  `json:"playerMinutes"`
}

type TacticalMatchup struct {
	RunID                 string  `json:"runId"`
	ClubID                string  `json:"clubId"`
	OpponentID            string  `json:"opponentId"`
	ClubFormation         string  `json:"clubFormation"`
	ClubMentality         string  `json:"clubMentality"`
	ClubPressing          float64 `json:"clubPressing"`
	ClubTempo             float64 `json:"clubTempo"`
	ClubDefensiveLine     float64 `json:"clubDefensiveLine"`
	OpponentFormation     string  `json:"opponentFormation"`
	OpponentMentality     string  `json:"opponentMentality"`
	OpponentPressing      float64 `json:"opponentPressing"`
	OpponentTempo         float64 `json:"opponentTempo"`
	OpponentDefensiveLine float64 `json:"opponentDefensiveLine"`
	Matches               uint64  `json:"matches"`
	GoalsFor              uint64  `json:"goalsFor"`
	GoalsAgainst          uint64  `json:"goalsAgainst"`
	XGFor                 float64 `json:"xgFor"`
	XGAgainst             float64 `json:"xgAgainst"`
	Possession            float64 `json:"possession"`
	PressWins             float64 `json:"pressWins"`
	OpponentPressWins     float64 `json:"opponentPressWins"`
	BoxEntries            float64 `json:"boxEntries"`
	OpponentBoxEntries    float64 `json:"opponentBoxEntries"`
	Counters              float64 `json:"counters"`
	BuildUpFails          float64 `json:"buildUpFails"`
	OpponentBuildUpFails  float64 `json:"opponentBuildUpFails"`
}

type ActionMixRow struct {
	ActionType  string  `json:"actionType"`
	Actions     uint64  `json:"actions"`
	Successful  uint64  `json:"successful"`
	SuccessRate float64 `json:"successRate"`
}

type PassNetworkLink struct {
	PasserID          string  `json:"passerId"`
	ReceiverID        string  `json:"receiverId"`
	Attempts          uint64  `json:"attempts"`
	Completions       uint64  `json:"completions"`
	ProgressivePasses uint64  `json:"progressivePasses"`
	CompletionRate    float64 `json:"completionRate"`
}

type PlayerActionProfile struct {
	PlayerID           string  `json:"playerId"`
	PrimaryRole        string  `json:"primaryRole"`
	Actions            uint64  `json:"actions"`
	Passes             uint64  `json:"passes"`
	CompletedPasses    uint64  `json:"completedPasses"`
	CompletionRate     float64 `json:"completionRate"`
	ProgressiveActions uint64  `json:"progressiveActions"`
	Carries            uint64  `json:"carries"`
	Shots              uint64  `json:"shots"`
	XG                 float64 `json:"xg"`
	DefensiveActions   uint64  `json:"defensiveActions"`
	BuildUpActions     uint64  `json:"buildUpActions"`
	FinalThirdActions  uint64  `json:"finalThirdActions"`
	BoxActions         uint64  `json:"boxActions"`
}

type ActionInsights struct {
	RunID             string                `json:"runId"`
	ClubID            string                `json:"clubId"`
	Matches           uint64                `json:"matches"`
	Actions           uint64                `json:"actions"`
	Possessions       uint64                `json:"possessions"`
	Passes            uint64                `json:"passes"`
	CompletedPasses   uint64                `json:"completedPasses"`
	PassCompletion    float64               `json:"passCompletion"`
	ProgressivePasses uint64                `json:"progressivePasses"`
	Shots             uint64                `json:"shots"`
	ShotsOnTarget     uint64                `json:"shotsOnTarget"`
	XG                float64               `json:"xg"`
	Carries           uint64                `json:"carries"`
	SuccessfulCarries uint64                `json:"successfulCarries"`
	FinalThirdActions uint64                `json:"finalThirdActions"`
	ActionMix         []ActionMixRow        `json:"actionMix"`
	PassNetwork       []PassNetworkLink     `json:"passNetwork"`
	PlayerProfiles    []PlayerActionProfile `json:"playerProfiles"`
	AnalystNote       string                `json:"analystNote"`
}

type IcebergSnapshot struct {
	SnapshotID       int64  `json:"snapshotId"`
	ParentSnapshotID *int64 `json:"parentSnapshotId,omitempty"`
	SequenceNumber   int64  `json:"sequenceNumber"`
	TimestampMs      int64  `json:"timestampMs"`
	OccurredAt       string `json:"occurredAt"`
	Summary          string `json:"summary"`
}

type IcebergHistory struct {
	Table             string            `json:"table"`
	CurrentSnapshotID *int64            `json:"currentSnapshotId,omitempty"`
	Snapshots         []IcebergSnapshot `json:"snapshots"`
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

var icebergTableNames = map[string]struct{}{
	"runs": {}, "matches": {}, "match_club_facts": {}, "match_events": {}, "match_actions": {}, "player_frames": {},
	"player_ratings": {}, "player_match_facts": {}, "standings": {}, "player_snapshots": {},
	"real_matches": {}, "real_actions": {},
	"worldcup_2026_matches": {}, "worldcup_2026_goals": {},
}

func main() {
	store, err := newStore()
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", store.handleHealth)
	mux.HandleFunc("/ingest", store.handleIngest)
	mux.HandleFunc("/api/analytics/runs", store.handleRuns)
	mux.HandleFunc("/api/analytics/season-comparison", store.handleSeasonComparison)
	mux.HandleFunc("/api/analytics/tactical-matchup", store.handleTacticalMatchup)
	mux.HandleFunc("/api/analytics/action-insights", store.handleActionInsights)
	mux.HandleFunc("/api/analytics/real-data/import", store.handleRealDataImport)
	mux.HandleFunc("/api/analytics/real-data/matches", store.handleRealDataMatches)
	mux.HandleFunc("/api/analytics/real-data/match", store.handleRealDataMatch)
	mux.HandleFunc("/api/analytics/worldcup-2026/import", store.handleWorldCup2026Import)
	mux.HandleFunc("/api/analytics/worldcup-2026/overview", store.handleWorldCup2026Overview)
	mux.HandleFunc("/api/analytics/iceberg/history", store.handleIcebergHistory)
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

func (s *Store) handleRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}

	runs, err := s.runs(r.Context())
	if err != nil {
		log.Printf("runs failed: %v", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "analytics store unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, runs)
}

func (s *Store) handleSeasonComparison(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}

	careerID := r.URL.Query().Get("career_id")
	clubID := r.URL.Query().Get("club_id")
	if careerID == "" || clubID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "career_id and club_id are required"})
		return
	}
	if len(careerID) > 160 || len(clubID) > 80 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "career_id or club_id is too long"})
		return
	}

	comparison, err := s.seasonComparison(r.Context(), careerID, clubID)
	if err != nil {
		log.Printf("season comparison failed: %v", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "analytics store unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, comparison)
}

func (s *Store) handleTacticalMatchup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}

	runID := r.URL.Query().Get("run_id")
	clubID := r.URL.Query().Get("club_id")
	opponentID := r.URL.Query().Get("opponent_id")
	if runID == "" || clubID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "run_id and club_id are required"})
		return
	}
	if len(runID) > 160 || len(clubID) > 80 || len(opponentID) > 80 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "run_id, club_id, or opponent_id is too long"})
		return
	}

	matchups, err := s.tacticalMatchup(r.Context(), runID, clubID, opponentID)
	if err != nil {
		log.Printf("tactical matchup failed: %v", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "analytics store unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, matchups)
}

func (s *Store) handleActionInsights(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}

	runID := r.URL.Query().Get("run_id")
	clubID := r.URL.Query().Get("club_id")
	if runID == "" || clubID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "run_id and club_id are required"})
		return
	}
	if len(runID) > 160 || len(clubID) > 80 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "run_id or club_id is too long"})
		return
	}

	insights, err := s.actionInsights(r.Context(), runID, clubID)
	if err != nil {
		log.Printf("action insights failed: %v", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "analytics store unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, insights)
}

func (s *Store) handleIcebergHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}

	tableName := r.URL.Query().Get("table")
	if _, ok := icebergTableNames[tableName]; !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown Iceberg table"})
		return
	}
	history, err := s.iceberg.history(r.Context(), tableName)
	if err != nil {
		log.Printf("Iceberg history failed for %s: %v", tableName, err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Iceberg catalog unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, history)
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

func (s *Store) runs(ctx context.Context) ([]AnalyticsRun, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := ensureClickHouseSchema(ctx, s.clickhouse); err != nil {
		return nil, err
	}

	rows, err := s.clickhouse.Query(ctx, `
		SELECT
			run_id,
			any(career_id),
			any(season),
			argMax(last_round, updated_at),
			argMax(rounds_completed, updated_at),
			argMax(matches_in_round, updated_at),
			argMax(clubs_expected, updated_at),
			argMax(status, updated_at),
			argMax(schema_version, updated_at),
			toString(max(updated_at))
		FROM touchline_runs_v2
		GROUP BY run_id
		ORDER BY max(updated_at) DESC
		LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	runs := make([]AnalyticsRun, 0)
	for rows.Next() {
		var run AnalyticsRun
		if err := rows.Scan(
			&run.RunID, &run.CareerID, &run.Season, &run.LastRound, &run.RoundsCompleted,
			&run.MatchesInRound, &run.ClubsExpected, &run.Status, &run.SchemaVersion, &run.UpdatedAt,
		); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return runs, nil
}

func (s *Store) seasonComparison(ctx context.Context, careerID string, clubID string) ([]AnalyticsSeasonComparison, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := ensureClickHouseSchema(ctx, s.clickhouse); err != nil {
		return nil, err
	}

	rows, err := s.clickhouse.Query(ctx, `
		WITH run_registry AS (
			SELECT
				run_id,
				any(career_id) AS career_id,
				any(season) AS season,
				argMax(status, updated_at) AS status,
				argMax(last_round, updated_at) AS last_round
			FROM touchline_runs_v2
			WHERE touchline_runs_v2.career_id = ?
			GROUP BY run_id
		), club_facts AS (
			SELECT
				f.run_id,
				f.season,
				count() AS matches,
				sum(f.goals_for) AS goals_for,
				sum(f.goals_against) AS goals_against,
				sum(f.xg_for) AS xg_for,
				avg(f.possession) AS average_possession,
				avg(f.pressure) AS average_pressure,
				avg(f.press_wins) AS average_press_wins,
				avg(f.box_entries) AS average_box_entries
			FROM touchline_match_club_facts_v2 AS f
			INNER JOIN run_registry AS r ON r.run_id = f.run_id AND r.season = f.season
			WHERE f.club_id = ?
			GROUP BY f.run_id, f.season
		), opponents AS (
			SELECT
				f.run_id,
				f.season,
				sum(f.xg_for) AS xg_against
			FROM touchline_match_club_facts_v2 AS f
			INNER JOIN run_registry AS r ON r.run_id = f.run_id AND r.season = f.season
			WHERE f.opponent_id = ?
			GROUP BY f.run_id, f.season
		), latest_standings AS (
			SELECT
				s.run_id,
				s.season,
				argMax(s.rank, s.round) AS rank,
				argMax(s.played, s.round) AS played,
				argMax(s.won, s.round) AS won,
				argMax(s.drawn, s.round) AS drawn,
				argMax(s.lost, s.round) AS lost,
				argMax(s.points, s.round) AS points,
				argMax(s.goals_for, s.round) AS goals_for,
				argMax(s.goals_against, s.round) AS goals_against
			FROM (SELECT * FROM touchline_standings FINAL) AS s
			INNER JOIN run_registry AS r ON r.run_id = s.run_id AND r.season = s.season
			WHERE s.club_id = ?
			GROUP BY s.run_id, s.season
		), player_facts AS (
			SELECT
				f.run_id,
				f.season,
				sum(f.minutes_played) AS player_minutes,
				avgIf(f.rating, f.minutes_played > 0) AS average_rating
			FROM touchline_player_match_facts_v2 AS f
			INNER JOIN run_registry AS r ON r.run_id = f.run_id AND r.season = f.season
			WHERE f.club_id = ?
			GROUP BY f.run_id, f.season
		)
		SELECT
			r.career_id,
			r.run_id,
			r.season,
			r.status,
			r.last_round,
			ifNull(s.rank, 0),
			ifNull(s.played, 0),
			ifNull(s.won, 0),
			ifNull(s.drawn, 0),
			ifNull(s.lost, 0),
			ifNull(s.points, 0),
			ifNull(s.goals_for, 0),
			ifNull(s.goals_against, 0),
			ifNull(f.matches, 0),
			ifNull(f.xg_for, 0),
			ifNull(o.xg_against, 0),
			ifNull(p.average_rating, 0),
			ifNull(f.average_possession, 0),
			ifNull(f.average_pressure, 0),
			ifNull(f.average_press_wins, 0),
			ifNull(f.average_box_entries, 0),
			ifNull(p.player_minutes, 0)
		FROM run_registry AS r
		LEFT JOIN latest_standings AS s ON s.run_id = r.run_id AND s.season = r.season
		LEFT JOIN club_facts AS f ON f.run_id = r.run_id AND f.season = r.season
		LEFT JOIN opponents AS o ON o.run_id = r.run_id AND o.season = r.season
		LEFT JOIN player_facts AS p ON p.run_id = r.run_id AND p.season = r.season
		ORDER BY r.season`, careerID, clubID, clubID, clubID, clubID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comparison := make([]AnalyticsSeasonComparison, 0)
	for rows.Next() {
		var season AnalyticsSeasonComparison
		if err := rows.Scan(
			&season.CareerID, &season.RunID, &season.Season, &season.Status, &season.LastRound,
			&season.Rank, &season.Played, &season.Won, &season.Drawn, &season.Lost, &season.Points,
			&season.GoalsFor, &season.GoalsAgainst, &season.Matches, &season.XGFor, &season.XGAgainst,
			&season.AverageRating, &season.AveragePossession, &season.AveragePressure, &season.AveragePressWins,
			&season.AverageBoxEntries, &season.PlayerMinutes,
		); err != nil {
			return nil, err
		}
		comparison = append(comparison, season)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return comparison, nil
}

func (s *Store) tacticalMatchup(ctx context.Context, runID string, clubID string, opponentID string) ([]TacticalMatchup, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := ensureClickHouseSchema(ctx, s.clickhouse); err != nil {
		return nil, err
	}

	rows, err := s.clickhouse.Query(ctx, `
		SELECT
			cf.run_id,
			cf.club_id,
			cf.opponent_id,
			cf.formation,
			cf.mentality,
			avg(cf.pressing),
			avg(cf.tempo),
			avg(cf.defensive_line),
			opp.formation,
			opp.mentality,
			avg(opp.pressing),
			avg(opp.tempo),
			avg(opp.defensive_line),
			count(),
			sum(cf.goals_for),
			sum(cf.goals_against),
			avg(cf.xg_for),
			avg(opp.xg_for),
			avg(cf.possession),
			avg(cf.press_wins),
			avg(opp.press_wins),
			avg(cf.box_entries),
			avg(opp.box_entries),
			avg(cf.counters),
			avg(cf.build_up_fails),
			avg(opp.build_up_fails)
		FROM touchline_match_club_facts_v2 AS cf
		INNER JOIN touchline_match_club_facts_v2 AS opp
			ON opp.run_id = cf.run_id
			AND opp.match_id = cf.match_id
			AND opp.club_id = cf.opponent_id
		WHERE cf.run_id = ?
			AND cf.club_id = ?
			AND (? = '' OR cf.opponent_id = ?)
		GROUP BY
			cf.run_id,
			cf.club_id,
			cf.opponent_id,
			cf.formation,
			cf.mentality,
			opp.formation,
			opp.mentality
		ORDER BY count() DESC, avg(cf.xg_for) DESC`, runID, clubID, opponentID, opponentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	matchups := make([]TacticalMatchup, 0)
	for rows.Next() {
		var matchup TacticalMatchup
		if err := rows.Scan(
			&matchup.RunID, &matchup.ClubID, &matchup.OpponentID, &matchup.ClubFormation, &matchup.ClubMentality,
			&matchup.ClubPressing, &matchup.ClubTempo, &matchup.ClubDefensiveLine, &matchup.OpponentFormation,
			&matchup.OpponentMentality, &matchup.OpponentPressing, &matchup.OpponentTempo, &matchup.OpponentDefensiveLine,
			&matchup.Matches, &matchup.GoalsFor, &matchup.GoalsAgainst, &matchup.XGFor, &matchup.XGAgainst,
			&matchup.Possession, &matchup.PressWins, &matchup.OpponentPressWins, &matchup.BoxEntries,
			&matchup.OpponentBoxEntries, &matchup.Counters, &matchup.BuildUpFails, &matchup.OpponentBuildUpFails,
		); err != nil {
			return nil, err
		}
		matchups = append(matchups, matchup)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return matchups, nil
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
		complete, err := s.hasCompleteMatch(ctx, payload.RunID, result)
		if err != nil {
			return err
		}
		if ingested && complete {
			continue
		}
		if !ingested {
			icebergResults = append(icebergResults, result)
		}
		if !complete {
			clickHouseResults = append(clickHouseResults, result)
		}
	}
	if len(clickHouseResults) == 0 && len(icebergResults) == 0 {
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
	if len(icebergResults) > 0 {
		if err := s.iceberg.append(ctx, icebergPayload); err != nil {
			return err
		}
	}
	if len(icebergResults) > 0 {
		if err := s.markIngested(ctx, payload.RunID, icebergResults); err != nil {
			return err
		}
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
	var matches, events, actions, frames, ratings, clubFacts, playerFacts uint64
	if err := s.clickhouse.QueryRow(ctx, `
		SELECT
			(SELECT count() FROM touchline_matches_v2 WHERE run_id = ? AND match_id = ?),
			(SELECT count() FROM touchline_match_events_v2 WHERE run_id = ? AND match_id = ?),
			(SELECT count() FROM touchline_match_actions_v2 WHERE run_id = ? AND match_id = ?),
			(SELECT count() FROM touchline_player_frames_v2 WHERE run_id = ? AND match_id = ?),
			(SELECT count() FROM touchline_player_ratings_v2 WHERE run_id = ? AND match_id = ?),
			(SELECT count() FROM touchline_match_club_facts_v2 WHERE run_id = ? AND match_id = ?),
			(SELECT count() FROM touchline_player_match_facts_v2 WHERE run_id = ? AND match_id = ?)`,
		runID, result.FixtureID, runID, result.FixtureID, runID, result.FixtureID, runID, result.FixtureID,
		runID, result.FixtureID, runID, result.FixtureID, runID, result.FixtureID,
	).Scan(&matches, &events, &actions, &frames, &ratings, &clubFacts, &playerFacts); err != nil {
		return false, err
	}
	expectedFrames := 0
	for _, frame := range result.Trace {
		expectedFrames += len(frame.Players)
	}
	return matches == 1 && events == uint64(len(result.Events)) &&
		(len(result.Actions) == 0 || actions == uint64(len(result.Actions))) && frames == uint64(expectedFrames) &&
		ratings == uint64(len(result.PlayerRatings)) && clubFacts == 2 && playerFacts == uint64(len(result.PlayerRatings)), nil
}

func clearClickHouseMatch(ctx context.Context, conn driver.Conn, runID string, matchID string) error {
	for _, tableName := range []string{"touchline_matches_v2", "touchline_club_round_metrics", "touchline_match_club_facts_v2", "touchline_match_events_v2", "touchline_match_actions_v2", "touchline_player_frames_v2", "touchline_player_ratings_v2", "touchline_player_match_facts_v2"} {
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

func (s *Store) actionInsights(ctx context.Context, runID string, clubID string) (ActionInsights, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := ensureClickHouseSchema(ctx, s.clickhouse); err != nil {
		return ActionInsights{}, err
	}

	insights := ActionInsights{
		RunID:          runID,
		ClubID:         clubID,
		ActionMix:      []ActionMixRow{},
		PassNetwork:    []PassNetworkLink{},
		PlayerProfiles: []PlayerActionProfile{},
	}
	if err := s.clickhouse.QueryRow(ctx, `
		SELECT
			uniqExact(match_id),
			count(),
			uniqExact(possession_id),
			countIf(action_type = 'pass'),
			countIf(action_type = 'pass' AND outcome = 'successful'),
			countIf(action_type = 'pass' AND outcome = 'successful' AND JSONExtractBool(qualifiers, 'progressive')),
			countIf(action_type = 'shot'),
			countIf(action_type = 'shot' AND JSONExtractBool(qualifiers, 'onTarget')),
			sumIf(toFloat64OrZero(JSONExtractRaw(qualifiers, 'xg')), action_type = 'shot'),
			countIf(action_type = 'carry'),
			countIf(action_type = 'carry' AND outcome = 'successful'),
			countIf(start_x >= 60)
		FROM touchline_match_actions_v2
		WHERE run_id = ? AND team_id = ?`, runID, clubID).Scan(
		&insights.Matches,
		&insights.Actions,
		&insights.Possessions,
		&insights.Passes,
		&insights.CompletedPasses,
		&insights.ProgressivePasses,
		&insights.Shots,
		&insights.ShotsOnTarget,
		&insights.XG,
		&insights.Carries,
		&insights.SuccessfulCarries,
		&insights.FinalThirdActions,
	); err != nil {
		return ActionInsights{}, err
	}
	if insights.Passes > 0 {
		insights.PassCompletion = float64(insights.CompletedPasses) / float64(insights.Passes) * 100
	}

	mixRows, err := s.clickhouse.Query(ctx, `
		SELECT
			action_type,
			count(),
			countIf(outcome = 'successful'),
			round(toFloat64(countIf(outcome = 'successful')) / greatest(toFloat64(count()), 1) * 100, 1)
		FROM touchline_match_actions_v2
		WHERE run_id = ? AND team_id = ?
		GROUP BY action_type
		ORDER BY count() DESC
		LIMIT 12`, runID, clubID)
	if err != nil {
		return ActionInsights{}, err
	}
	for mixRows.Next() {
		var row ActionMixRow
		if err := mixRows.Scan(&row.ActionType, &row.Actions, &row.Successful, &row.SuccessRate); err != nil {
			mixRows.Close()
			return ActionInsights{}, err
		}
		insights.ActionMix = append(insights.ActionMix, row)
	}
	if err := mixRows.Err(); err != nil {
		mixRows.Close()
		return ActionInsights{}, err
	}
	mixRows.Close()

	networkRows, err := s.clickhouse.Query(ctx, `
		SELECT
			player_id,
			recipient_player_id,
			count(),
			countIf(outcome = 'successful'),
			countIf(JSONExtractBool(qualifiers, 'progressive')),
			round(toFloat64(countIf(outcome = 'successful')) / greatest(toFloat64(count()), 1) * 100, 1)
		FROM touchline_match_actions_v2
		WHERE run_id = ?
		  AND team_id = ?
		  AND action_type = 'pass'
		  AND recipient_player_id != ''
		GROUP BY player_id, recipient_player_id
		ORDER BY countIf(outcome = 'successful') DESC, count() DESC
		LIMIT 8`, runID, clubID)
	if err != nil {
		return ActionInsights{}, err
	}
	for networkRows.Next() {
		var row PassNetworkLink
		if err := networkRows.Scan(
			&row.PasserID,
			&row.ReceiverID,
			&row.Attempts,
			&row.Completions,
			&row.ProgressivePasses,
			&row.CompletionRate,
		); err != nil {
			networkRows.Close()
			return ActionInsights{}, err
		}
		insights.PassNetwork = append(insights.PassNetwork, row)
	}
	if err := networkRows.Err(); err != nil {
		networkRows.Close()
		return ActionInsights{}, err
	}
	networkRows.Close()

	profileRows, err := s.clickhouse.Query(ctx, `
		SELECT
			player_id,
			count(),
			countIf(action_type = 'pass'),
			countIf(action_type = 'pass' AND outcome = 'successful'),
			countIf(JSONExtractBool(qualifiers, 'progressive')),
			countIf(action_type = 'carry'),
			countIf(action_type = 'shot'),
			sumIf(toFloat64OrZero(JSONExtractRaw(qualifiers, 'xg')), action_type = 'shot'),
			countIf(action_type IN ('tackle', 'interception', 'duel', 'block', 'recovery', 'clearance')),
			countIf(JSONExtractString(qualifiers, 'phase') = 'build'),
			countIf(JSONExtractString(qualifiers, 'phase') = 'final-third'),
			countIf(JSONExtractString(qualifiers, 'phase') = 'box')
		FROM touchline_match_actions_v2
		WHERE run_id = ? AND team_id = ?
		GROUP BY player_id
		ORDER BY count() DESC
		LIMIT 40`, runID, clubID)
	if err != nil {
		return ActionInsights{}, err
	}
	for profileRows.Next() {
		var profile PlayerActionProfile
		if err := profileRows.Scan(
			&profile.PlayerID,
			&profile.Actions,
			&profile.Passes,
			&profile.CompletedPasses,
			&profile.ProgressiveActions,
			&profile.Carries,
			&profile.Shots,
			&profile.XG,
			&profile.DefensiveActions,
			&profile.BuildUpActions,
			&profile.FinalThirdActions,
			&profile.BoxActions,
		); err != nil {
			profileRows.Close()
			return ActionInsights{}, err
		}
		if profile.Passes > 0 {
			profile.CompletionRate = float64(profile.CompletedPasses) / float64(profile.Passes) * 100
		}
		profile.PrimaryRole = primaryActionRole(profile)
		insights.PlayerProfiles = append(insights.PlayerProfiles, profile)
	}
	if err := profileRows.Err(); err != nil {
		profileRows.Close()
		return ActionInsights{}, err
	}
	profileRows.Close()

	insights.AnalystNote = actionAnalystNote(insights)
	return insights, nil
}

func primaryActionRole(profile PlayerActionProfile) string {
	if profile.Shots >= 2 || (profile.XG >= 0.18 && profile.BoxActions > 0) {
		return "Box threat"
	}
	if profile.DefensiveActions >= 2 && profile.DefensiveActions >= profile.ProgressiveActions {
		return "Pressing winner"
	}
	if profile.ProgressiveActions >= 2 && profile.ProgressiveActions >= profile.Carries {
		return "Progressive creator"
	}
	if profile.Carries >= 2 {
		return "Progressive carrier"
	}
	if profile.BuildUpActions >= 2 && profile.Passes >= 2 {
		return "Build-up hub"
	}
	if profile.FinalThirdActions >= 2 {
		return "Final-third connector"
	}
	return "Connector"
}

func actionAnalystNote(insights ActionInsights) string {
	if insights.Actions == 0 {
		return "No structured actions have landed for this run yet. Play a matchday to start the action feed."
	}
	if insights.Passes > 0 && insights.PassCompletion < 72 {
		return "Build-up is breaking down frequently. Consider a safer passing option or a more measured tempo."
	}
	if insights.Passes > 0 && float64(insights.ProgressivePasses)/float64(insights.Passes) < 0.16 {
		return "The team is circulating the ball safely but rarely breaking lines. Add a progressive midfield or final-third outlet."
	}
	if insights.Shots > 0 && insights.XG/float64(insights.Shots) < 0.055 {
		return "Shot volume is arriving from low-value locations. Improve final-third access before increasing tempo."
	}
	return "The action profile shows a coherent attacking sequence. Use the pass network to identify the next role or recruitment upgrade."
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
		`CREATE TABLE IF NOT EXISTS touchline_runs_v2 (
			run_id String,
			career_id String,
			season UInt32,
			last_round UInt32,
			rounds_completed UInt32,
			matches_in_round UInt32,
			clubs_expected UInt32,
			status LowCardinality(String),
			schema_version UInt32,
			updated_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = ReplacingMergeTree(updated_at) ORDER BY (run_id, season)`,
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
		`CREATE TABLE IF NOT EXISTS touchline_match_club_facts_v2 (
			run_id String,
			match_id String,
			season UInt32,
			round UInt32,
			club_id String,
			opponent_id String,
			is_home Bool,
			goals_for UInt32,
			goals_against UInt32,
			xg_for Float32,
			shots UInt32,
			shots_on_target UInt32,
			possession Float32,
			pressure Float32,
			territory Float32,
			possessions UInt32,
			final_third_entries UInt32,
			box_entries UInt32,
			press_wins UInt32,
			build_up_fails UInt32,
			midfield_wins UInt32,
			line_breaks UInt32,
			balls_behind UInt32,
			counters UInt32,
			saves UInt32,
			cards UInt32,
			late_fatigue_losses UInt32,
			formation LowCardinality(String),
			mentality LowCardinality(String),
			pressing UInt32,
			tempo UInt32,
			defensive_line UInt32,
			ingested_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = MergeTree ORDER BY (run_id, season, round, match_id, club_id)`,
		`CREATE TABLE IF NOT EXISTS touchline_player_match_facts_v2 (
			run_id String,
			fact_key String,
			match_id String,
			season UInt32,
			round UInt32,
			player_id String,
			club_id String,
			opponent_id String,
			player_name String,
			position LowCardinality(String),
			started Bool,
			minutes_played UInt32,
			rating Float32,
			goals UInt32,
			shots UInt32,
			xg Float32,
			ingested_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = MergeTree ORDER BY (run_id, season, player_id, match_id)`,
		`CREATE TABLE IF NOT EXISTS touchline_club_round_metrics (
			run_id String,
			match_id String,
			season UInt32,
			round UInt32,
			club_id String,
			matches UInt64,
			goals_for UInt64,
			goals_against UInt64,
			xg_for Float64,
			xg_against Float64,
			shots_for UInt64,
			shots_against UInt64,
			possession Float64,
			pressure Float64,
			territory Float64,
			ingested_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = SummingMergeTree ORDER BY (run_id, season, club_id, round, match_id)`,
		`CREATE MATERIALIZED VIEW IF NOT EXISTS touchline_club_round_metrics_mv
		TO touchline_club_round_metrics
		AS SELECT
			run_id,
			match_id,
			season,
			round,
			arrayJoin([home_id, away_id]) AS club_id,
			toUInt64(1) AS matches,
			if(club_id = home_id, toUInt64(home_goals), toUInt64(away_goals)) AS goals_for,
			if(club_id = home_id, toUInt64(away_goals), toUInt64(home_goals)) AS goals_against,
			if(club_id = home_id, toFloat64(home_xg), toFloat64(away_xg)) AS xg_for,
			if(club_id = home_id, toFloat64(away_xg), toFloat64(home_xg)) AS xg_against,
			if(club_id = home_id, toUInt64(home_shots), toUInt64(away_shots)) AS shots_for,
			if(club_id = home_id, toUInt64(away_shots), toUInt64(home_shots)) AS shots_against,
			if(club_id = home_id, toFloat64(home_possession), 100.0 - toFloat64(home_possession)) AS possession,
			if(club_id = home_id, toFloat64(home_pressure), toFloat64(away_pressure)) AS pressure,
			if(club_id = home_id, toFloat64(home_territory), 100.0 - toFloat64(home_territory)) AS territory
		FROM touchline_matches_v2`,
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
		`CREATE TABLE IF NOT EXISTS touchline_match_actions_v2 (
			run_id String,
			action_id String,
			match_id String,
			season UInt32,
			round UInt32,
			sequence_id String,
			possession_id String,
			period UInt8,
			second UInt32,
			team_id String,
			player_id String,
			recipient_player_id String,
			action_type LowCardinality(String),
			outcome LowCardinality(String),
			start_x Float32,
			start_y Float32,
			end_x Float32,
			end_y Float32,
			qualifiers String,
			ingested_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = MergeTree ORDER BY (run_id, season, match_id, second, action_id)`,
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
	runs, err := conn.PrepareBatch(ctx, `INSERT INTO touchline_runs_v2 (
		run_id, career_id, season, last_round, rounds_completed, matches_in_round, clubs_expected, status, schema_version
	)`)
	if err != nil {
		return err
	}
	status := "in_progress"
	if payload.Round >= 37 {
		status = "complete"
	}
	if err := runs.Append(
		payload.RunID, careerIDFromRunID(payload.RunID), payload.Season, payload.Round, payload.Round+1,
		len(payload.Results), len(payload.Clubs), status, 3,
	); err != nil {
		return err
	}
	if err := runs.Send(); err != nil {
		return err
	}

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

	clubFacts, err := conn.PrepareBatch(ctx, `INSERT INTO touchline_match_club_facts_v2 (
		run_id, match_id, season, round, club_id, opponent_id, is_home, goals_for, goals_against, xg_for,
		shots, shots_on_target, possession, pressure, territory, possessions, final_third_entries, box_entries,
		press_wins, build_up_fails, midfield_wins, line_breaks, balls_behind, counters, saves, cards,
		late_fatigue_losses, formation, mentality, pressing, tempo, defensive_line
	)`)
	if err != nil {
		return err
	}
	for _, result := range payload.Results {
		for _, fact := range buildMatchClubFacts(result) {
			if err := clubFacts.Append(
				payload.RunID, result.FixtureID, payload.Season, result.Round, fact.ClubID, fact.OpponentID, fact.IsHome,
				fact.GoalsFor, fact.GoalsAgainst, fact.XGFor, fact.Shots, fact.ShotsOnTarget, fact.Possession,
				fact.Pressure, fact.Territory, fact.Possessions, fact.FinalThirdEntries, fact.BoxEntries, fact.PressWins,
				fact.BuildUpFails, fact.MidfieldWins, fact.LineBreaks, fact.BallsBehind, fact.Counters, fact.Saves,
				fact.Cards, fact.LateFatigueLosses, fact.Formation, fact.Mentality, fact.Pressing, fact.Tempo, fact.DefensiveLine,
			); err != nil {
				return err
			}
		}
	}
	if err := clubFacts.Send(); err != nil {
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

	actions, err := conn.PrepareBatch(ctx, `INSERT INTO touchline_match_actions_v2 (
		run_id, action_id, match_id, season, round, sequence_id, possession_id, period, second, team_id,
		player_id, recipient_player_id, action_type, outcome, start_x, start_y, end_x, end_y, qualifiers
	)`)
	if err != nil {
		return err
	}
	for _, result := range payload.Results {
		for index, action := range result.Actions {
			actionID := action.ID
			if actionID == "" {
				actionID = fmt.Sprintf("%s:action:%d", result.FixtureID, index)
			}
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
				payload.RunID, actionID, result.FixtureID, payload.Season, result.Round, action.SequenceID,
				action.PossessionID, action.Period, action.Second, action.TeamID, action.PlayerID,
				action.RecipientPlayerID, action.Type, action.Outcome, action.StartX, action.StartY, endX, endY,
				string(qualifiers),
			); err != nil {
				return err
			}
		}
	}
	if err := actions.Send(); err != nil {
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
	if err := ratings.Send(); err != nil {
		return err
	}

	playerFacts, err := conn.PrepareBatch(ctx, `INSERT INTO touchline_player_match_facts_v2 (
		run_id, fact_key, match_id, season, round, player_id, club_id, opponent_id, player_name, position,
		started, minutes_played, rating, goals, shots, xg
	)`)
	if err != nil {
		return err
	}
	for _, result := range payload.Results {
		for _, fact := range buildPlayerMatchFacts(payload.Players, result) {
			if err := playerFacts.Append(
				payload.RunID, fact.FactKey, result.FixtureID, payload.Season, result.Round, fact.PlayerID,
				fact.ClubID, fact.OpponentID, fact.PlayerName, fact.Position, fact.Started, fact.MinutesPlayed,
				fact.Rating, fact.Goals, fact.Shots, fact.XG,
			); err != nil {
				return err
			}
		}
	}
	return playerFacts.Send()
}

func (w *icebergWriter) ready(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.ensureCatalog(ctx)
}

func (w *icebergWriter) history(ctx context.Context, tableName string) (IcebergHistory, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.ensureCatalog(ctx); err != nil {
		return IcebergHistory{}, err
	}

	tbl, err := w.catalog.LoadTable(ctx, catalog.ToIdentifier("touchline", tableName))
	if err != nil {
		return IcebergHistory{}, err
	}
	history := IcebergHistory{Table: tableName, Snapshots: []IcebergSnapshot{}}
	if current := tbl.CurrentSnapshot(); current != nil {
		currentID := current.SnapshotID
		history.CurrentSnapshotID = &currentID
	}
	for _, snapshot := range tbl.Metadata().Snapshots() {
		item := IcebergSnapshot{
			SnapshotID:       snapshot.SnapshotID,
			ParentSnapshotID: snapshot.ParentSnapshotID,
			SequenceNumber:   snapshot.SequenceNumber,
			TimestampMs:      snapshot.TimestampMs,
			OccurredAt:       time.UnixMilli(snapshot.TimestampMs).UTC().Format(time.RFC3339Nano),
		}
		if snapshot.Summary != nil {
			item.Summary = snapshot.Summary.String()
		}
		history.Snapshots = append(history.Snapshots, item)
	}
	sort.Slice(history.Snapshots, func(left, right int) bool {
		return history.Snapshots[left].TimestampMs < history.Snapshots[right].TimestampMs
	})
	return history, nil
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
		{name: "runs", schema: runsIcebergSchema(), table: runsArrowTable(payload)},
		{name: "matches", schema: matchesIcebergSchema(), table: matchesArrowTable(payload)},
		{name: "match_club_facts", schema: matchClubFactsIcebergSchema(), table: matchClubFactsArrowTable(payload)},
		{name: "match_events", schema: eventsIcebergSchema(), table: eventsArrowTable(payload)},
		{name: "match_actions", schema: actionsIcebergSchema(), table: actionsArrowTable(payload)},
		{name: "player_frames", schema: framesIcebergSchema(), table: framesArrowTable(payload)},
		{name: "player_ratings", schema: ratingsIcebergSchema(), table: ratingsArrowTable(payload)},
		{name: "player_match_facts", schema: playerMatchFactsIcebergSchema(), table: playerMatchFactsArrowTable(payload)},
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

func runsArrowTable(payload MatchdayPayload) arrow.Table {
	schema := arrowSchema([]arrow.Field{
		stringField("run_id"), stringField("career_id"), intField("season"), intField("round"),
		intField("rounds_completed"), intField("matches_in_round"), intField("clubs_expected"),
		stringField("status"), intField("schema_version"),
	})
	builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	builder.Field(0).(*array.StringBuilder).Append(payload.RunID)
	builder.Field(1).(*array.StringBuilder).Append(careerIDFromRunID(payload.RunID))
	builder.Field(2).(*array.Int64Builder).Append(int64(payload.Season))
	builder.Field(3).(*array.Int64Builder).Append(int64(payload.Round))
	builder.Field(4).(*array.Int64Builder).Append(int64(payload.Round + 1))
	builder.Field(5).(*array.Int64Builder).Append(int64(len(payload.Results)))
	builder.Field(6).(*array.Int64Builder).Append(int64(len(payload.Clubs)))
	status := "in_progress"
	if payload.Round >= 37 {
		status = "complete"
	}
	builder.Field(7).(*array.StringBuilder).Append(status)
	builder.Field(8).(*array.Int64Builder).Append(2)
	return finishArrowTable(schema, builder)
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

func matchClubFactsArrowTable(payload MatchdayPayload) arrow.Table {
	schema := arrowSchema([]arrow.Field{
		stringField("run_id"), stringField("match_id"), intField("season"), intField("round"), stringField("club_id"),
		stringField("opponent_id"), intField("is_home"), intField("goals_for"), intField("goals_against"), floatField("xg_for"),
		intField("shots"), intField("shots_on_target"), floatField("possession"), floatField("pressure"), floatField("territory"),
		intField("possessions"), intField("final_third_entries"), intField("box_entries"), intField("press_wins"), intField("build_up_fails"),
		intField("midfield_wins"), intField("line_breaks"), intField("balls_behind"), intField("counters"), intField("saves"), intField("cards"),
		intField("late_fatigue_losses"), stringField("formation"), stringField("mentality"), intField("pressing"), intField("tempo"), intField("defensive_line"),
	})
	builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	for _, result := range payload.Results {
		for _, fact := range buildMatchClubFacts(result) {
			isHome := int64(0)
			if fact.IsHome {
				isHome = 1
			}
			builder.Field(0).(*array.StringBuilder).Append(payload.RunID)
			builder.Field(1).(*array.StringBuilder).Append(result.FixtureID)
			builder.Field(2).(*array.Int64Builder).Append(int64(payload.Season))
			builder.Field(3).(*array.Int64Builder).Append(int64(result.Round))
			builder.Field(4).(*array.StringBuilder).Append(fact.ClubID)
			builder.Field(5).(*array.StringBuilder).Append(fact.OpponentID)
			builder.Field(6).(*array.Int64Builder).Append(isHome)
			builder.Field(7).(*array.Int64Builder).Append(int64(fact.GoalsFor))
			builder.Field(8).(*array.Int64Builder).Append(int64(fact.GoalsAgainst))
			builder.Field(9).(*array.Float64Builder).Append(fact.XGFor)
			builder.Field(10).(*array.Int64Builder).Append(int64(fact.Shots))
			builder.Field(11).(*array.Int64Builder).Append(int64(fact.ShotsOnTarget))
			builder.Field(12).(*array.Float64Builder).Append(fact.Possession)
			builder.Field(13).(*array.Float64Builder).Append(fact.Pressure)
			builder.Field(14).(*array.Float64Builder).Append(fact.Territory)
			builder.Field(15).(*array.Int64Builder).Append(int64(fact.Possessions))
			builder.Field(16).(*array.Int64Builder).Append(int64(fact.FinalThirdEntries))
			builder.Field(17).(*array.Int64Builder).Append(int64(fact.BoxEntries))
			builder.Field(18).(*array.Int64Builder).Append(int64(fact.PressWins))
			builder.Field(19).(*array.Int64Builder).Append(int64(fact.BuildUpFails))
			builder.Field(20).(*array.Int64Builder).Append(int64(fact.MidfieldWins))
			builder.Field(21).(*array.Int64Builder).Append(int64(fact.LineBreaks))
			builder.Field(22).(*array.Int64Builder).Append(int64(fact.BallsBehind))
			builder.Field(23).(*array.Int64Builder).Append(int64(fact.Counters))
			builder.Field(24).(*array.Int64Builder).Append(int64(fact.Saves))
			builder.Field(25).(*array.Int64Builder).Append(int64(fact.Cards))
			builder.Field(26).(*array.Int64Builder).Append(int64(fact.LateFatigueLosses))
			builder.Field(27).(*array.StringBuilder).Append(fact.Formation)
			builder.Field(28).(*array.StringBuilder).Append(fact.Mentality)
			builder.Field(29).(*array.Int64Builder).Append(int64(fact.Pressing))
			builder.Field(30).(*array.Int64Builder).Append(int64(fact.Tempo))
			builder.Field(31).(*array.Int64Builder).Append(int64(fact.DefensiveLine))
		}
	}
	return finishArrowTable(schema, builder)
}

func playerMatchFactsArrowTable(payload MatchdayPayload) arrow.Table {
	schema := arrowSchema([]arrow.Field{
		stringField("run_id"), stringField("fact_key"), stringField("match_id"), intField("season"), intField("round"),
		stringField("player_id"), stringField("club_id"), stringField("opponent_id"), stringField("player_name"), stringField("position"),
		intField("started"), intField("minutes_played"), floatField("rating"), intField("goals"), intField("shots"), floatField("xg"),
	})
	builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	for _, result := range payload.Results {
		for _, fact := range buildPlayerMatchFacts(payload.Players, result) {
			started := int64(0)
			if fact.Started {
				started = 1
			}
			builder.Field(0).(*array.StringBuilder).Append(payload.RunID)
			builder.Field(1).(*array.StringBuilder).Append(fact.FactKey)
			builder.Field(2).(*array.StringBuilder).Append(result.FixtureID)
			builder.Field(3).(*array.Int64Builder).Append(int64(payload.Season))
			builder.Field(4).(*array.Int64Builder).Append(int64(result.Round))
			builder.Field(5).(*array.StringBuilder).Append(fact.PlayerID)
			builder.Field(6).(*array.StringBuilder).Append(fact.ClubID)
			builder.Field(7).(*array.StringBuilder).Append(fact.OpponentID)
			builder.Field(8).(*array.StringBuilder).Append(fact.PlayerName)
			builder.Field(9).(*array.StringBuilder).Append(fact.Position)
			builder.Field(10).(*array.Int64Builder).Append(started)
			builder.Field(11).(*array.Int64Builder).Append(int64(fact.MinutesPlayed))
			builder.Field(12).(*array.Float64Builder).Append(fact.Rating)
			builder.Field(13).(*array.Int64Builder).Append(int64(fact.Goals))
			builder.Field(14).(*array.Int64Builder).Append(int64(fact.Shots))
			builder.Field(15).(*array.Float64Builder).Append(fact.XG)
		}
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

func actionsArrowTable(payload MatchdayPayload) arrow.Table {
	schema := arrowSchema([]arrow.Field{
		stringField("run_id"), stringField("action_id"), stringField("match_id"), intField("season"), intField("round"),
		stringField("sequence_id"), stringField("possession_id"), intField("period"), intField("second"), stringField("team_id"),
		stringField("player_id"), stringField("recipient_player_id"), stringField("action_type"), stringField("outcome"),
		floatField("start_x"), floatField("start_y"), floatField("end_x"), floatField("end_y"), stringField("qualifiers"),
	})
	builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	for _, result := range payload.Results {
		for index, action := range result.Actions {
			actionID := action.ID
			if actionID == "" {
				actionID = fmt.Sprintf("%s:action:%d", result.FixtureID, index)
			}
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
			builder.Field(0).(*array.StringBuilder).Append(payload.RunID)
			builder.Field(1).(*array.StringBuilder).Append(actionID)
			builder.Field(2).(*array.StringBuilder).Append(result.FixtureID)
			builder.Field(3).(*array.Int64Builder).Append(int64(payload.Season))
			builder.Field(4).(*array.Int64Builder).Append(int64(result.Round))
			builder.Field(5).(*array.StringBuilder).Append(action.SequenceID)
			builder.Field(6).(*array.StringBuilder).Append(action.PossessionID)
			builder.Field(7).(*array.Int64Builder).Append(int64(action.Period))
			builder.Field(8).(*array.Int64Builder).Append(int64(action.Second))
			builder.Field(9).(*array.StringBuilder).Append(action.TeamID)
			builder.Field(10).(*array.StringBuilder).Append(action.PlayerID)
			builder.Field(11).(*array.StringBuilder).Append(action.RecipientPlayerID)
			builder.Field(12).(*array.StringBuilder).Append(action.Type)
			builder.Field(13).(*array.StringBuilder).Append(action.Outcome)
			builder.Field(14).(*array.Float64Builder).Append(action.StartX)
			builder.Field(15).(*array.Float64Builder).Append(action.StartY)
			builder.Field(16).(*array.Float64Builder).Append(endX)
			builder.Field(17).(*array.Float64Builder).Append(endY)
			builder.Field(18).(*array.StringBuilder).Append(string(qualifiers))
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

func runsIcebergSchema() *iceberg.Schema {
	return icebergSchema([]iceberg.NestedField{
		{ID: 1, Name: "run_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 2, Name: "career_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 3, Name: "season", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 4, Name: "round", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 5, Name: "rounds_completed", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 6, Name: "matches_in_round", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 7, Name: "clubs_expected", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 8, Name: "status", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 9, Name: "schema_version", Type: iceberg.PrimitiveTypes.Int64, Required: true},
	})
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

func matchClubFactsIcebergSchema() *iceberg.Schema {
	return icebergSchema([]iceberg.NestedField{
		{ID: 1, Name: "run_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 2, Name: "match_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 3, Name: "season", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 4, Name: "round", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 5, Name: "club_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 6, Name: "opponent_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 7, Name: "is_home", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 8, Name: "goals_for", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 9, Name: "goals_against", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 10, Name: "xg_for", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 11, Name: "shots", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 12, Name: "shots_on_target", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 13, Name: "possession", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 14, Name: "pressure", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 15, Name: "territory", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 16, Name: "possessions", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 17, Name: "final_third_entries", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 18, Name: "box_entries", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 19, Name: "press_wins", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 20, Name: "build_up_fails", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 21, Name: "midfield_wins", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 22, Name: "line_breaks", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 23, Name: "balls_behind", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 24, Name: "counters", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 25, Name: "saves", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 26, Name: "cards", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 27, Name: "late_fatigue_losses", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 28, Name: "formation", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 29, Name: "mentality", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 30, Name: "pressing", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 31, Name: "tempo", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 32, Name: "defensive_line", Type: iceberg.PrimitiveTypes.Int64, Required: true},
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

func actionsIcebergSchema() *iceberg.Schema {
	return icebergSchema([]iceberg.NestedField{
		{ID: 1, Name: "run_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 2, Name: "action_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 3, Name: "match_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 4, Name: "season", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 5, Name: "round", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 6, Name: "sequence_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 7, Name: "possession_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 8, Name: "period", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 9, Name: "second", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 10, Name: "team_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 11, Name: "player_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 12, Name: "recipient_player_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 13, Name: "action_type", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 14, Name: "outcome", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 15, Name: "start_x", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 16, Name: "start_y", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 17, Name: "end_x", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 18, Name: "end_y", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 19, Name: "qualifiers", Type: iceberg.PrimitiveTypes.String, Required: true},
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

func playerMatchFactsIcebergSchema() *iceberg.Schema {
	return icebergSchema([]iceberg.NestedField{
		{ID: 1, Name: "run_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 2, Name: "fact_key", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 3, Name: "match_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 4, Name: "season", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 5, Name: "round", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 6, Name: "player_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 7, Name: "club_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 8, Name: "opponent_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 9, Name: "player_name", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 10, Name: "position", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 11, Name: "started", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 12, Name: "minutes_played", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 13, Name: "rating", Type: iceberg.PrimitiveTypes.Float64, Required: true},
		{ID: 14, Name: "goals", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 15, Name: "shots", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 16, Name: "xg", Type: iceberg.PrimitiveTypes.Float64, Required: true},
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

type matchClubFact struct {
	ClubID            string
	OpponentID        string
	IsHome            bool
	GoalsFor          int
	GoalsAgainst      int
	XGFor             float64
	Shots             int
	ShotsOnTarget     int
	Possession        float64
	Pressure          float64
	Territory         float64
	Possessions       int
	FinalThirdEntries int
	BoxEntries        int
	PressWins         int
	BuildUpFails      int
	MidfieldWins      int
	LineBreaks        int
	BallsBehind       int
	Counters          int
	Saves             int
	Cards             int
	LateFatigueLosses int
	Formation         string
	Mentality         string
	Pressing          int
	Tempo             int
	DefensiveLine     int
}

type playerMatchFact struct {
	FactKey       string
	PlayerID      string
	ClubID        string
	OpponentID    string
	PlayerName    string
	Position      string
	Started       bool
	MinutesPlayed int
	Rating        float64
	Goals         int
	Shots         int
	XG            float64
}

func buildMatchClubFacts(result MatchResult) []matchClubFact {
	m := result.Metrics
	return []matchClubFact{
		{
			ClubID: result.HomeID, OpponentID: result.AwayID, IsHome: true,
			GoalsFor: result.HomeGoals, GoalsAgainst: result.AwayGoals, XGFor: m.HomeXG,
			Shots: m.HomeShots, ShotsOnTarget: m.HomeShotsOnTarget, Possession: m.HomePossession,
			Pressure: m.HomePressure, Territory: m.HomeTerritory, Possessions: m.HomePossessions,
			FinalThirdEntries: m.HomeFinalThird, BoxEntries: m.HomeBoxEntries, PressWins: m.HomePressWins,
			BuildUpFails: m.HomeBuildUpFails, MidfieldWins: m.HomeMidfieldWins, LineBreaks: m.HomeLineBreaks,
			BallsBehind: m.HomeBallsBehind, Counters: m.HomeCounters, Saves: m.HomeSaves, Cards: m.HomeCards,
			LateFatigueLosses: m.HomeFatigueLosses, Formation: result.Tactics.Home.Formation,
			Mentality: result.Tactics.Home.Mentality, Pressing: result.Tactics.Home.Pressing,
			Tempo: result.Tactics.Home.Tempo, DefensiveLine: result.Tactics.Home.DefensiveLine,
		},
		{
			ClubID: result.AwayID, OpponentID: result.HomeID, IsHome: false,
			GoalsFor: result.AwayGoals, GoalsAgainst: result.HomeGoals, XGFor: m.AwayXG,
			Shots: m.AwayShots, ShotsOnTarget: m.AwayShotsOnTarget, Possession: 100 - m.HomePossession,
			Pressure: m.AwayPressure, Territory: 100 - m.HomeTerritory, Possessions: m.AwayPossessions,
			FinalThirdEntries: m.AwayFinalThird, BoxEntries: m.AwayBoxEntries, PressWins: m.AwayPressWins,
			BuildUpFails: m.AwayBuildUpFails, MidfieldWins: m.AwayMidfieldWins, LineBreaks: m.AwayLineBreaks,
			BallsBehind: m.AwayBallsBehind, Counters: m.AwayCounters, Saves: m.AwaySaves, Cards: m.AwayCards,
			LateFatigueLosses: m.AwayFatigueLosses, Formation: result.Tactics.Away.Formation,
			Mentality: result.Tactics.Away.Mentality, Pressing: result.Tactics.Away.Pressing,
			Tempo: result.Tactics.Away.Tempo, DefensiveLine: result.Tactics.Away.DefensiveLine,
		},
	}
}

func buildPlayerMatchFacts(players []Player, result MatchResult) []playerMatchFact {
	playerIDs := make([]string, 0, len(result.PlayerRatings))
	for playerID := range result.PlayerRatings {
		playerIDs = append(playerIDs, playerID)
	}
	sort.Strings(playerIDs)

	type eventStats struct {
		goals int
		shots int
		xg    float64
	}
	stats := make(map[string]eventStats)
	index := playerIndex(players)
	for _, event := range result.Events {
		playerID := index[event.TeamID+"\x00"+event.PlayerName]
		if playerID == "" {
			continue
		}
		current := stats[playerID]
		if event.Type == "goal" || event.Type == "save" || event.Type == "chance" {
			current.shots++
		}
		if event.Type == "goal" {
			current.goals++
		}
		if event.XG != nil {
			current.xg += *event.XG
		}
		stats[playerID] = current
	}

	facts := make([]playerMatchFact, 0, len(playerIDs))
	for _, playerID := range playerIDs {
		player, ok := playerByID(players, playerID)
		if !ok {
			continue
		}
		opponentID := result.AwayID
		if player.ClubID == result.AwayID {
			opponentID = result.HomeID
		}
		eventStats := stats[playerID]
		matchStats := PlayerMatchStats{
			Started:       true,
			MinutesPlayed: 90,
			Goals:         eventStats.goals,
			Shots:         eventStats.shots,
			XG:            eventStats.xg,
		}
		if persisted, ok := result.PlayerStats[playerID]; ok {
			matchStats = persisted
		}
		facts = append(facts, playerMatchFact{
			FactKey:       fmt.Sprintf("%s:player:%s", result.FixtureID, playerID),
			PlayerID:      playerID,
			ClubID:        player.ClubID,
			OpponentID:    opponentID,
			PlayerName:    player.Name,
			Position:      player.Position,
			Started:       matchStats.Started,
			MinutesPlayed: matchStats.MinutesPlayed,
			Rating:        result.PlayerRatings[playerID],
			Goals:         matchStats.Goals,
			Shots:         matchStats.Shots,
			XG:            matchStats.XG,
		})
	}
	return facts
}

func careerIDFromRunID(runID string) string {
	if index := strings.Index(runID, ":season:"); index >= 0 {
		return runID[:index]
	}
	return runID
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
		if len(result.Events) > 128 || len(result.Actions) > 1200 || len(result.Trace) > 600 || len(result.PlayerRatings) > 40 {
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
