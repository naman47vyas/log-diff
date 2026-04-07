package main

import (
	"os"
	"testing"

	"github.com/naman47vyas/log-diff/internal/differ"
	"github.com/naman47vyas/log-diff/internal/drain"
	"github.com/naman47vyas/log-diff/internal/normalizer"
	"github.com/naman47vyas/log-diff/internal/parser"
)

func runDiff(t *testing.T) *differ.Report {
	t.Helper()

	p := parser.NewBracketParser()
	norm := normalizer.NewFast()
	drainCfg := drain.DefaultConfig()
	drainCfg.SimThreshold = 0.4

	preFile, err := os.Open("testdata/pre_release.txt")
	if err != nil {
		t.Fatalf("open pre: %v", err)
	}
	defer preFile.Close()

	postFile, err := os.Open("testdata/post_release.txt")
	if err != nil {
		t.Fatalf("open post: %v", err)
	}
	defer postFile.Close()

	preDrain, preErrs := processReader(preFile, p, norm, drainCfg)
	postDrain, postErrs := processReader(postFile, p, norm, drainCfg)

	if preErrs > 0 {
		t.Errorf("pre-release had %d parse errors", preErrs)
	}
	if postErrs > 0 {
		t.Errorf("post-release had %d parse errors", postErrs)
	}

	diffCfg := differ.DefaultConfig()
	return differ.Diff(preDrain.Clusters(), postDrain.Clusters(), diffCfg)
}

func TestDiffTotals(t *testing.T) {
	r := runDiff(t)

	if r.PreTotal != 100 {
		t.Errorf("PreTotal = %d, want 100", r.PreTotal)
	}
	if r.PostTotal != 100 {
		t.Errorf("PostTotal = %d, want 100", r.PostTotal)
	}
}

func TestNewTemplatesAreGenuinelyNew(t *testing.T) {
	r := runDiff(t)
	newTemplates := templates(r.New)

	// These patterns only exist in post-release
	expectNew := []string{
		"circuit breaker",
		"OOM kill",
		"gRPC stream",
		"feature flag",
	}
	for _, kw := range expectNew {
		if !anyContains(newTemplates, kw) {
			t.Errorf("expected a NEW template containing %q", kw)
		}
	}
}

func TestGoneTemplatesAreGenuinelyGone(t *testing.T) {
	r := runDiff(t)
	goneTemplates := templates(r.Gone)

	// These patterns only exist in pre-release
	expectGone := []string{
		"legacy sync",
		"deprecated endpoint",
		"xml config fallback",
	}
	for _, kw := range expectGone {
		if !anyContains(goneTemplates, kw) {
			t.Errorf("expected a GONE template containing %q", kw)
		}
	}
}

func TestSharedPatternsNotReportedAsNewOrGone(t *testing.T) {
	r := runDiff(t)
	newTemplates := templates(r.New)
	goneTemplates := templates(r.Gone)

	// These patterns exist in BOTH files and have stable structure —
	// they must never appear as new or gone.
	sharedPatterns := []string{
		"health check passed",
		"flushed",
		"connected to database",
		"memory usage",
		"slow query",
		"retry attempt",
		"compaction completed",
		"span completed",
	}

	// Patterns with variable-length error suffixes need looser checks.
	// Different token counts (e.g. "EOF" vs "too many open files")
	// are legitimately different templates — but same-length variants
	// that exist in both files must not show up as new/gone.
	//
	// Pre has "EOF" (7 tokens) only → 1 legit gone
	// Post has "too many open files" (10 tokens) only → 1 legit new
	// All other token-count variants exist in both files.
	configNew := countContaining(newTemplates, "failed to read config from")
	configGone := countContaining(goneTemplates, "failed to read config from")
	if configNew > 1 {
		t.Errorf("expected at most 1 genuinely new 'failed to read config' template, got %d", configNew)
	}
	if configGone > 1 {
		t.Errorf("expected at most 1 genuinely gone 'failed to read config' template, got %d", configGone)
	}

	// Pre has "disk quota exceeded"/"context deadline exceeded" that
	// don't appear in post, and post has unique variants too.
	// But same-length variants like "connection refused" must match.
	requestNew := countContaining(newTemplates, "request <UUID> failed:")
	requestGone := countContaining(goneTemplates, "request <UUID> failed:")
	if requestNew > 2 {
		t.Errorf("expected at most 2 genuinely new 'request failed' templates, got %d", requestNew)
	}
	if requestGone > 2 {
		t.Errorf("expected at most 2 genuinely gone 'request failed' templates, got %d", requestGone)
	}
	for _, kw := range sharedPatterns {
		if anyContains(newTemplates, kw) {
			t.Errorf("template containing %q should not be NEW — it exists in both files", kw)
		}
		if anyContains(goneTemplates, kw) {
			t.Errorf("template containing %q should not be GONE — it exists in both files", kw)
		}
	}
}

func TestSpanCompletedFrequencyDrop(t *testing.T) {
	r := runDiff(t)
	changedTemplates := templates(r.Changed)

	if !anyContains(changedTemplates, "span completed") {
		t.Errorf("expected 'span completed' in CHANGED (11%% → 4%%)")
	}
}

// --- helpers -----------------------------------------------------------------

func templates(entries []differ.Entry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.Template
	}
	return out
}

func anyContains(list []string, substr string) bool {
	for _, s := range list {
		if contains(s, substr) {
			return true
		}
	}
	return false
}

func countContaining(list []string, substr string) int {
	n := 0
	for _, s := range list {
		if contains(s, substr) {
			n++
		}
	}
	return n
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
