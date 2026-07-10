package plugin

import "testing"

func TestBuildAnalyzers(t *testing.T) {
	p, err := New(nil)
	if err != nil {
		t.Fatal(err)
	}
	as, err := p.BuildAnalyzers()
	if err != nil {
		t.Fatal(err)
	}
	if len(as) != 1 || as[0].Name != "bigo" {
		t.Fatalf("BuildAnalyzers = %v, want [bigo]", as)
	}
}
