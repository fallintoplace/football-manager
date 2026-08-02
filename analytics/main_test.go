package main

import "testing"

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
