package differ

import (
	"strings"

	"github.com/naman47vyas/log-diff/internal/drain"
)

// Config controls how the differ classifies frequency changes.
type Config struct {
	// FreqRatioThreshold is the minimum ratio of relative
	// frequency change to flag a template as "changed".
	// E.g. 2.0 means a template must double or halve in
	// relative frequency to be reported.
	FreqRatioThreshold float64
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		FreqRatioThreshold: 2.0,
	}
}

// Category describes what happened to a log template across releases.
type Category string

const (
	New     Category = "new"
	Gone    Category = "gone"
	Changed Category = "changed"
)

// Entry is a single row in the diff report.
type Entry struct {
	Category    Category
	Template    string
	PreCount    int
	PostCount   int
	PreFreqPct  float64
	PostFreqPct float64
	Samples     []string
}

// Report holds the full diff result.
type Report struct {
	New     []Entry
	Gone    []Entry
	Changed []Entry

	PreTotal      int
	PostTotal     int
	PreTemplates  int
	PostTemplates int
}

// Diff compares pre-release and post-release clusters and
// classifies each template as new, gone, or changed.
func Diff(pre, post []*drain.LogCluster, cfg Config) *Report {
	preTotal := totalCount(pre)
	postTotal := totalCount(post)

	preMap := indexByTemplate(pre)
	postMap := indexByTemplate(post)

	// Build match maps: pre template -> post template and vice versa.
	// First try exact match, then fall back to fuzzy (wildcard-aware).
	preToPost := make(map[string]string)
	postToPre := make(map[string]string)

	// Pass 1: exact matches
	for tmpl := range preMap {
		if _, exists := postMap[tmpl]; exists {
			preToPost[tmpl] = tmpl
			postToPre[tmpl] = tmpl
		}
	}

	// Pass 2: fuzzy matches for unmatched templates
	for preTmpl := range preMap {
		if _, matched := preToPost[preTmpl]; matched {
			continue
		}
		for postTmpl := range postMap {
			if _, matched := postToPre[postTmpl]; matched {
				continue
			}
			if templatesMatch(preTmpl, postTmpl) {
				preToPost[preTmpl] = postTmpl
				postToPre[postTmpl] = preTmpl
				break
			}
		}
	}

	r := &Report{
		PreTotal:      preTotal,
		PostTotal:     postTotal,
		PreTemplates:  len(pre),
		PostTemplates: len(post),
	}

	// find gone and changed templates
	for preTmpl, preCluster := range preMap {
		postTmpl, matched := preToPost[preTmpl]
		if !matched {
			r.Gone = append(r.Gone, Entry{
				Category:   Gone,
				Template:   preTmpl,
				PreCount:   preCluster.Count,
				PostCount:  0,
				PreFreqPct: pct(preCluster.Count, preTotal),
				Samples:    preCluster.Samples,
			})
			continue
		}

		postCluster := postMap[postTmpl]
		preFreq := pct(preCluster.Count, preTotal)
		postFreq := pct(postCluster.Count, postTotal)

		if isSignificantChange(preFreq, postFreq, cfg.FreqRatioThreshold) {
			r.Changed = append(r.Changed, Entry{
				Category:    Changed,
				Template:    preTmpl,
				PreCount:    preCluster.Count,
				PostCount:   postCluster.Count,
				PreFreqPct:  preFreq,
				PostFreqPct: postFreq,
				Samples:     postCluster.Samples,
			})
		}
	}

	// find new templates
	for postTmpl, postCluster := range postMap {
		if _, matched := postToPre[postTmpl]; !matched {
			r.New = append(r.New, Entry{
				Category:    New,
				Template:    postTmpl,
				PreCount:    0,
				PostCount:   postCluster.Count,
				PostFreqPct: pct(postCluster.Count, postTotal),
				Samples:     postCluster.Samples,
			})
		}
	}

	return r
}

// templatesMatch returns true if two templates have the same
// token count and every position either matches exactly or
// one side has a <*> wildcard.
func templatesMatch(a, b string) bool {
	aToks := strings.Fields(a)
	bToks := strings.Fields(b)

	if len(aToks) != len(bToks) {
		return false
	}

	for i := range aToks {
		if aToks[i] == bToks[i] {
			continue
		}
		if aToks[i] == "<*>" || bToks[i] == "<*>" {
			continue
		}
		return false
	}
	return true
}

// --- helpers -----------------------------------------------------------------

func totalCount(clusters []*drain.LogCluster) int {
	total := 0
	for _, c := range clusters {
		total += c.Count
	}
	return total
}

func indexByTemplate(clusters []*drain.LogCluster) map[string]*drain.LogCluster {
	m := make(map[string]*drain.LogCluster, len(clusters))
	for _, c := range clusters {
		m[c.Template()] = c
	}
	return m
}

func pct(count, total int) float64 {
	if total == 0 {
		return 0
	}
	return (float64(count) / float64(total)) * 100
}

func isSignificantChange(preFreq, postFreq, threshold float64) bool {
	if preFreq == 0 || postFreq == 0 {
		return true
	}
	ratio := postFreq / preFreq
	return ratio >= threshold || ratio <= 1/threshold
}
