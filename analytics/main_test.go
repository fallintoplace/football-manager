package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidatePayloadAcceptsLocalMatchday(t *testing.T) {
	payload := MatchdayPayload{
		RunID:  "career:season:1",
		Season: 1,
		Round:  0,
		Results: []MatchResult{{
			FixtureID: "match-1",
			HomeID:    "arsenal",
			AwayID:    "villa",
		}},
	}

	if err := validatePayload(payload); err != nil {
		t.Fatalf("expected payload to be valid: %v", err)
	}
}

func TestValidatePayloadRejectsOversizedReplay(t *testing.T) {
	trace := make([]MatchFrame, 601)
	payload := MatchdayPayload{
		RunID:  "career:season:1",
		Season: 1,
		Round:  0,
		Results: []MatchResult{{
			FixtureID: "match-1",
			HomeID:    "arsenal",
			AwayID:    "villa",
			Trace:     trace,
		}},
	}

	if err := validatePayload(payload); err == nil {
		t.Fatal("expected oversized replay to be rejected")
	}
}

func TestValidatePayloadRequiresRunID(t *testing.T) {
	payload := MatchdayPayload{
		Season: 1,
		Round:  0,
		Results: []MatchResult{{
			FixtureID: "match-1",
			HomeID:    "arsenal",
			AwayID:    "villa",
		}},
	}

	if err := validatePayload(payload); err == nil {
		t.Fatal("expected missing run ID to be rejected")
	}
}

func TestCorsAllowsConfiguredLoopbackOrigins(t *testing.T) {
	handler := cors(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), "http://localhost:5173, http://127.0.0.1:5173")

	for _, origin := range []string{"http://localhost:5173", "http://127.0.0.1:5173"} {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("Origin", origin)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		if got := response.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Fatalf("expected CORS origin %q, got %q", origin, got)
		}
	}
}

func TestBuildMatchClubFactsPreservesTacticalLedger(t *testing.T) {
	result := MatchResult{
		HomeID:    "arsenal",
		AwayID:    "villa",
		HomeGoals: 2,
		AwayGoals: 1,
		Metrics: MatchMetrics{
			HomeXG: 1.8, AwayXG: 0.7, HomePossession: 61, HomePressure: 74, AwayPressure: 52, HomeTerritory: 58,
			HomePossessions: 64, AwayPossessions: 48, HomeBoxEntries: 12, AwayBoxEntries: 5,
			HomePressWins: 9, AwayPressWins: 3, HomeCounters: 4, AwayCounters: 1,
		},
		Tactics: MatchTactics{
			Home: TacticSnapshot{Formation: "4-3-3", Mentality: "Relentless", Pressing: 82, Tempo: 70, DefensiveLine: 74},
			Away: TacticSnapshot{Formation: "4-4-2", Mentality: "Measured", Pressing: 48, Tempo: 44, DefensiveLine: 51},
		},
	}

	facts := buildMatchClubFacts(result)
	if len(facts) != 2 {
		t.Fatalf("expected two club facts, got %d", len(facts))
	}
	if facts[0].ClubID != "arsenal" || !facts[0].IsHome || facts[0].Pressing != 82 || facts[0].BoxEntries != 12 {
		t.Fatalf("unexpected home fact: %+v", facts[0])
	}
	if facts[1].ClubID != "villa" || facts[1].IsHome || facts[1].Possession != 39 || facts[1].Formation != "4-4-2" {
		t.Fatalf("unexpected away fact: %+v", facts[1])
	}
}

func TestBuildPlayerMatchFactsAggregatesEvents(t *testing.T) {
	chanceXG := 0.22
	players := []Player{
		{ID: "arsenal-fwd-1", ClubID: "arsenal", Name: "Forward One", Position: "FWD"},
		{ID: "villa-gk-1", ClubID: "villa", Name: "Keeper One", Position: "GK"},
	}
	result := MatchResult{
		FixtureID: "match-1",
		HomeID:    "arsenal",
		AwayID:    "villa",
		Events: []MatchEvent{
			{Type: "goal", TeamID: "arsenal", PlayerName: "Forward One", XG: &chanceXG},
			{Type: "save", TeamID: "villa", PlayerName: "Keeper One"},
		},
		PlayerRatings: map[string]float64{"arsenal-fwd-1": 8.4, "villa-gk-1": 7.1},
	}

	facts := buildPlayerMatchFacts(players, result)
	if len(facts) != 2 {
		t.Fatalf("expected two player facts, got %d", len(facts))
	}
	if facts[0].PlayerID != "arsenal-fwd-1" || facts[0].Goals != 1 || facts[0].Shots != 1 || facts[0].XG != chanceXG {
		t.Fatalf("unexpected forward fact: %+v", facts[0])
	}
	if facts[1].PlayerID != "villa-gk-1" || facts[1].Shots != 1 || facts[1].OpponentID != "arsenal" {
		t.Fatalf("unexpected keeper fact: %+v", facts[1])
	}
}

func TestCareerIDFromRunID(t *testing.T) {
	if got := careerIDFromRunID("career-123:season:2"); got != "career-123" {
		t.Fatalf("expected career ID to be extracted, got %q", got)
	}
	if got := careerIDFromRunID("legacy-run"); got != "legacy-run" {
		t.Fatalf("expected legacy run ID to be preserved, got %q", got)
	}
}

func TestActionAnalystNoteFlagsBuildUpRisk(t *testing.T) {
	note := actionAnalystNote(ActionInsights{Actions: 20, Passes: 10, CompletedPasses: 6, PassCompletion: 60})

	if note != "Build-up is breaking down frequently. Consider a safer passing option or a more measured tempo." {
		t.Fatalf("unexpected build-up note: %q", note)
	}
}

func TestActionAnalystNoteHandlesEmptyFeed(t *testing.T) {
	note := actionAnalystNote(ActionInsights{})

	if note != "No structured actions have landed for this run yet. Play a matchday to start the action feed." {
		t.Fatalf("unexpected empty-feed note: %q", note)
	}
}

func TestPrimaryActionRoleClassifiesProgressiveCreator(t *testing.T) {
	role := primaryActionRole(PlayerActionProfile{Passes: 8, ProgressiveActions: 4, Actions: 10})

	if role != "Progressive creator" {
		t.Fatalf("expected progressive creator, got %q", role)
	}
}

func TestPrimaryActionRoleClassifiesBoxThreat(t *testing.T) {
	role := primaryActionRole(PlayerActionProfile{Shots: 3, XG: 0.21, BoxActions: 2, Actions: 7})

	if role != "Box threat" {
		t.Fatalf("expected box threat, got %q", role)
	}
}

func TestValidateRealDataPayloadAcceptsStatsBombSample(t *testing.T) {
	payload := RealDataPayload{
		Source:        "statsbomb",
		SourceVersion: "hudl/open-data@master",
		Competition:   "Premier League",
		Season:        2015,
		SeasonLabel:   "2015/2016",
		Matches: []RealMatch{{
			SourceMatchID: "3754217",
			MatchDate:     "2015-09-19",
			HomeTeamID:    "statsbomb:team:33",
			AwayTeamID:    "statsbomb:team:1",
			Actions: []RealDataAction{{
				SourceActionID: "3754217:event-1",
				TeamID:         "statsbomb:team:1",
				PlayerID:       "statsbomb:player:3668",
				ActionType:     "pass",
				StartX:         50,
				StartY:         40,
			}},
		}},
	}

	if err := validateRealDataPayload(payload); err != nil {
		t.Fatalf("expected real-data payload to be valid: %v", err)
	}
}

func TestValidateRealDataPayloadRejectsOutOfBoundsCoordinates(t *testing.T) {
	payload := RealDataPayload{
		Source:        "statsbomb",
		SourceVersion: "hudl/open-data@master",
		Competition:   "Premier League",
		Season:        2015,
		SeasonLabel:   "2015/2016",
		Matches: []RealMatch{{
			SourceMatchID: "3754217",
			MatchDate:     "2015-09-19",
			HomeTeamID:    "statsbomb:team:33",
			AwayTeamID:    "statsbomb:team:1",
			Actions: []RealDataAction{{
				SourceActionID: "3754217:event-1",
				TeamID:         "statsbomb:team:1",
				PlayerID:       "statsbomb:player:3668",
				ActionType:     "pass",
				StartX:         101,
				StartY:         40,
			}},
		}},
	}

	if err := validateRealDataPayload(payload); err == nil {
		t.Fatal("expected out-of-bounds coordinates to be rejected")
	}
}

func TestValidateWorldCup2026PayloadRejectsDuplicateMatchNumbers(t *testing.T) {
	match := WorldCup2026MatchPayload{
		SourceMatchID: "match-1",
		MatchNumber:   1,
		MatchDate:     "2026-06-11",
		HomeTeam:      "Alpha",
		AwayTeam:      "Beta",
	}
	payload := WorldCup2026Payload{
		Source:        "openfootball",
		SourceVersion: "test",
		Tournament:    "world-cup-2026",
		Matches:       []WorldCup2026MatchPayload{match, match},
	}

	if err := validateWorldCup2026Payload(payload); err == nil {
		t.Fatal("expected duplicate match numbers to be rejected")
	}
}

func TestWorldCup2026GoalBucketHandlesStoppageAndExtraTime(t *testing.T) {
	tests := map[string]string{
		"15":    "0-15",
		"45+3":  "31-45+",
		"90+2":  "76-90+",
		"105":   "91-105",
		"120+1": "106-120",
		"n/a":   "",
	}

	for minute, want := range tests {
		if got := worldCup2026GoalBucket(minute); got != want {
			t.Errorf("worldCup2026GoalBucket(%q) = %q, want %q", minute, got, want)
		}
	}
}

func TestBuildWorldCup2026OverviewCalculatesTournamentStory(t *testing.T) {
	matches := []WorldCup2026MatchSummary{
		{
			MatchNumber: 1, GroupName: "Group A", Round: "Group stage", Venue: "North Stadium",
			HomeTeam: "Alpha", AwayTeam: "Beta", HomeGoals: 2, AwayGoals: 0,
			Goals: []WorldCup2026Goal{{MatchNumber: 1, TeamName: "Alpha", PlayerName: "A One", Minute: "12", MinuteValue: 12}},
		},
		{
			MatchNumber: 2, GroupName: "Group A", Round: "Group stage", Venue: "South Stadium",
			HomeTeam: "Beta", AwayTeam: "Gamma", HomeGoals: 1, AwayGoals: 1,
		},
		{
			MatchNumber: 3, Round: "Final", Venue: "North Stadium", RegulationHomeGoals: 0, RegulationAwayGoals: 0,
			HomeTeam: "Alpha", AwayTeam: "Gamma", HomeGoals: 1, AwayGoals: 0,
			Goals: []WorldCup2026Goal{{MatchNumber: 3, TeamName: "Alpha", PlayerName: "A One", Minute: "105", MinuteValue: 105}},
		},
	}
	for index := range matches {
		matches[index].Winner = worldCup2026Winner(matches[index])
	}

	overview := buildWorldCup2026Overview(matches, []WorldCup2026Goal{
		matches[0].Goals[0],
		matches[2].Goals[0],
	})

	if overview.Summary.Teams != 3 || overview.Summary.Goals != 2 || overview.Summary.Champion != "Alpha" || overview.Summary.RunnerUp != "Gamma" || overview.Summary.FinalScore != "1-0" {
		t.Fatalf("unexpected World Cup summary: %+v", overview.Summary)
	}
	if len(overview.TopScorers) != 1 || overview.TopScorers[0].PlayerName != "A One" || overview.TopScorers[0].Goals != 2 || overview.TopScorers[0].Matches != 2 {
		t.Fatalf("unexpected top scorers: %+v", overview.TopScorers)
	}
	if overview.Teams[0].TeamName != "Alpha" || overview.Teams[0].Points != 3 || overview.Teams[0].Stage != "Champion" {
		t.Fatalf("unexpected team table: %+v", overview.Teams)
	}
}
