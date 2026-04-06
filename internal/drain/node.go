package drain

// Wildcard is the placeholder token used in templates for
// variable positions, and as the child key for tokens that
// look like parameters (e.g. digits).
const Wildcard = "<*>"

// Node is a single node in the Drain parse tree.
// Internal nodes use Children to route by token value.
// Leaf nodes use Clusters to hold candidate log groups.
type Node struct {
	Children map[string]*Node
	Clusters []*LogCluster
}

// NewNode returns an empty internal node.
func NewNode() *Node {
	return &Node{
		Children: make(map[string]*Node),
	}
}
