package differ

import (
	"testing"

	"github.com/naman47vyas/log-diff/internal/drain"
)

func cluster(id int, template string, count int) *drain.LogCluster {
	return &drain.LogCluster{
		ID:      id,
		Tokens:  tokenize(template),
		Count:   count,
		Samples: []string{"sample line"},
	}
}

// simple tokenizer for test helpers
func tokenize(s string) []string {
	out := []string{}
	current := ""
	for _, c := range s {
		if c == ' ' {
			if current != "" {
				out = append(out, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		out = append(out, current)
	}
	return out
}

func TestDiffNewTemplates(t *testing.T) {
	pre := []*drain.LogCluster{
		cluster(1, "connected to <*>", 10),
	}
	post := []*drain.LogCluster{
		cluster(1, "connected to <*>", 10),
		cluster(2, "panic in module <*>", 5),
	}

	r := Diff(pre, post, DefaultConfig())

	if len(r.New) != 1 {
		t.Fatalf("expected 1 new, got %d", len(r.New))
	}
	if r.New[0].Template != "panic in module <*>" {
		t.Errorf("new template = %q", r.New[0].Template)
	}
}

func TestDiffGoneTemplates(t *testing.T) {
	pre := []*drain.LogCluster{
		cluster(1, "connected to <*>", 10),
		cluster(2, "legacy cleanup ran", 5),
	}
	post := []*drain.LogCluster{
		cluster(1, "connected to <*>", 10),
	}

	r := Diff(pre, post, DefaultConfig())

	if len(r.Gone) != 1 {
		t.Fatalf("expected 1 gone, got %d", len(r.Gone))
	}
	if r.Gone[0].Template != "legacy cleanup ran" {
		t.Errorf("gone template = %q", r.Gone[0].Template)
	}
}

func TestDiffChangedFrequency(t *testing.T) {
	// same total lines, but template frequency shifts dramatically
	pre := []*drain.LogCluster{
		cluster(1, "health check ok", 90),
		cluster(2, "error <*>", 10),
	}
	post := []*drain.LogCluster{
		cluster(1, "health check ok", 50),
		cluster(2, "error <*>", 50),
	}

	r := Diff(pre, post, DefaultConfig())

	if len(r.Changed) != 1 {
		t.Fatalf("expected 1 changed, got %d", len(r.Changed))
	}
	if r.Changed[0].Template != "error <*>" {
		t.Errorf("changed template = %q", r.Changed[0].Template)
	}
}

func TestDiffNoChanges(t *testing.T) {
	pre := []*drain.LogCluster{
		cluster(1, "connected to <*>", 50),
		cluster(2, "request handled", 50),
	}
	post := []*drain.LogCluster{
		cluster(1, "connected to <*>", 50),
		cluster(2, "request handled", 50),
	}

	r := Diff(pre, post, DefaultConfig())

	if len(r.New) != 0 || len(r.Gone) != 0 || len(r.Changed) != 0 {
		t.Errorf("expected no diffs, got new=%d gone=%d changed=%d",
			len(r.New), len(r.Gone), len(r.Changed))
	}
}

func TestDiffTotals(t *testing.T) {
	pre := []*drain.LogCluster{
		cluster(1, "a", 30),
		cluster(2, "b", 70),
	}
	post := []*drain.LogCluster{
		cluster(1, "a", 40),
		cluster(2, "b", 60),
	}

	r := Diff(pre, post, DefaultConfig())

	if r.PreTotal != 100 {
		t.Errorf("PreTotal = %d, want 100", r.PreTotal)
	}
	if r.PostTotal != 100 {
		t.Errorf("PostTotal = %d, want 100", r.PostTotal)
	}
}

func TestDiffCustomThreshold(t *testing.T) {
	pre := []*drain.LogCluster{
		cluster(1, "request <*>", 50),
	}
	post := []*drain.LogCluster{
		cluster(1, "request <*>", 75),
	}

	// strict threshold — 1.5x change should trigger
	cfg := Config{FreqRatioThreshold: 1.5}
	r := Diff(pre, post, cfg)

	if len(r.Changed) != 0 {
		t.Errorf("expected 0 changed with equal totals and same ratio, got %d", len(r.Changed))
	}

	// make the shift more dramatic
	post2 := []*drain.LogCluster{
		cluster(1, "request <*>", 40),
		cluster(2, "other stuff", 60),
	}
	pre2 := []*drain.LogCluster{
		cluster(1, "request <*>", 90),
		cluster(2, "other stuff", 10),
	}

	r2 := Diff(pre2, post2, cfg)
	if len(r2.Changed) != 2 {
		t.Errorf("expected 2 changed, got %d", len(r2.Changed))
	}
}
