// Package assumeok exercises the -assume flag: the O(1) budget on Lookup can
// hold only when an assumption file prices os.Getenv, so a silent run proves
// the assumption was honored.
package assumeok

import "os"

//bigo:max O(1)
func Lookup(k string) string { return os.Getenv(k) }
