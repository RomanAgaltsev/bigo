// Package trustinitkata is the trust-file generator fixture for the kata cost
// model: its blocker is a call the KATA profile prices but the curated table
// does not, so the ranking must differ between the two models.
package trustinitkata

import "bufio"

// Write is blocked ONLY by the bufio write. Under the default model that is a
// real, offerable trust candidate. Under -kata the profile already prices it —
// K-1 says output is not graded work — so offering it would spend the user's
// reasoning on a line that changes nothing.
func Write(w *bufio.Writer, s string) error {
	_, err := w.WriteString(s)
	return err
}

// AlsoWrite shares the blocker so the count can rise above one.
func AlsoWrite(w *bufio.Writer, s string) error {
	_, err := w.WriteString(s + "!")
	return err
}
