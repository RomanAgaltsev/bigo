package kata

import "testing"

func TestProfileParses(t *testing.T) {
	s, err := Profile()
	if err != nil {
		t.Fatalf("embedded kata profile does not parse: %v", err)
	}
	if s == nil {
		t.Fatal("Profile returned a nil set")
	}
	if err := s.Err(); err != nil {
		t.Fatalf("kata profile has a bound that does not compile: %v", err)
	}
	// Keys measured by running bigo against real kata solutions. Losing one
	// silently would make the profile stop covering the case it was built for.
	for _, key := range []string{
		"bufio.NewScanner",
		"(*bufio.Scanner).Scan",
		"(*bufio.Writer).Flush",
		"strconv.Atoi",
		"strings.Split",
		"strings.Compare",
		"cmp.Compare",
		"math/rand.Intn",
		// The hashing pair, measured against the hashtable kata 2026-08-29.
		// Both are INTERFACE dispatches, and the overlay reached neither until
		// costtable.CallKey existed.
		"(io.Writer).Write",
		"(hash.Hash32).Sum32",
	} {
		if !s.Has(key) {
			t.Errorf("kata profile is missing measured key %q", key)
		}
	}
}

func TestSpaceProfileParses(t *testing.T) {
	s, err := SpaceProfile()
	if err != nil {
		t.Fatalf("embedded kata space profile does not parse: %v", err)
	}
	if err := s.Err(); err != nil {
		t.Fatalf("kata space profile has a bound that does not compile: %v", err)
	}
	for _, key := range []string{
		"strconv.Atoi",
		"strings.Compare",
		"bufio.NewScanner",
		"(*bufio.Scanner).Text",
		"strings.Split",
	} {
		if !s.Has(key) {
			t.Errorf("kata space profile is missing measured key %q", key)
		}
	}
}

// The two profiles are separate claims, but a key priced for time and left
// unpriced for space would report a bound on one axis and unverifiable on the
// other — the exact half-delivered state this file exists to fix.
func TestSpaceProfileCoversTheTimeProfile(t *testing.T) {
	timeSet, err := Profile()
	if err != nil {
		t.Fatal(err)
	}
	spaceSet, err := SpaceProfile()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range timeSet.Entries() {
		if !spaceSet.Has(e.Key) {
			t.Errorf("key %q is priced for time but not for space", e.Key)
		}
	}
}
