package differ

import "github.com/naman47vyas/log-diff/internal/drain"

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

	r := &Report{
		PreTotal:      preTotal,
		PostTotal:     postTotal,
		PreTemplates:  len(pre),
		PostTemplates: len(post),
	}

	// find gone and changed templates
	for tmpl, preCluster := range preMap {
		postCluster, exists := postMap[tmpl]
		if !exists {
			r.Gone = append(r.Gone, Entry{
				Category:   Gone,
				Template:   tmpl,
				PreCount:   preCluster.Count,
				PostCount:  0,
				PreFreqPct: pct(preCluster.Count, preTotal),
				Samples:    preCluster.Samples,
			})
			continue
		}

		// exists in both — check frequency change
		preFreq := pct(preCluster.Count, preTotal)
		postFreq := pct(postCluster.Count, postTotal)

		if isSignificantChange(preFreq, postFreq, cfg.FreqRatioThreshold) {
			r.Changed = append(r.Changed, Entry{
				Category:    Changed,
				Template:    tmpl,
				PreCount:    preCluster.Count,
				PostCount:   postCluster.Count,
				PreFreqPct:  preFreq,
				PostFreqPct: postFreq,
				Samples:     postCluster.Samples,
			})
		}
	}

	// find new templates
	for tmpl, postCluster := range postMap {
		if _, exists := preMap[tmpl]; !exists {
			r.New = append(r.New, Entry{
				Category:    New,
				Template:    tmpl,
				PreCount:    0,
				PostCount:   postCluster.Count,
				PostFreqPct: pct(postCluster.Count, postTotal),
				Samples:     postCluster.Samples,
			})
		}
	}

	return r
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
