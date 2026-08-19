package survey

import (
	"strings"
	"testing"
)

func TestRenderContribQueue(t *testing.T) {
	r := Report{
		Generated:       "2026-08-18",
		BigoVersion:     "1.46.0",
		AggSmellsByRule: map[string]int{"SM3": 120, "SM1": 40},
		Targets: []Target{
			{Name: "caddy", Module: "github.com/caddyserver/caddy/v2", Commit: "0e8eb41b"},
			{Name: "gone", Skipped: "path not present on this machine"},
		},
	}
	sample := []SmellFinding{
		{Target: "caddy", Rule: "SM3", Package: "github.com/caddyserver/caddy/v2/x",
			File: "x/a.go", Line: 42, Message: "append in a loop | with a pipe"},
	}

	md := RenderContribQueue(r, sample)

	if !strings.Contains(md, "GENERATED") {
		t.Error("no generated-file warning")
	}
	if !strings.Contains(md, "40") || !strings.Contains(md, "8") {
		t.Error("the sampling parameters must appear so the draw is auditable")
	}
	if !strings.Contains(md, "x/a.go:42") {
		t.Errorf("finding position missing:\n%s", md)
	}
	if !strings.Contains(md, "caddy") {
		t.Error("target column missing")
	}
	if strings.Contains(md, "loop | with") {
		t.Error("unescaped pipe breaks the table")
	}
	if !strings.Contains(md, "gone") || !strings.Contains(md, "not present") {
		t.Error("skipped targets must be visible; a silent skip changes the population")
	}
}

func TestRenderContribQueueEmptySample(t *testing.T) {
	md := RenderContribQueue(Report{Generated: "2026-08-18", BigoVersion: "1.46.0"}, nil)
	if !strings.Contains(md, "No findings") {
		t.Errorf("an empty draw must say so explicitly, not render an empty table:\n%s", md)
	}
}
