package oracle

import (
	"path/filepath"
	"testing"

	"github.com/RomanAgaltsev/bigo/internal/kata"
)

// The canonical corpus must be BYTE-IDENTICAL through the new entry point with
// empty options. CollectWith is a widening, not a change: if this drifts, every
// literature pin in the project moved for a reason nobody asked for.
func TestCollectWithEmptyOptionsEqualsCollect(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "corpus", "testdata", "src"))
	if err != nil {
		t.Fatal(err)
	}
	want, wantWrongs, err := Collect(root)
	if err != nil {
		t.Fatal(err)
	}
	got, gotWrongs, err := CollectWith(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(wantWrongs) != len(gotWrongs) {
		t.Fatalf("wrongs: Collect %d, CollectWith %d", len(wantWrongs), len(gotWrongs))
	}
	if string(want.JSON()) != string(got.JSON()) {
		t.Error("CollectWith(root, Options{}) differs from Collect(root)")
	}
}

// The overlay must actually reach the walk. strconv.Atoi is O(1) under the kata
// profile and unresolved without it, so a fixture calling it is the difference
// between a bounded row and a top row — which is the whole reason the kata
// corpus needs its own collection.
func TestCollectWithOverlayChangesTheAnswer(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "overlaysrc"))
	if err != nil {
		t.Fatal(err)
	}
	plain, _, err := CollectWith(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := kata.Profile()
	if err != nil {
		t.Fatal(err)
	}
	spaceProfile, err := kata.SpaceProfile()
	if err != nil {
		t.Fatal(err)
	}
	overlaid, _, err := CollectWith(root, Options{Overlay: profile, SpaceOverlay: spaceProfile})
	if err != nil {
		t.Fatal(err)
	}
	if plain.TimeByStatus["top"] == 0 {
		t.Fatal("fixture must be top WITHOUT the overlay, or it proves nothing")
	}
	if overlaid.TimeByStatus["top"] != 0 {
		t.Errorf("fixture must be bounded WITH the overlay; got %+v", overlaid.TimeByStatus)
	}
}
