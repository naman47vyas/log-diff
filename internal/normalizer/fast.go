package normalizer

import "strings"

// FastNormalizer replaces variable tokens using a single-pass
// token scanner instead of regex. Much faster for high-throughput
// log processing.
type FastNormalizer struct{}

func NewFast() *FastNormalizer {
	return &FastNormalizer{}
}
func (f *FastNormalizer) Normalize(msg string) string {
	tokens := strings.Fields(msg)
	changed := false

	for i, tok := range tokens {
		replacement := classifyToken(tok)
		if replacement != "" {
			tokens[i] = replacement
			changed = true
		}
	}

	if !changed {
		return msg
	}
	return strings.Join(tokens, " ")
}

// classifyToken checks a single token and returns a placeholder
// if it's a known variable pattern, or "" to keep it as-is.
// Order matters — most specific patterns first.
func classifyToken(tok string) string {
	if isUUID(tok) {
		return "<UUID>"
	}
	if isHex(tok) {
		return "<HEX>"
	}
	if ip := isIP(tok); ip {
		return "<IP>"
	}
	if isPath(tok) {
		return "<PATH>"
	}
	if isDuration(tok) {
		return "<DUR>"
	}
	if isSize(tok) {
		return "<SIZE>"
	}
	if isNumber(tok) {
		return "<NUM>"
	}
	return ""
}

// --- detectors ---------------------------------------------------------------

// isUUID checks for 8-4-4-4-12 hex pattern: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
func isUUID(tok string) bool {
	if len(tok) != 36 {
		return false
	}
	// dashes at positions 8, 13, 18, 23
	if tok[8] != '-' || tok[13] != '-' || tok[18] != '-' || tok[23] != '-' {
		return false
	}
	for i, c := range tok {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !isHexChar(byte(c)) {
			return false
		}
	}
	return true
}

// isHex checks for 32+ contiguous hex characters (SHA-256, trace IDs).
func isHex(tok string) bool {
	if len(tok) < 32 {
		return false
	}
	for i := 0; i < len(tok); i++ {
		if !isHexChar(tok[i]) {
			return false
		}
	}
	return true
}

// isIP checks for IPv4 with optional :port.
// Matches: 192.168.1.1, 10.0.0.5:8080
func isIP(tok string) bool {
	// strip trailing punctuation like colon at end of "10.0.0.1:"
	clean := tok
	if len(clean) > 0 && clean[len(clean)-1] == ':' {
		clean = clean[:len(clean)-1]
	}

	// split host:port
	host := clean
	if colonIdx := strings.LastIndex(clean, ":"); colonIdx != -1 {
		port := clean[colonIdx+1:]
		if allDigits(port) && len(port) > 0 {
			host = clean[:colonIdx]
		}
	}

	// validate 4 octets
	parts := strings.Split(host, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		if len(p) == 0 || len(p) > 3 || !allDigits(p) {
			return false
		}
	}
	return true
}

// isPath checks for unix-style paths with at least two segments.
// Matches: /var/log/app.log, /api/v1/users
func isPath(tok string) bool {
	if len(tok) < 2 || tok[0] != '/' {
		return false
	}
	// need at least one more slash for two segments
	return strings.IndexByte(tok[1:], '/') != -1
}

// isDuration checks for number followed by a time unit suffix.
// Matches: 500ms, 1.5s, 200µs, 30m, 2h, 100ns, 50us
func isDuration(tok string) bool {
	// try each suffix
	suffixes := []string{"ms", "µs", "us", "ns", "s", "m", "h"}
	for _, suf := range suffixes {
		if strings.HasSuffix(tok, suf) {
			num := tok[:len(tok)-len(suf)]
			if isNumericStr(num) {
				return true
			}
		}
	}
	return false
}

// isSize checks for number followed by a byte size suffix.
// Matches: 512KB, 1.2GB, 4MB, 16TB, 1KiB, 2GiB
func isSize(tok string) bool {
	suffixes := []string{"KiB", "MiB", "GiB", "TiB", "KB", "MB", "GB", "TB"}
	for _, suf := range suffixes {
		if strings.HasSuffix(tok, suf) {
			num := tok[:len(tok)-len(suf)]
			if isNumericStr(num) {
				return true
			}
		}
	}
	return false
}

// isNumber checks if the entire token is a pure number (int or float).
// Matches: 42, 3.14, -7, -0.5
// Does NOT match: abc123, 192.168.1.1, 500ms
func isNumber(tok string) bool {
	return isNumericStr(tok)
}

// --- helpers -----------------------------------------------------------------

func isHexChar(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func allDigits(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// isNumericStr checks if s is a valid integer or float, optionally negative.
func isNumericStr(s string) bool {
	if len(s) == 0 {
		return false
	}
	start := 0
	if s[0] == '-' {
		start = 1
		if len(s) == 1 {
			return false
		}
	}
	dotSeen := false
	for i := start; i < len(s); i++ {
		if s[i] == '.' {
			if dotSeen {
				return false
			}
			dotSeen = true
			continue
		}
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
