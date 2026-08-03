package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	iceberg "github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	"github.com/apache/iceberg-go/table"
)

type WorldCup2026Payload struct {
	Source        string                     `json:"source"`
	SourceVersion string                     `json:"sourceVersion"`
	Tournament    string                     `json:"tournament"`
	Matches       []WorldCup2026MatchPayload `json:"matches"`
}

type WorldCup2026MatchPayload struct {
	SourceMatchID       string                    `json:"sourceMatchId"`
	MatchNumber         int                       `json:"matchNumber"`
	Round               string                    `json:"round"`
	GroupName           string                    `json:"groupName"`
	MatchDate           string                    `json:"matchDate"`
	KickoffTime         string                    `json:"kickoffTime"`
	Venue               string                    `json:"venue"`
	HomeTeam            string                    `json:"homeTeam"`
	AwayTeam            string                    `json:"awayTeam"`
	RegulationHomeGoals int                       `json:"regulationHomeGoals"`
	RegulationAwayGoals int                       `json:"regulationAwayGoals"`
	HomeGoals           int                       `json:"homeGoals"`
	AwayGoals           int                       `json:"awayGoals"`
	ShootoutHomeGoals   int                       `json:"shootoutHomeGoals"`
	ShootoutAwayGoals   int                       `json:"shootoutAwayGoals"`
	Goals               []WorldCup2026GoalPayload `json:"goals"`
}

type WorldCup2026GoalPayload struct {
	TeamName    string `json:"teamName"`
	PlayerName  string `json:"playerName"`
	Minute      string `json:"minute"`
	MinuteValue int    `json:"minuteValue"`
	IsPenalty   bool   `json:"isPenalty"`
	IsOwnGoal   bool   `json:"isOwnGoal"`
}

type WorldCup2026MatchSummary struct {
	SourceMatchID       string             `json:"sourceMatchId"`
	MatchNumber         int                `json:"matchNumber"`
	Round               string             `json:"round"`
	GroupName           string             `json:"groupName"`
	MatchDate           string             `json:"matchDate"`
	KickoffTime         string             `json:"kickoffTime"`
	Venue               string             `json:"venue"`
	HomeTeam            string             `json:"homeTeam"`
	AwayTeam            string             `json:"awayTeam"`
	RegulationHomeGoals int                `json:"regulationHomeGoals"`
	RegulationAwayGoals int                `json:"regulationAwayGoals"`
	HomeGoals           int                `json:"homeGoals"`
	AwayGoals           int                `json:"awayGoals"`
	ShootoutHomeGoals   int                `json:"shootoutHomeGoals"`
	ShootoutAwayGoals   int                `json:"shootoutAwayGoals"`
	PenaltyShootout     bool               `json:"penaltyShootout"`
	Winner              string             `json:"winner"`
	Goals               []WorldCup2026Goal `json:"goals"`
}

type WorldCup2026Goal struct {
	MatchNumber int    `json:"matchNumber,omitempty"`
	TeamName    string `json:"teamName"`
	PlayerName  string `json:"playerName"`
	Minute      string `json:"minute"`
	MinuteValue int    `json:"minuteValue"`
	IsPenalty   bool   `json:"isPenalty"`
	IsOwnGoal   bool   `json:"isOwnGoal"`
}

type WorldCup2026Summary struct {
	Matches          int     `json:"matches"`
	Teams            int     `json:"teams"`
	Goals            int     `json:"goals"`
	AverageGoals     float64 `json:"averageGoals"`
	Venues           int     `json:"venues"`
	PenaltyShootouts int     `json:"penaltyShootouts"`
	ExtraTimeMatches int     `json:"extraTimeMatches"`
	Champion         string  `json:"champion"`
	RunnerUp         string  `json:"runnerUp"`
	FinalScore       string  `json:"finalScore"`
}

type WorldCup2026TeamRow struct {
	TeamName       string `json:"teamName"`
	GroupName      string `json:"groupName"`
	Rank           int    `json:"rank"`
	Played         int    `json:"played"`
	Won            int    `json:"won"`
	Drawn          int    `json:"drawn"`
	Lost           int    `json:"lost"`
	GoalsFor       int    `json:"goalsFor"`
	GoalsAgainst   int    `json:"goalsAgainst"`
	GoalDifference int    `json:"goalDifference"`
	Points         int    `json:"points"`
	Stage          string `json:"stage"`
}

type WorldCup2026Scorer struct {
	PlayerName   string `json:"playerName"`
	TeamName     string `json:"teamName"`
	Goals        int    `json:"goals"`
	PenaltyGoals int    `json:"penaltyGoals"`
	Matches      int    `json:"matches"`
}

type WorldCup2026TimingBucket struct {
	Label string `json:"label"`
	Goals int    `json:"goals"`
}

type WorldCup2026VenueRow struct {
	Venue        string  `json:"venue"`
	Matches      int     `json:"matches"`
	Goals        int     `json:"goals"`
	AverageGoals float64 `json:"averageGoals"`
}

type WorldCup2026Overview struct {
	Source        string                     `json:"source"`
	SourceVersion string                     `json:"sourceVersion"`
	Tournament    string                     `json:"tournament"`
	Summary       WorldCup2026Summary        `json:"summary"`
	Teams         []WorldCup2026TeamRow      `json:"teams"`
	TopScorers    []WorldCup2026Scorer       `json:"topScorers"`
	GoalTiming    []WorldCup2026TimingBucket `json:"goalTiming"`
	Venues        []WorldCup2026VenueRow     `json:"venues"`
	Matches       []WorldCup2026MatchSummary `json:"matches"`
}

type worldCup2026TeamAccumulator struct {
	WorldCup2026TeamRow
	stageRank int
}

type worldCup2026ScorerAccumulator struct {
	WorldCup2026Scorer
	matches map[int]struct{}
}

func (s *Store) handleWorldCup2026Import(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}

	var payload WorldCup2026Payload
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<20))
	if err := decoder.Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid World Cup payload"})
		return
	}
	if err := validateWorldCup2026Payload(payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := s.ingestWorldCup2026(r.Context(), payload); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "World Cup stores unavailable"})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"accepted": true,
		"source":   payload.Source,
		"matches":  len(payload.Matches),
		"goals":    countWorldCup2026Goals(payload),
	})
}

func (s *Store) handleWorldCup2026Overview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}
	overview, err := s.worldCup2026Overview(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "World Cup store unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, overview)
}

func validateWorldCup2026Payload(payload WorldCup2026Payload) error {
	if payload.Source == "" || len(payload.Source) > 80 {
		return fmt.Errorf("source is required and must be at most 80 characters")
	}
	if payload.SourceVersion == "" || len(payload.SourceVersion) > 160 {
		return fmt.Errorf("sourceVersion is required and must be at most 160 characters")
	}
	if payload.Tournament == "" || len(payload.Tournament) > 100 {
		return fmt.Errorf("tournament is required and must be at most 100 characters")
	}
	if len(payload.Matches) == 0 || len(payload.Matches) > 120 {
		return fmt.Errorf("World Cup import must contain between 1 and 120 matches")
	}
	seen := map[int]struct{}{}
	for _, match := range payload.Matches {
		if match.MatchNumber < 1 || match.MatchNumber > 120 {
			return fmt.Errorf("match number is outside the supported range")
		}
		if _, exists := seen[match.MatchNumber]; exists {
			return fmt.Errorf("duplicate match number %d", match.MatchNumber)
		}
		seen[match.MatchNumber] = struct{}{}
		if match.SourceMatchID == "" || match.HomeTeam == "" || match.AwayTeam == "" || match.MatchDate == "" {
			return fmt.Errorf("match metadata is incomplete")
		}
		if match.HomeGoals < 0 || match.AwayGoals < 0 || match.RegulationHomeGoals < 0 || match.RegulationAwayGoals < 0 {
			return fmt.Errorf("match scores cannot be negative")
		}
		if len(match.Goals) > 30 {
			return fmt.Errorf("match goal count exceeds the safety limit")
		}
		for _, goal := range match.Goals {
			if goal.TeamName == "" || goal.PlayerName == "" || goal.Minute == "" || goal.MinuteValue < 0 || goal.MinuteValue > 140 {
				return fmt.Errorf("goal metadata is incomplete")
			}
		}
	}
	return nil
}

func countWorldCup2026Goals(payload WorldCup2026Payload) int {
	count := 0
	for _, match := range payload.Matches {
		count += len(match.Goals)
	}
	return count
}

func (s *Store) ingestWorldCup2026(ctx context.Context, payload WorldCup2026Payload) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	if err := ensureWorldCup2026Schema(ctx, s.clickhouse); err != nil {
		return err
	}
	for _, tableName := range []string{"touchline_worldcup_2026_matches_v1", "touchline_worldcup_2026_goals_v1"} {
		statement := fmt.Sprintf("ALTER TABLE %s DELETE WHERE source = ? AND tournament = ? SETTINGS mutations_sync = 2", tableName)
		if err := s.clickhouse.Exec(ctx, statement, payload.Source, payload.Tournament); err != nil {
			return err
		}
	}
	if err := insertWorldCup2026ClickHouse(ctx, s.clickhouse, payload); err != nil {
		return err
	}
	return s.iceberg.appendWorldCup2026(ctx, payload)
}

func ensureWorldCup2026Schema(ctx context.Context, conn driver.Conn) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS touchline_worldcup_2026_matches_v1 (
			source LowCardinality(String),
			source_version String,
			tournament LowCardinality(String),
			source_match_id String,
			match_number UInt16,
			round LowCardinality(String),
			group_name LowCardinality(String),
			match_date Date,
			kickoff_time String,
			venue LowCardinality(String),
			home_team LowCardinality(String),
			away_team LowCardinality(String),
			regulation_home_goals UInt8,
			regulation_away_goals UInt8,
			home_goals UInt8,
			away_goals UInt8,
			shootout_home_goals UInt8,
			shootout_away_goals UInt8,
			imported_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = MergeTree ORDER BY (tournament, match_number)`,
		`CREATE TABLE IF NOT EXISTS touchline_worldcup_2026_goals_v1 (
			source LowCardinality(String),
			source_version String,
			tournament LowCardinality(String),
			match_number UInt16,
			team_name LowCardinality(String),
			player_name String,
			minute String,
			minute_value UInt16,
			is_penalty UInt8,
			is_own_goal UInt8,
			imported_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = MergeTree ORDER BY (tournament, match_number, minute_value, player_name)`,
	}
	for _, statement := range statements {
		if err := conn.Exec(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func insertWorldCup2026ClickHouse(ctx context.Context, conn driver.Conn, payload WorldCup2026Payload) error {
	matches, err := conn.PrepareBatch(ctx, `INSERT INTO touchline_worldcup_2026_matches_v1 (
		source, source_version, tournament, source_match_id, match_number, round, group_name, match_date,
		kickoff_time, venue, home_team, away_team, regulation_home_goals, regulation_away_goals,
		home_goals, away_goals, shootout_home_goals, shootout_away_goals
	)`)
	if err != nil {
		return err
	}
	for _, match := range payload.Matches {
		matchDate, err := time.Parse("2006-01-02", match.MatchDate)
		if err != nil {
			return err
		}
		if err := matches.Append(
			payload.Source, payload.SourceVersion, payload.Tournament, match.SourceMatchID, uint16(match.MatchNumber),
			match.Round, match.GroupName, matchDate, match.KickoffTime, match.Venue, match.HomeTeam, match.AwayTeam,
			uint8(match.RegulationHomeGoals), uint8(match.RegulationAwayGoals), uint8(match.HomeGoals), uint8(match.AwayGoals),
			uint8(match.ShootoutHomeGoals), uint8(match.ShootoutAwayGoals),
		); err != nil {
			return err
		}
	}
	if err := matches.Send(); err != nil {
		return err
	}

	goals, err := conn.PrepareBatch(ctx, `INSERT INTO touchline_worldcup_2026_goals_v1 (
		source, source_version, tournament, match_number, team_name, player_name, minute, minute_value, is_penalty, is_own_goal
	)`)
	if err != nil {
		return err
	}
	for _, match := range payload.Matches {
		for _, goal := range match.Goals {
			if err := goals.Append(
				payload.Source, payload.SourceVersion, payload.Tournament, uint16(match.MatchNumber), goal.TeamName,
				goal.PlayerName, goal.Minute, uint16(goal.MinuteValue), boolToUInt8(goal.IsPenalty), boolToUInt8(goal.IsOwnGoal),
			); err != nil {
				return err
			}
		}
	}
	return goals.Send()
}

func boolToUInt8(value bool) uint8 {
	if value {
		return 1
	}
	return 0
}

func (s *Store) worldCup2026Overview(ctx context.Context) (WorldCup2026Overview, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := ensureWorldCup2026Schema(ctx, s.clickhouse); err != nil {
		return WorldCup2026Overview{}, err
	}
	const source = "openfootball"
	const tournament = "world-cup-2026"

	matchRows, err := s.clickhouse.Query(ctx, `
		SELECT source_match_id, match_number, round, group_name, toString(match_date), kickoff_time, venue,
			home_team, away_team, regulation_home_goals, regulation_away_goals, home_goals, away_goals,
			shootout_home_goals, shootout_away_goals
		FROM touchline_worldcup_2026_matches_v1
		WHERE source = ? AND tournament = ?
		ORDER BY match_number`, source, tournament)
	if err != nil {
		return WorldCup2026Overview{}, err
	}
	defer matchRows.Close()
	matches := make([]WorldCup2026MatchSummary, 0)
	for matchRows.Next() {
		var match WorldCup2026MatchSummary
		var matchNumber uint16
		var regulationHome, regulationAway, homeGoals, awayGoals, shootoutHome, shootoutAway uint8
		if err := matchRows.Scan(
			&match.SourceMatchID, &matchNumber, &match.Round, &match.GroupName, &match.MatchDate, &match.KickoffTime,
			&match.Venue, &match.HomeTeam, &match.AwayTeam, &regulationHome, &regulationAway, &homeGoals,
			&awayGoals, &shootoutHome, &shootoutAway,
		); err != nil {
			return WorldCup2026Overview{}, err
		}
		match.MatchNumber = int(matchNumber)
		match.RegulationHomeGoals = int(regulationHome)
		match.RegulationAwayGoals = int(regulationAway)
		match.HomeGoals = int(homeGoals)
		match.AwayGoals = int(awayGoals)
		match.ShootoutHomeGoals = int(shootoutHome)
		match.ShootoutAwayGoals = int(shootoutAway)
		match.PenaltyShootout = shootoutHome != 0 || shootoutAway != 0
		match.Winner = worldCup2026Winner(match)
		matches = append(matches, match)
	}
	if err := matchRows.Err(); err != nil {
		return WorldCup2026Overview{}, err
	}

	goalRows, err := s.clickhouse.Query(ctx, `
		SELECT match_number, team_name, player_name, minute, minute_value, is_penalty, is_own_goal
		FROM touchline_worldcup_2026_goals_v1
		WHERE source = ? AND tournament = ?
		ORDER BY match_number, minute_value, player_name`, source, tournament)
	if err != nil {
		return WorldCup2026Overview{}, err
	}
	defer goalRows.Close()
	goalsByMatch := map[int][]WorldCup2026Goal{}
	allGoals := make([]WorldCup2026Goal, 0)
	for goalRows.Next() {
		var matchNumber, minuteValue uint16
		var penalty, ownGoal uint8
		var goal WorldCup2026Goal
		if err := goalRows.Scan(&matchNumber, &goal.TeamName, &goal.PlayerName, &goal.Minute, &minuteValue, &penalty, &ownGoal); err != nil {
			return WorldCup2026Overview{}, err
		}
		goal.MatchNumber = int(matchNumber)
		goal.MinuteValue = int(minuteValue)
		goal.IsPenalty = penalty == 1
		goal.IsOwnGoal = ownGoal == 1
		goalsByMatch[int(matchNumber)] = append(goalsByMatch[int(matchNumber)], goal)
		allGoals = append(allGoals, goal)
	}
	if err := goalRows.Err(); err != nil {
		return WorldCup2026Overview{}, err
	}
	for index := range matches {
		matches[index].Goals = goalsByMatch[matches[index].MatchNumber]
	}

	overview := buildWorldCup2026Overview(matches, allGoals)
	overview.Source = source
	overview.SourceVersion = "openfootball/worldcup.json@2026"
	overview.Tournament = tournament
	return overview, nil
}

func worldCup2026Winner(match WorldCup2026MatchSummary) string {
	if match.HomeGoals > match.AwayGoals {
		return match.HomeTeam
	}
	if match.AwayGoals > match.HomeGoals {
		return match.AwayTeam
	}
	if match.ShootoutHomeGoals > match.ShootoutAwayGoals {
		return match.HomeTeam
	}
	if match.ShootoutAwayGoals > match.ShootoutHomeGoals {
		return match.AwayTeam
	}
	return ""
}

func buildWorldCup2026Overview(matches []WorldCup2026MatchSummary, goals []WorldCup2026Goal) WorldCup2026Overview {
	teamMap := map[string]*worldCup2026TeamAccumulator{}
	venueMap := map[string]*WorldCup2026VenueRow{}
	scorerMap := map[string]*worldCup2026ScorerAccumulator{}
	goalTiming := map[string]int{}
	teamNames := map[string]struct{}{}
	for _, match := range matches {
		for _, teamName := range []string{match.HomeTeam, match.AwayTeam} {
			teamNames[teamName] = struct{}{}
			if teamMap[teamName] == nil {
				teamMap[teamName] = &worldCup2026TeamAccumulator{WorldCup2026TeamRow: WorldCup2026TeamRow{TeamName: teamName, GroupName: match.GroupName, Stage: "Group stage"}}
			}
			if match.GroupName != "" && teamMap[teamName].GroupName == "" {
				teamMap[teamName].GroupName = match.GroupName
			}
		}
		if venueMap[match.Venue] == nil {
			venueMap[match.Venue] = &WorldCup2026VenueRow{Venue: match.Venue}
		}
		venueMap[match.Venue].Matches++
		venueMap[match.Venue].Goals += match.HomeGoals + match.AwayGoals
		stageRank := worldCup2026StageRank(match.Round)
		for _, teamName := range []string{match.HomeTeam, match.AwayTeam} {
			team := teamMap[teamName]
			if stageRank > team.stageRank {
				team.stageRank = stageRank
				team.Stage = match.Round
			}
		}
		if match.GroupName == "" {
			continue
		}
		home, away := teamMap[match.HomeTeam], teamMap[match.AwayTeam]
		home.Played++
		away.Played++
		home.GoalsFor += match.HomeGoals
		home.GoalsAgainst += match.AwayGoals
		away.GoalsFor += match.AwayGoals
		away.GoalsAgainst += match.HomeGoals
		if match.HomeGoals > match.AwayGoals {
			home.Won++
			home.Points += 3
			away.Lost++
		} else if match.AwayGoals > match.HomeGoals {
			away.Won++
			away.Points += 3
			home.Lost++
		} else {
			home.Drawn++
			away.Drawn++
			home.Points++
			away.Points++
		}
	}

	for _, goal := range goals {
		if bucket := worldCup2026GoalBucket(goal.Minute); bucket != "" {
			goalTiming[bucket]++
		}
		if goal.IsOwnGoal {
			continue
		}
		key := goal.TeamName + "\x00" + goal.PlayerName
		if scorerMap[key] == nil {
			scorerMap[key] = &worldCup2026ScorerAccumulator{WorldCup2026Scorer: WorldCup2026Scorer{TeamName: goal.TeamName, PlayerName: goal.PlayerName}, matches: map[int]struct{}{}}
		}
		scorerMap[key].Goals++
		scorerMap[key].matches[goal.MatchNumber] = struct{}{}
		if goal.IsPenalty {
			scorerMap[key].PenaltyGoals++
		}
	}
	for _, scorer := range scorerMap {
		scorer.Matches = len(scorer.matches)
	}

	if final := worldCup2026MatchByRound(matches, "Final"); final != nil {
		if final.Winner != "" {
			loser := final.HomeTeam
			if loser == final.Winner {
				loser = final.AwayTeam
			}
			teamMap[final.Winner].Stage = "Champion"
			teamMap[loser].Stage = "Runner-up"
		}
	}
	if third := worldCup2026MatchByRound(matches, "Match for third place"); third != nil && third.Winner != "" {
		loser := third.HomeTeam
		if loser == third.Winner {
			loser = third.AwayTeam
		}
		teamMap[third.Winner].Stage = "Third place"
		teamMap[loser].Stage = "Fourth place"
	}

	groupTeams := map[string][]*worldCup2026TeamAccumulator{}
	for _, team := range teamMap {
		team.GoalDifference = team.GoalsFor - team.GoalsAgainst
		groupTeams[team.GroupName] = append(groupTeams[team.GroupName], team)
	}
	teamRows := make([]WorldCup2026TeamRow, 0, len(teamMap))
	for groupName, teams := range groupTeams {
		sort.Slice(teams, func(left, right int) bool {
			return teams[left].Points > teams[right].Points ||
				(teams[left].Points == teams[right].Points && teams[left].GoalDifference > teams[right].GoalDifference) ||
				(teams[left].Points == teams[right].Points && teams[left].GoalDifference == teams[right].GoalDifference && teams[left].GoalsFor > teams[right].GoalsFor) ||
				(teams[left].Points == teams[right].Points && teams[left].GoalDifference == teams[right].GoalDifference && teams[left].GoalsFor == teams[right].GoalsFor && teams[left].TeamName < teams[right].TeamName)
		})
		for rank, team := range teams {
			team.Rank = rank + 1
			team.GroupName = groupName
			teamRows = append(teamRows, team.WorldCup2026TeamRow)
		}
	}
	sort.Slice(teamRows, func(left, right int) bool {
		return teamRows[left].GroupName < teamRows[right].GroupName || (teamRows[left].GroupName == teamRows[right].GroupName && teamRows[left].Rank < teamRows[right].Rank)
	})

	scorers := make([]WorldCup2026Scorer, 0, len(scorerMap))
	for _, scorer := range scorerMap {
		scorers = append(scorers, scorer.WorldCup2026Scorer)
	}
	sort.Slice(scorers, func(left, right int) bool {
		return scorers[left].Goals > scorers[right].Goals || (scorers[left].Goals == scorers[right].Goals && scorers[left].PenaltyGoals < scorers[right].PenaltyGoals) || (scorers[left].Goals == scorers[right].Goals && scorers[left].PenaltyGoals == scorers[right].PenaltyGoals && scorers[left].PlayerName < scorers[right].PlayerName)
	})
	if len(scorers) > 12 {
		scorers = scorers[:12]
	}

	timingLabels := []string{"0-15", "16-30", "31-45+", "46-60", "61-75", "76-90+", "91-105", "106-120"}
	timing := make([]WorldCup2026TimingBucket, 0, len(timingLabels))
	for _, label := range timingLabels {
		timing = append(timing, WorldCup2026TimingBucket{Label: label, Goals: goalTiming[label]})
	}
	venues := make([]WorldCup2026VenueRow, 0, len(venueMap))
	for _, venue := range venueMap {
		venue.AverageGoals = float64(venue.Goals) / float64(venue.Matches)
		venues = append(venues, *venue)
	}
	sort.Slice(venues, func(left, right int) bool {
		return venues[left].Goals > venues[right].Goals || (venues[left].Goals == venues[right].Goals && venues[left].Venue < venues[right].Venue)
	})

	finalScore, champion, runnerUp := "", "", ""
	if final := worldCup2026MatchByRound(matches, "Final"); final != nil {
		finalScore = fmt.Sprintf("%d-%d", final.HomeGoals, final.AwayGoals)
		champion = final.Winner
		if champion == final.HomeTeam {
			runnerUp = final.AwayTeam
		} else {
			runnerUp = final.HomeTeam
		}
	}
	penaltyShootouts, extraTime := 0, 0
	for _, match := range matches {
		if match.PenaltyShootout {
			penaltyShootouts++
		}
		if match.HomeGoals != match.RegulationHomeGoals || match.AwayGoals != match.RegulationAwayGoals {
			extraTime++
		}
	}
	return WorldCup2026Overview{
		Summary: WorldCup2026Summary{Matches: len(matches), Teams: len(teamNames), Goals: len(goals), AverageGoals: float64(len(goals)) / float64(maxInt(len(matches), 1)), Venues: len(venueMap), PenaltyShootouts: penaltyShootouts, ExtraTimeMatches: extraTime, Champion: champion, RunnerUp: runnerUp, FinalScore: finalScore},
		Teams:   teamRows, TopScorers: scorers, GoalTiming: timing, Venues: venues, Matches: matches,
	}
}

func worldCup2026StageRank(round string) int {
	switch round {
	case "Round of 32":
		return 1
	case "Round of 16":
		return 2
	case "Quarter-final":
		return 3
	case "Semi-final":
		return 4
	case "Match for third place":
		return 5
	case "Final":
		return 6
	default:
		return 0
	}
}

func worldCup2026MatchByRound(matches []WorldCup2026MatchSummary, round string) *WorldCup2026MatchSummary {
	for index := range matches {
		if matches[index].Round == round {
			return &matches[index]
		}
	}
	return nil
}

func worldCup2026GoalBucket(minute string) string {
	base := minute
	if separator := strings.IndexByte(minute, '+'); separator >= 0 {
		base = minute[:separator]
	}
	value, err := strconv.Atoi(base)
	if err != nil {
		return ""
	}
	switch {
	case value <= 15:
		return "0-15"
	case value <= 30:
		return "16-30"
	case value <= 45:
		return "31-45+"
	case value <= 60:
		return "46-60"
	case value <= 75:
		return "61-75"
	case value <= 90:
		return "76-90+"
	case value <= 105:
		return "91-105"
	default:
		return "106-120"
	}
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func worldCup2026MatchesIcebergSchema() *iceberg.Schema {
	return icebergSchema([]iceberg.NestedField{
		{ID: 1, Name: "source", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 2, Name: "source_version", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 3, Name: "tournament", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 4, Name: "source_match_id", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 5, Name: "match_number", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 6, Name: "round", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 7, Name: "group_name", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 8, Name: "match_date", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 9, Name: "kickoff_time", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 10, Name: "venue", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 11, Name: "home_team", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 12, Name: "away_team", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 13, Name: "regulation_home_goals", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 14, Name: "regulation_away_goals", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 15, Name: "home_goals", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 16, Name: "away_goals", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 17, Name: "shootout_home_goals", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 18, Name: "shootout_away_goals", Type: iceberg.PrimitiveTypes.Int64, Required: true},
	})
}

func worldCup2026GoalsIcebergSchema() *iceberg.Schema {
	return icebergSchema([]iceberg.NestedField{
		{ID: 1, Name: "source", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 2, Name: "source_version", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 3, Name: "tournament", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 4, Name: "match_number", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 5, Name: "team_name", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 6, Name: "player_name", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 7, Name: "minute", Type: iceberg.PrimitiveTypes.String, Required: true},
		{ID: 8, Name: "minute_value", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 9, Name: "is_penalty", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		{ID: 10, Name: "is_own_goal", Type: iceberg.PrimitiveTypes.Int64, Required: true},
	})
}

func worldCup2026MatchesArrowTable(payload WorldCup2026Payload) arrow.Table {
	schema := arrowSchema([]arrow.Field{
		stringField("source"), stringField("source_version"), stringField("tournament"), stringField("source_match_id"), intField("match_number"),
		stringField("round"), stringField("group_name"), stringField("match_date"), stringField("kickoff_time"), stringField("venue"),
		stringField("home_team"), stringField("away_team"), intField("regulation_home_goals"), intField("regulation_away_goals"),
		intField("home_goals"), intField("away_goals"), intField("shootout_home_goals"), intField("shootout_away_goals"),
	})
	builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	for _, match := range payload.Matches {
		builder.Field(0).(*array.StringBuilder).Append(payload.Source)
		builder.Field(1).(*array.StringBuilder).Append(payload.SourceVersion)
		builder.Field(2).(*array.StringBuilder).Append(payload.Tournament)
		builder.Field(3).(*array.StringBuilder).Append(match.SourceMatchID)
		builder.Field(4).(*array.Int64Builder).Append(int64(match.MatchNumber))
		builder.Field(5).(*array.StringBuilder).Append(match.Round)
		builder.Field(6).(*array.StringBuilder).Append(match.GroupName)
		builder.Field(7).(*array.StringBuilder).Append(match.MatchDate)
		builder.Field(8).(*array.StringBuilder).Append(match.KickoffTime)
		builder.Field(9).(*array.StringBuilder).Append(match.Venue)
		builder.Field(10).(*array.StringBuilder).Append(match.HomeTeam)
		builder.Field(11).(*array.StringBuilder).Append(match.AwayTeam)
		builder.Field(12).(*array.Int64Builder).Append(int64(match.RegulationHomeGoals))
		builder.Field(13).(*array.Int64Builder).Append(int64(match.RegulationAwayGoals))
		builder.Field(14).(*array.Int64Builder).Append(int64(match.HomeGoals))
		builder.Field(15).(*array.Int64Builder).Append(int64(match.AwayGoals))
		builder.Field(16).(*array.Int64Builder).Append(int64(match.ShootoutHomeGoals))
		builder.Field(17).(*array.Int64Builder).Append(int64(match.ShootoutAwayGoals))
	}
	return finishArrowTable(schema, builder)
}

func worldCup2026GoalsArrowTable(payload WorldCup2026Payload) arrow.Table {
	schema := arrowSchema([]arrow.Field{
		stringField("source"), stringField("source_version"), stringField("tournament"), intField("match_number"),
		stringField("team_name"), stringField("player_name"), stringField("minute"), intField("minute_value"), intField("is_penalty"), intField("is_own_goal"),
	})
	builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	for _, match := range payload.Matches {
		for _, goal := range match.Goals {
			builder.Field(0).(*array.StringBuilder).Append(payload.Source)
			builder.Field(1).(*array.StringBuilder).Append(payload.SourceVersion)
			builder.Field(2).(*array.StringBuilder).Append(payload.Tournament)
			builder.Field(3).(*array.Int64Builder).Append(int64(match.MatchNumber))
			builder.Field(4).(*array.StringBuilder).Append(goal.TeamName)
			builder.Field(5).(*array.StringBuilder).Append(goal.PlayerName)
			builder.Field(6).(*array.StringBuilder).Append(goal.Minute)
			builder.Field(7).(*array.Int64Builder).Append(int64(goal.MinuteValue))
			builder.Field(8).(*array.Int64Builder).Append(int64(boolToUInt8(goal.IsPenalty)))
			builder.Field(9).(*array.Int64Builder).Append(int64(boolToUInt8(goal.IsOwnGoal)))
		}
	}
	return finishArrowTable(schema, builder)
}

func (w *icebergWriter) appendWorldCup2026(ctx context.Context, payload WorldCup2026Payload) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.ensureCatalog(ctx); err != nil {
		return err
	}
	if err := w.catalog.CreateNamespace(ctx, catalog.ToIdentifier("touchline"), nil); err != nil && !errors.Is(err, catalog.ErrNamespaceAlreadyExists) {
		return err
	}
	items := []struct {
		name   string
		schema *iceberg.Schema
		table  arrow.Table
	}{
		{name: "worldcup_2026_matches", schema: worldCup2026MatchesIcebergSchema(), table: worldCup2026MatchesArrowTable(payload)},
		{name: "worldcup_2026_goals", schema: worldCup2026GoalsIcebergSchema(), table: worldCup2026GoalsArrowTable(payload)},
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
				iceberg.EqualTo(iceberg.Reference("tournament"), payload.Tournament),
			)),
		)
		item.table.Release()
		if err != nil {
			return err
		}
	}
	return nil
}
