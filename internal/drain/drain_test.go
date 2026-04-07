package drain

import "testing"

func TestTokenize(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"connected to server", 3},
		{"  extra   spaces  ", 2},
		{"single", 1},
		{"", 0},
	}
	for _, tt := range tests {
		got := tokenize(tt.in)
		if len(got) != tt.want {
			t.Errorf("tokenize(%q) got %d tokens, want %d", tt.in, len(got), tt.want)
		}
	}
}

func TestIsParam(t *testing.T) {
	tests := []struct {
		token string
		want  bool
	}{
		{"192.168.1.1", true},
		{"<IP>", true},
		{"<UUID>", true},
		{"connected", false},
		{"user", false},
		{"ERROR", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isParam(tt.token)
		if got != tt.want {
			t.Errorf("isParam(%q) = %v, want %v", tt.token, got, tt.want)
		}
	}
}

func TestSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		msg      []string
		template []string
		want     float64
	}{
		{
			name:     "exact match",
			msg:      []string{"user", "logged", "in"},
			template: []string{"user", "logged", "in"},
			want:     1.0,
		},
		{
			name:     "one wildcard",
			msg:      []string{"user", "admin", "logged", "in"},
			template: []string{"user", "<*>", "logged", "in"},
			want:     0.75,
		},
		{
			name:     "all wildcards",
			msg:      []string{"a", "b", "c"},
			template: []string{"<*>", "<*>", "<*>"},
			want:     0.0,
		},
		{
			name:     "no match",
			msg:      []string{"foo", "bar"},
			template: []string{"baz", "qux"},
			want:     0.0,
		},
		{
			name:     "length mismatch",
			msg:      []string{"a", "b"},
			template: []string{"a", "b", "c"},
			want:     0.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := similarity(tt.msg, tt.template)
			if got != tt.want {
				t.Errorf("similarity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTrainCreatesCluster(t *testing.T) {
	d := New(DefaultConfig())
	c := d.Train("connected to server")

	if c == nil {
		t.Fatal("Train returned nil")
	}
	if c.ID != 1 {
		t.Errorf("cluster ID = %d, want 1", c.ID)
	}
	if c.Count != 1 {
		t.Errorf("cluster Count = %d, want 1", c.Count)
	}
	if c.Template() != "connected to server" {
		t.Errorf("template = %q, want %q", c.Template(), "connected to server")
	}
	if len(d.Clusters()) != 1 {
		t.Errorf("cluster count = %d, want 1", len(d.Clusters()))
	}
}

func TestTrainMergesSimilarMessages(t *testing.T) {
	d := New(DefaultConfig())
	// 4 tokens, 1 differs → similarity 3/4 = 0.75, above 0.7 threshold
	d.Train("connected to server at 10.0.0.1")
	d.Train("connected to server at 10.0.0.2")
	c := d.Train("connected to server at 10.0.0.3")

	if len(d.Clusters()) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(d.Clusters()))
	}
	if c.Count != 3 {
		t.Errorf("count = %d, want 3", c.Count)
	}
	if c.Template() != "connected to server at <*>" {
		t.Errorf("template = %q, want %q", c.Template(), "connected to server at <*>")
	}
}

func TestTrainSeparatesDifferentPatterns(t *testing.T) {
	d := New(DefaultConfig())
	d.Train("connected to server")
	d.Train("user admin logged in")

	if len(d.Clusters()) != 2 {
		t.Errorf("expected 2 clusters, got %d", len(d.Clusters()))
	}
}

func TestTrainWithNormalizedTokens(t *testing.T) {
	d := New(DefaultConfig())
	// simulates messages after regex normalizer has run
	d.Train("connection from <IP> established on <NUM>")
	d.Train("connection from <IP> established on <NUM>")

	if len(d.Clusters()) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(d.Clusters()))
	}
	c := d.Clusters()[0]
	if c.Count != 2 {
		t.Errorf("count = %d, want 2", c.Count)
	}
	if c.Template() != "connection from <IP> established on <NUM>" {
		t.Errorf("template = %q", c.Template())
	}
}

func TestTrainSamplesCapAtMax(t *testing.T) {
	d := New(DefaultConfig())
	for i := 0; i < 10; i++ {
		d.Train("repeated message")
	}

	c := d.Clusters()[0]
	if len(c.Samples) != maxSamples {
		t.Errorf("samples = %d, want %d", len(c.Samples), maxSamples)
	}
	if c.Count != 10 {
		t.Errorf("count = %d, want 10", c.Count)
	}
}

func TestTrainEmptyMessage(t *testing.T) {
	d := New(DefaultConfig())
	c := d.Train("")

	if c != nil {
		t.Error("expected nil for empty message")
	}
	if len(d.Clusters()) != 0 {
		t.Error("expected no clusters for empty message")
	}
}

func TestTrainTemplateGeneralizes(t *testing.T) {
	d := New(DefaultConfig())
	// 7 tokens, 2 differ → similarity 5/7 ≈ 0.71, above 0.7 threshold
	d.Train("error occurred in module auth with code 500")
	d.Train("error occurred in module payments with code 404")

	if len(d.Clusters()) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(d.Clusters()))
	}
	want := "error occurred in module <*> with code <*>"
	got := d.Clusters()[0].Template()
	if got != want {
		t.Errorf("template = %q, want %q", got, want)
	}
}
