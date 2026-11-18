package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// kindsTimelineJSON is the subset of agent-deck.recall.rows/v2 these tests
// read: the visible kinds plus the two cursors.
type kindsTimelineJSON struct {
	Turns []struct {
		ID   string `json:"id"`
		Kind string `json:"kind"`
	} `json:"turns"`
	ThroughCursor string `json:"through_cursor"`
	NextCursor    string `json:"next_cursor"`
}

func decodeKindsTimeline(t *testing.T, stdout string) kindsTimelineJSON {
	t.Helper()
	var tl kindsTimelineJSON
	if err := json.Unmarshal([]byte(stdout), &tl); err != nil {
		t.Fatalf("JSON: %v %s", err, stdout)
	}
	return tl
}

// TestRecallTimelineKindsFilter: --kinds keeps only the named kinds on the
// owner host, and the answer carries the pre-filter boundary in next_cursor so
// a client resuming from it cannot step backwards over a dropped row.
func TestRecallTimelineKindsFilter(t *testing.T) {
	home, id, _ := rowsTestSession(t)

	stdout, stderr, code := runAgentDeck(t, home, "recall", "timeline", id, "--json")
	if code != 0 {
		t.Fatalf("unfiltered: %d %s %s", code, stdout, stderr)
	}
	full := decodeKindsTimeline(t, stdout)
	if len(full.Turns) == 0 {
		t.Fatal("fixture produced no rows")
	}
	if full.NextCursor != "" {
		t.Fatalf("unfiltered answer must stay byte-identical: next_cursor = %q", full.NextCursor)
	}
	lastUnfiltered := full.Turns[len(full.Turns)-1].ID

	stdout, stderr, code = runAgentDeck(t, home, "recall", "timeline", id, "--json", "--kinds", "user,assistant")
	if code != 0 {
		t.Fatalf("--kinds: %d %s %s", code, stdout, stderr)
	}
	filtered := decodeKindsTimeline(t, stdout)
	if len(filtered.Turns) == 0 || len(filtered.Turns) >= len(full.Turns) {
		t.Fatalf("filter kept %d of %d rows", len(filtered.Turns), len(full.Turns))
	}
	for _, row := range filtered.Turns {
		if row.Kind != "user" && row.Kind != "assistant" {
			t.Fatalf("filtered answer carries kind %q", row.Kind)
		}
	}
	if filtered.NextCursor != lastUnfiltered {
		t.Fatalf("next_cursor = %q, want the pre-filter last row %q", filtered.NextCursor, lastUnfiltered)
	}
	if filtered.ThroughCursor != full.ThroughCursor {
		t.Fatalf("through_cursor moved with the filter: %q vs %q", filtered.ThroughCursor, full.ThroughCursor)
	}

	// The resume token is through_cursor, which filtering leaves alone, so a
	// client that pages with it never replays the filtered window.
	stdout, stderr, code = runAgentDeck(t, home, "recall", "timeline", id, "--json", "--since", filtered.ThroughCursor, "--kinds", "user,assistant")
	if code != 0 {
		t.Fatalf("--since through_cursor: %d %s %s", code, stdout, stderr)
	}
	if resumed := decodeKindsTimeline(t, stdout); len(resumed.Turns) != 0 {
		t.Fatalf("resume from through_cursor replayed %d rows: %s", len(resumed.Turns), stdout)
	}
}

// TestRecallTimelineKindsValidation: a typo is refused before any read, and
// --skip-empty-body drops rows with nothing to show.
func TestRecallTimelineKindsValidation(t *testing.T) {
	home, id, _ := rowsTestSession(t)
	if _, _, code := runAgentDeck(t, home, "recall", "timeline", id, "--json", "--kinds", "user,assistent"); code != 2 {
		t.Fatalf("unknown kind accepted: exit %d", code)
	}
	if _, _, code := runAgentDeck(t, home, "recall", "timeline", id, "--json", "--kinds", " , "); code != 2 {
		t.Fatalf("empty --kinds accepted: exit %d", code)
	}

	stdout, stderr, code := runAgentDeck(t, home, "recall", "timeline", id, "--json", "--skip-empty-body")
	if code != 0 {
		t.Fatalf("--skip-empty-body: %d %s %s", code, stdout, stderr)
	}
	skipped := decodeKindsTimeline(t, stdout)
	if skipped.NextCursor == "" {
		t.Fatal("a filtered answer must carry next_cursor")
	}
	stdout, _, code = runAgentDeck(t, home, "recall", "timeline", id, "--json")
	if code != 0 {
		t.Fatalf("unfiltered: %d %s", code, stdout)
	}
	if full := decodeKindsTimeline(t, stdout); len(skipped.Turns) > len(full.Turns) {
		t.Fatalf("--skip-empty-body kept more rows (%d) than the unfiltered read (%d)", len(skipped.Turns), len(full.Turns))
	}

	// --v1 answers turns, not rows: a row filter is refused rather than
	// silently ignored.
	if _, _, code := runAgentDeck(t, home, "recall", "timeline", id, "--json", "--v1", "--kinds", "user"); code != 2 {
		t.Fatalf("--kinds with --v1 accepted: exit %d", code)
	}

	// The batch path takes the same filter.
	stdout, stderr, code = runAgentDeck(t, home, "recall", "timeline", "--json", "--sessions", id, "--kinds", "user")
	if code != 0 || !strings.Contains(stdout, `"next_cursor"`) {
		t.Fatalf("batch --kinds: %d %s %s", code, stdout, stderr)
	}
}
