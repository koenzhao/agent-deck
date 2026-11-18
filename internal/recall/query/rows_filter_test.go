package query

import (
	"strings"
	"testing"
)

func TestParseRowKindsAcceptsVocabulary(t *testing.T) {
	got, err := ParseRowKinds(" user, Assistant ,user,,thinking ")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []string{"user", "assistant", "thinking"}
	if len(got) != len(want) {
		t.Fatalf("kinds = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("kinds = %v, want %v (order and dedupe)", got, want)
		}
	}
}

func TestParseRowKindsRejectsUnknownAndEmpty(t *testing.T) {
	if _, err := ParseRowKinds("user,assistent"); err == nil {
		t.Fatal("a typo must not silently return an empty timeline")
	} else if !strings.Contains(err.Error(), "assistent") || !strings.Contains(err.Error(), "user") {
		t.Fatalf("error must name the bad kind and the vocabulary: %v", err)
	}
	for _, list := range []string{"", " , ", ","} {
		if _, err := ParseRowKinds(list); err == nil {
			t.Fatalf("--kinds %q must be refused", list)
		}
	}
}

func TestFilterRowsByKind(t *testing.T) {
	rows := []Row{
		{ID: "a", Kind: "user", Body: "hi"},
		{ID: "b", Kind: "system", Body: "hook"},
		{ID: "c", Kind: "assistant", Body: "hello"},
		{ID: "d", Kind: "bash", Body: "ls"},
	}
	kept, dropped := FilterRows(rows, RowsOptions{Kinds: []string{"user", "assistant"}})
	if dropped != 2 || len(kept) != 2 || kept[0].ID != "a" || kept[1].ID != "c" {
		t.Fatalf("kept=%v dropped=%d", kept, dropped)
	}
}

func TestFilterRowsSkipEmptyBody(t *testing.T) {
	rows := []Row{
		{ID: "a", Kind: "user", Body: "hi"},
		{ID: "b", Kind: "system", Title: "Model changed"},
		{ID: "c", Kind: "assistant", Body: "   \n\t "},
		{ID: "d", Kind: "subagent", Children: []Row{{ID: "d1", Body: "child"}}},
		{ID: "e", Kind: "tool", Summary: "ran"},
	}
	kept, dropped := FilterRows(rows, RowsOptions{SkipEmptyBody: true})
	if dropped != 2 {
		t.Fatalf("dropped = %d, want 2 (blank body and whitespace-only body)", dropped)
	}
	var ids []string
	for _, r := range kept {
		ids = append(ids, r.ID)
	}
	if strings.Join(ids, ",") != "a,d,e" {
		t.Fatalf("kept = %v, want a,d,e (children and summary count as visible)", ids)
	}
}

func TestFilterRowsCombinesAndPassesThrough(t *testing.T) {
	rows := []Row{{ID: "a", Kind: "user", Body: "hi"}, {ID: "b", Kind: "system", Body: ""}}
	kept, dropped := FilterRows(rows, RowsOptions{Kinds: []string{"user"}, SkipEmptyBody: true})
	if dropped != 1 || len(kept) != 1 || kept[0].ID != "a" {
		t.Fatalf("combined filter: kept=%v dropped=%d", kept, dropped)
	}
	// No filter: the same slice comes back untouched, so the payload stays
	// byte-identical for existing clients.
	pass, n := FilterRows(rows, RowsOptions{})
	if n != 0 || len(pass) != len(rows) || &pass[0] != &rows[0] {
		t.Fatal("an empty filter must not copy or drop rows")
	}
	if RowsFiltered(RowsOptions{}) {
		t.Fatal("RowsFiltered must be false without a filter")
	}
}
