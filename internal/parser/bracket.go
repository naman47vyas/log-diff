package parser

import (
	"regexp"
)

// bracketRe matches: <timestamp> [<SEVERITY>] <message>
// Example: 2026-04-03T14:00:00.000Z [ERROR] failed to connect...
var brackerRe = regexp.MustCompile(`^(\S+)\s+\[(\w+)\]\s+(.+)$`)

type BracketParser struct{}

func NewBracketParser() *BracketParser {
	return &BracketParser{}
}

func (p *BracketParser) Parse(line string) (*LogEntry, error) {
	matches := brackerRe.FindStringSubmatch(line)

	if matches == nil {
		return nil, ErrUnparseable
	}

	return &LogEntry{
		Timestamp: matches[1],
		Severity:  matches[2],
		Message:   matches[3],
	}, nil
}
