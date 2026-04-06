package normalizer

import "testing"

func TestNormalize(t *testing.T) {
	n := New()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "IP and duration",
			in:   "Connection from 192.168.1.45 timed out after 30s",
			want: "Connection from <IP> timed out after <DUR>",
		},
		{
			name: "UUID in message",
			in:   "Processing request abc12345-def6-7890-abcd-ef1234567890 for user",
			want: "Processing request <UUID> for user",
		},
		{
			name: "URL replacement",
			in:   "Fetching config from https://api.example.com/v2/config?token=xyz",
			want: "Fetching config from <URL>",
		},
		{
			name: "email",
			in:   "Notification sent to admin@example.com successfully",
			want: "Notification sent to <EMAIL> successfully",
		},
		{
			name: "ISO timestamp in body",
			in:   "Snapshot created at 2024-06-15T10:30:00Z with size 512KB",
			want: "Snapshot created at <DATETIME> with size <SIZE>",
		},
		{
			name: "file path",
			in:   "Failed to read /var/log/app/errors.log",
			want: "Failed to read <PATH>",
		},
		{
			name: "numbers and percentages",
			in:   "CPU at 85.3% memory usage 2048 bytes",
			want: "CPU at <PCT> memory usage <NUM> bytes",
		},
		{
			name: "quoted strings",
			in:   `User "john_doe" triggered action 'deploy'`,
			want: `User "<STR>" triggered action '<STR>'`,
		},
		{
			name: "hex hash",
			in:   "Checksum mismatch: got 9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
			want: "Checksum mismatch: got <HEX>",
		},
		{
			name: "IP with port",
			in:   "Listening on 0.0.0.0:8080",
			want: "Listening on <IP>",
		},
		{
			name: "mixed realistic log",
			in:   "Request from 10.0.0.5 to https://api.svc.local/health took 245ms status 200",
			want: "Request from <IP> to <URL> took <DUR> status <NUM>",
		},
		{
			name: "pure text unchanged",
			in:   "Application started successfully",
			want: "Application started successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := n.Normalize(tt.in)
			if got != tt.want {
				t.Errorf("\n  input: %s\n   got:  %s\n  want:  %s", tt.in, got, tt.want)
			}
		})
	}
}
