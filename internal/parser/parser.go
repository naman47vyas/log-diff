package parser

import "errors"

// ErrUnparseable is returned when a log line doesn't match
// the expected format.
var ErrUnparseable = errors.New("unparseable log line")

// Parser extracts the meaningful content from a raw log line,
// stripping metadata like timestamps that would prevent clustering.
type Parser interface {
	Parse(line string) (string, error)
}
