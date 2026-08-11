// Package trustinit is the trust-file generator fixture: one blocker that CAN
// be named as a trust key and one that cannot.
package trustinit

import "os"

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
