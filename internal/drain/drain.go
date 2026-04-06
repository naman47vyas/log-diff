package drain

import (
	"strconv"
	"strings"
)

// Config holds tuning parameters for the Drain algorithm.
type Config struct {
	// Depth controls how many token layers the tree uses for
	// routing before reaching leaf nodes. Minimum 3.
	// The layers are: root → token-count → (Depth-2) token layers → leaves.
	Depth int

	// SimThreshold is the minimum ratio of matching tokens
	// required for a message to join an existing cluster.
	// Range 0.0–1.0. Lower = more aggressive merging.
	SimThreshold float64

	// MaxChildren is the maximum number of children per
	// internal node. Excess tokens route to the wildcard edge.
	MaxChildren int
}

// DefaultConfig returns the same defaults used by IBM Drain3:
// depth=4, similarity=0.4, maxChildren=100.
func DefaultConfig() Config {
	return Config{
		Depth:        4,
		SimThreshold: 0.7,
		MaxChildren:  100,
	}
}

// Drain is an online log template miner using a fixed-depth
// parse tree to cluster log messages by their structure.
type Drain struct {
	config   Config
	root     *Node
	clusters []*LogCluster
	nextID   int
}

// New creates a Drain instance with the given config.
func New(cfg Config) *Drain {
	if cfg.Depth < 3 {
		cfg.Depth = 3
	}
	return &Drain{
		config: cfg,
		root:   NewNode(),
		nextID: 1,
	}
}

// Clusters returns all discovered log clusters.
func (d *Drain) Clusters() []*LogCluster {
	return d.clusters
}

// Train processes a single log message, either matching it to
// an existing cluster or creating a new one. Returns the
// cluster the message was assigned to.
func (d *Drain) Train(raw string) *LogCluster {
	tokens := tokenize(raw)
	if len(tokens) == 0 {
		return nil
	}

	// Step 1: route to the length node
	lenKey := strconv.Itoa(len(tokens))
	lenNode, ok := d.root.Children[lenKey]
	if !ok {
		lenNode = NewNode()
		d.root.Children[lenKey] = lenNode
	}

	// Step 2: walk token layers to reach a leaf
	leaf := d.walkTree(lenNode, tokens)

	// Step 3: find best matching cluster at the leaf
	bestCluster, bestScore := d.bestMatch(leaf, tokens)

	if bestCluster != nil && bestScore >= d.config.SimThreshold {
		// match — update template and add sample
		bestCluster.AddSample(raw)
		updateTemplate(bestCluster, tokens)
		return bestCluster
	}

	// no match — create a new cluster
	cluster := NewLogCluster(d.nextID, copyTokens(tokens), raw)
	d.nextID++
	leaf.Clusters = append(leaf.Clusters, cluster)
	d.clusters = append(d.clusters, cluster)
	return cluster
}

// bestMatch finds the cluster in a leaf node with the highest
// similarity to the incoming tokens. Returns nil if the leaf
// has no clusters.
func (d *Drain) bestMatch(leaf *Node, tokens []string) (*LogCluster, float64) {
	var best *LogCluster
	bestScore := -1.0

	for _, c := range leaf.Clusters {
		score := similarity(tokens, c.Tokens)
		if score > bestScore {
			bestScore = score
			best = c
		}
	}
	return best, bestScore
}

// similarity returns the ratio of matching constant tokens
// between a message and a cluster template. Wildcard positions
// are not counted as matches.
func similarity(msg, template []string) float64 {
	if len(msg) != len(template) {
		return 0
	}
	matches := 0
	for i := range msg {
		if template[i] == Wildcard {
			continue
		}
		if msg[i] == template[i] {
			matches++
		}
	}
	return float64(matches) / float64(len(msg))
}

// updateTemplate replaces positions in the cluster template
// with <*> where the incoming message differs.
func updateTemplate(c *LogCluster, tokens []string) {
	for i := range tokens {
		if c.Tokens[i] != tokens[i] {
			c.Tokens[i] = Wildcard
		}
	}
}

// copyTokens returns a shallow copy so the cluster owns its
// own slice and isn't affected if the caller mutates the original.
func copyTokens(tokens []string) []string {
	cp := make([]string, len(tokens))
	copy(cp, tokens)
	return cp
}

// tokenize splits a log message into tokens by whitespace.
func tokenize(msg string) []string {
	return strings.Fields(msg)
}

// isParam returns true if a token looks like a variable rather
// than a constant word. This includes tokens that start with a
// digit (e.g. "192", "3000") or with '<' (e.g. "<IP>", "<UUID>"
// from our regex normalizer).
func isParam(token string) bool {
	if len(token) == 0 {
		return false
	}
	return token[0] >= '0' && token[0] <= '9' || token[0] == '<'
}

// walkTree descends from the length node through (Depth-2)
// token layers and returns the leaf node.
func (d *Drain) walkTree(lenNode *Node, tokens []string) *Node {
	cur := lenNode
	// walk at most Depth-2 layers, and at most len(tokens) layers
	layers := d.config.Depth - 2
	if layers > len(tokens) {
		layers = len(tokens)
	}

	for i := 0; i < layers; i++ {
		tok := tokens[i]

		if isParam(tok) {
			// parameter token — always route to wildcard edge
			child, ok := cur.Children[Wildcard]
			if !ok {
				child = NewNode()
				cur.Children[Wildcard] = child
			}
			cur = child
		} else if child, ok := cur.Children[tok]; ok {
			// exact match exists — follow it
			cur = child
		} else if len(cur.Children) < d.config.MaxChildren {
			// room for a new child — create it
			child := NewNode()
			cur.Children[tok] = child
			cur = child
		} else {
			// over MaxChildren — overflow to wildcard edge
			child, ok := cur.Children[Wildcard]
			if !ok {
				child = NewNode()
				cur.Children[Wildcard] = child
			}
			cur = child
		}
	}

	return cur
}
