package query

import (
	"fmt"
	"strings"
)

// RowKinds is the row-kind vocabulary `recall timeline --kinds` accepts: the
// same values the parser assigns to Row.Kind, so a filter names rows the way
// the output already does.
var RowKinds = []string{
	"user", "assistant", "thinking", "tool", "bash", "edit", "read",
	"subagent", "todo", "question", "skill", "command", "system",
	"compaction", "turn_end", "other",
}

// rowKindSet is RowKinds as a membership test.
var rowKindSet = func() map[string]bool {
	set := make(map[string]bool, len(RowKinds))
	for _, k := range RowKinds {
		set[k] = true
	}
	return set
}()

// ParseRowKinds turns a comma-separated allowlist into the validated set a
// RowsOptions carries. An unknown name is an error naming the whole
// vocabulary, so a typo cannot silently return an empty timeline.
func ParseRowKinds(list string) ([]string, error) {
	var out []string
	seen := make(map[string]bool)
	for _, part := range strings.Split(list, ",") {
		kind := strings.ToLower(strings.TrimSpace(part))
		if kind == "" {
			continue
		}
		if !rowKindSet[kind] {
			return nil, fmt.Errorf("recall timeline: unknown row kind %q (kinds: %s)", kind, strings.Join(RowKinds, ", "))
		}
		if seen[kind] {
			continue
		}
		seen[kind] = true
		out = append(out, kind)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("recall timeline: --kinds needs at least one kind (kinds: %s)", strings.Join(RowKinds, ", "))
	}
	return out, nil
}

// RowsFiltered reports whether an option set drops any row.
func RowsFiltered(opts RowsOptions) bool {
	return len(opts.Kinds) > 0 || opts.SkipEmptyBody
}

// FilterRows keeps the rows a filter allows and reports how many it dropped.
// A kind allowlist matches Row.Kind exactly; SkipEmptyBody drops a row with no
// visible text, meaning Body, Summary and Detail are all blank and it has no
// children to show. Children of a kept row are left alone: they belong to it.
func FilterRows(rows []Row, opts RowsOptions) (kept []Row, dropped int) {
	if !RowsFiltered(opts) {
		return rows, 0
	}
	kindOK := make(map[string]bool, len(opts.Kinds))
	for _, k := range opts.Kinds {
		kindOK[k] = true
	}
	kept = make([]Row, 0, len(rows))
	for _, row := range rows {
		if len(kindOK) > 0 && !kindOK[row.Kind] {
			dropped++
			continue
		}
		if opts.SkipEmptyBody && !rowHasVisibleText(row) {
			dropped++
			continue
		}
		kept = append(kept, row)
	}
	return kept, dropped
}

// rowHasVisibleText reports whether a row renders something: any of the three
// text fields carries non-whitespace, or it has children to show.
func rowHasVisibleText(row Row) bool {
	if len(row.Children) > 0 {
		return true
	}
	for _, field := range []string{row.Body, row.Summary, row.Detail} {
		if strings.TrimSpace(field) != "" {
			return true
		}
	}
	return false
}
