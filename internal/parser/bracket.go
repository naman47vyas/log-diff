package parser

import (
	regexp "github.com/grafana/regexp"
)

// bracketRe matches: <timestamp> [<SEVERITY>] <message>
// It captures everything after the timestamp as a single group.
// Example: 2026-04-03T14:00:00.000Z [ERROR] failed to connect...
var bracketRe = regexp.MustCompile(`^\S+\s+(.+)$`)

type BracketParser struct{}

func NewBracketParser() *BracketParser {
	return &BracketParser{}
}

func (p *BracketParser) Parse(line string) (string, error) {
	matches := bracketRe.FindStringSubmatch(line)
	if matches == nil {
		return "", ErrUnparseable
	}
	return matches[1], nil
}
