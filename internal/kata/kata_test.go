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
	} {
		if !s.Has(key) {
			t.Errorf("kata profile is missing measured key %q", key)
		}
	}
}
