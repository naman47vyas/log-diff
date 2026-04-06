package drain

const maxSamples = 3

// LogCluster represents a group of log messages that share
// the same template structure. Positions that vary across
// messages are replaced with the wildcard token.
type LogCluster struct {
	ID      int
	Tokens  []string
	Count   int
	Samples []string
}

// NewLogCluster creates a cluster from the tokens of the first
// message that defines it. The raw message is stored as the
// first sample.
func NewLogCluster(id int, tokens []string, raw string) *LogCluster {
	return &LogCluster{
		ID:      id,
		Tokens:  tokens,
		Count:   1,
		Samples: []string{raw},
	}
}

// Template returns the cluster's template as a single string.
func (c *LogCluster) Template() string {
	result := ""
	for i, t := range c.Tokens {
		if i > 0 {
			result += " "
		}
		result += t
	}
	return result
}

// AddSample increments the match count and stores the raw
// message if the sample buffer isn't full yet.
func (c *LogCluster) AddSample(raw string) {
	c.Count++
	if len(c.Samples) < maxSamples {
		c.Samples = append(c.Samples, raw)
	}
}
