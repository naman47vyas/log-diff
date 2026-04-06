package normalizer

import (
	"strings"

	"regexp"
)

// Norm is the interface that any normalizer must satisfy.
type Norm interface {
	Normalize(msg string) string
}

// rule maps a regex pattern to a placeholder token.
type rule struct {
	re          *regexp.Regexp
	placeholder string
}

// Normalizer replaces variable tokens in a log message with stable
// placeholders, producing a "template" that represents the log's shape.
type Normalizer struct {
	rules []rule
}

// New returns a Normalizer loaded with the default rule chain.
func New() *Normalizer {
	return &Normalizer{
		rules: defaultRules(),
	}
}

func (n *Normalizer) Normalize(msg string) string {
	out := msg
	for _, r := range n.rules {
		out = r.re.ReplaceAllString(out, r.placeholder)
	}
	out = collapseSpaces(out)
	return out
}

func defaultRules() []rule {
	return []rule{
		r(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`, "<UUID>"),
		r(`\b[0-9a-fA-F]{32,}\b`, "<HEX>"),
		r(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`, "<EMAIL>"),
		r(`https?://[^\s"'\]>]+`, "<URL>"),
		r(`(?:/[a-zA-Z0-9._\-]+){2,}`, "<PATH>"),
		r(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}(:\d+)?\b`, "<IP>"),
		r(`\b[0-9a-fA-F]{1,4}(:[0-9a-fA-F]{1,4}){7}\b`, "<IP>"),
		r(`\b[0-9a-fA-F]{2}(:[0-9a-fA-F]{2}){5}\b`, "<MAC>"),
		r(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}([.\d]*)([Zz]|[+\-]\d{2}:?\d{2})?`, "<DATETIME>"),
		r(`\b\d{4}[/\-]\d{2}[/\-]\d{2}\b`, "<DATE>"),
		r(`\b\d{2}[/\-]\d{2}[/\-]\d{4}\b`, "<DATE>"),
		r(`\b\d{2}:\d{2}:\d{2}([.\d]*)?\b`, "<TIME>"),
		r(`\b\d+(\.\d+)?\s*(ms|µs|us|ns|s|m|h)\b`, "<DUR>"),
		r(`\b\d+(\.\d+)?\s*[KMGT]i?B\b`, "<SIZE>"),
		r(`\b\d+(\.\d+)?%`, "<PCT>"),
		r(`\b0x[0-9a-fA-F]+\b`, "<HEX>"),
		r(`\b-?\d+(\.\d+)?\b`, "<NUM>"),
		r(`"[^"]*"`, `"<STR>"`),
		r(`'[^']*'`, `'<STR>'`),
	}
}

func r(pattern, placeholder string) rule {
	return rule{
		re:          regexp.MustCompile(pattern),
		placeholder: placeholder,
	}
}

var multiSpace = regexp.MustCompile(`\s{2,}`)

func collapseSpaces(s string) string {
	return strings.TrimSpace(multiSpace.ReplaceAllString(s, " "))
}
