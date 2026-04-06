package parser

import "errors"

// ErrUnparseable is returned when a log line doesn't match
// the expected format.
var ErrUnparseable = errors.New("unparseable log line")

// Holds the structured parts of raw log line
type LogEntry struct {
	Timestamp string
	Severity  string
	Message   string
}

// Interface because we might see there would be different log line formats
// Implementations would hold specific formats. Json/brackets,Otlp etc
type Parser interface {
	Parse(line string) (*LogEntry, error)
}
