// Package assumefix is the assumption-mechanism fixture: Blocked and Caller
// are unverifiable solely because os.Getenv is unpriced, so one assumption
// graduates both; Clean is the no-taint control.
package assumefix

import "os"

func Blocked(k string) string { return os.Getenv(k) }

func Caller(k string) string { return Blocked(k) }

func Clean(xs []int) int { return len(xs) }
