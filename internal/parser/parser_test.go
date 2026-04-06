package parser

import (
	"errors"
	"testing"
)

func TestBracketParser(t *testing.T) {
	p := NewBracketParser()

	tests := []struct {
		name    string
		line    string
		wantTS  string
		wantSev string
		wantMsg string
		wantErr bool
	}{
		{
			name:    "error line",
			line:    "2026-04-03T14:00:00.000Z [ERROR] failed to connect to cache at 24.221.187.135:6379: TLS handshake timeout",
			wantTS:  "2026-04-03T14:00:00.000Z",
			wantSev: "ERROR",
			wantMsg: "failed to connect to cache at 24.221.187.135:6379: TLS handshake timeout",
		},
		{
			name:    "info line",
			line:    "2026-04-03T14:00:01.527Z [INFO] flushed 1618 records to WAL segment /tmp/crash-dump.bin",
			wantTS:  "2026-04-03T14:00:01.527Z",
			wantSev: "INFO",
			wantMsg: "flushed 1618 records to WAL segment /tmp/crash-dump.bin",
		},
		{
			name:    "warn line",
			line:    "2026-04-03T14:00:05.000Z [WARN] high memory usage detected",
			wantTS:  "2026-04-03T14:00:05.000Z",
			wantSev: "WARN",
			wantMsg: "high memory usage detected",
		},
		{
			name:    "message with brackets",
			line:    "2026-04-03T14:00:00.000Z [DEBUG] processing [batch 5] complete",
			wantTS:  "2026-04-03T14:00:00.000Z",
			wantSev: "DEBUG",
			wantMsg: "processing [batch 5] complete",
		},
		{
			name:    "empty line",
			line:    "",
			wantErr: true,
		},
		{
			name:    "missing severity brackets",
			line:    "2026-04-03T14:00:00.000Z ERROR some message",
			wantErr: true,
		},
		{
			name:    "no message",
			line:    "2026-04-03T14:00:00.000Z [INFO]",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := p.Parse(tt.line)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !errors.Is(err, ErrUnparseable) {
					t.Errorf("expected ErrUnparseable, got %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if entry.Timestamp != tt.wantTS {
				t.Errorf("timestamp = %q, want %q", entry.Timestamp, tt.wantTS)
			}
			if entry.Severity != tt.wantSev {
				t.Errorf("severity = %q, want %q", entry.Severity, tt.wantSev)
			}
			if entry.Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", entry.Message, tt.wantMsg)
			}
		})
	}
}

func TestBracketParserImplementsInterface(t *testing.T) {
	// compile-time check that BracketParser satisfies Parser
	var _ Parser = (*BracketParser)(nil)
}
