// Package trustinit is the trust-file generator fixture: one blocker that CAN
// be named as a trust key and one that cannot.
package trustinit

import (
	"os"
	"strconv"
)

// Reader is here only to produce an interface-dispatch cause.
type Reader interface{ Read() int }

// BlockedByStatic is blocked by an unpriced stdlib function, which has a static
// callee and therefore a nameable key.
func BlockedByStatic(k string) string { return os.Getenv(k) }

// BlockedByInterface is blocked by an interface method, which has no static
// callee and therefore no key a trust file could carry.
func BlockedByInterface(r Reader) int { return r.Read() }

// AlsoBlockedByStatic shares BlockedByStatic's blocker, so the generator can
// report a count above one.
func AlsoBlockedByStatic(k string) string { return os.Getenv(k) + "x" }

// Clean is the control: it needs no trust at all.
func Clean(xs []int) int { return len(xs) }

// mystery returns a string whose length bigo cannot resolve at the call site.
func mystery() string { return "x" }

// BlockedByArgSize is blocked by an ARGUMENT SIZE, not by a missing entry:
// strconv.Atoi is already priced, so the curated table answers this key and a
// trust entry for it would be shadowed and contribute nothing.
func BlockedByArgSize() (int, error) { return strconv.Atoi(mystery()) }

// Drain is FIRST-PARTY, GENERIC and unverifiable — the 2026-08-12 review's F2.
//
// A call to it renders the instantiation in the cause sentence while the
// document records the declaration, so the frontier walk used to miss the join,
// score the hop as a blocker, and offer the caller's own module code as a trust
// key. Code the user can edit wants //bigo:cost, never a trust entry.
func Drain[T any](xs []T) int {
	n := 0
	for len(xs) > 0 {
		xs = xs[1:]
		n++
	}
	return n
}

// BlockedByOwnGeneric calls it. Its real blocker is Drain's loop, two hops
// away, and a loop has no cost-table key — so nothing here is offerable.
func BlockedByOwnGeneric(xs []string) int { return Drain(xs) }
