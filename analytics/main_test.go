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
