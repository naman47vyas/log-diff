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
		want    string
		wantErr bool
	}{
		{
			name: "error line",
			line: "2026-04-03T14:00:00.000Z [ERROR] failed to connect to cache at 24.221.187.135:6379: TLS handshake timeout",
			want: "[ERROR] failed to connect to cache at 24.221.187.135:6379: TLS handshake timeout",
		},
		{
			name: "info line",
			line: "2026-04-03T14:00:01.527Z [INFO] flushed 1618 records to WAL segment /tmp/crash-dump.bin",
			want: "[INFO] flushed 1618 records to WAL segment /tmp/crash-dump.bin",
		},
		{
			name: "warn line",
			line: "2026-04-03T14:00:05.000Z [WARN] high memory usage detected",
			want: "[WARN] high memory usage detected",
		},
		{
			name: "message with brackets",
			line: "2026-04-03T14:00:00.000Z [DEBUG] processing [batch 5] complete",
			want: "[DEBUG] processing [batch 5] complete",
		},
		{
			name:    "empty line",
			line:    "",
			wantErr: true,
		},
		{
			name:    "no content after timestamp",
			line:    "2026-04-03T14:00:00.000Z",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.Parse(tt.line)

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
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBracketParserImplementsInterface(t *testing.T) {
	var _ Parser = (*BracketParser)(nil)
}
