package normalizer

import (
	"regexp"
	"strings"
)

type rule struct {
	re          *regexp.Regexp
	placeholder string
}

type Normalizer struct {
	rules []rule
}

func New() *Normalizer {
	return &Normalizer{
		rules: defaultRules(),
	}
}

// Normalize takes a raw log message (timestamp and severity already
// stripped) and returns the templatized form.
//
//	in:  "Connection from 192.168.1.45 timed out after 30s"
//	out: "Connection from <IP> timed out after <NUM>s"
func (n *Normalizer) Normalize(msg string) string {
	out := msg
	for _, r := range n.rules {
		out = r.re.ReplaceAllString(out, r.placeholder)
	}
	// collapse repeated placeholders separated only by whitespace/commas
	// collapse multiple spaces that replacements may leave behind
	out = collapseSpaces(out)
	return out
}

// --- default rules (order matters) -------------------------------------------

func defaultRules() []rule {
	return []rule{
		// ── identifiers / structured tokens ──
		// UUID  (8-4-4-4-12 hex)
		r(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`, "<UUID>"),
		// SHA-256 / long hex (>=32 hex chars)
		r(`\b[0-9a-fA-F]{32,}\b`, "<HEX>"),
		// Email addresses
		r(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`, "<EMAIL>"),
		// URLs  (http / https)
		r(`https?://[^\s"'\]>]+`, "<URL>"),
		// File paths (unix-style, at least two segments)
		r(`(?:/[a-zA-Z0-9._\-]+){2,}`, "<PATH>"),

		// ── network ──
		// IPv4 with optional port
		r(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}(:\d+)?\b`, "<IP>"),
		// IPv6 (simplified — colon-hex groups)
		r(`\b[0-9a-fA-F]{1,4}(:[0-9a-fA-F]{1,4}){7}\b`, "<IP>"),
		// MAC address
		r(`\b[0-9a-fA-F]{2}(:[0-9a-fA-F]{2}){5}\b`, "<MAC>"),

		// ── timestamps / dates inside the message body ──
		// ISO-8601  2024-01-02T15:04:05...
		r(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}([.\d]*)([Zz]|[+\-]\d{2}:?\d{2})?`, "<DATETIME>"),
		// Date only  2024-01-02 or 01/02/2024
		r(`\b\d{4}[/\-]\d{2}[/\-]\d{2}\b`, "<DATE>"),
		r(`\b\d{2}[/\-]\d{2}[/\-]\d{4}\b`, "<DATE>"),
		// Time only  15:04:05
		r(`\b\d{2}:\d{2}:\d{2}([.\d]*)?\b`, "<TIME>"),

		// ── quantities ──
		// Duration literals  30ms, 1.5s, 200µs
		r(`\b\d+(\.\d+)?\s*(ms|µs|us|ns|s|m|h)\b`, "<DUR>"),
		// Byte sizes  512KB, 1.2GB
		r(`\b\d+(\.\d+)?\s*[KMGT]i?B\b`, "<SIZE>"),
		// Percentages
		r(`\b\d+(\.\d+)?%`, "<PCT>"),
		// Hex numbers prefixed with 0x
		r(`\b0x[0-9a-fA-F]+\b`, "<HEX>"),
		// General numbers (int or float, including negatives)
		r(`\b-?\d+(\.\d+)?\b`, "<NUM>"),

		// ── quoted strings ──
		// Double-quoted
		r(`"[^"]*"`, `"<STR>"`),
		// Single-quoted
		r(`'[^']*'`, `'<STR>'`),
	}
}

// --- helpers -----------------------------------------------------------------

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
