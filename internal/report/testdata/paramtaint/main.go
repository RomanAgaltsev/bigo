// Package paramtaint has two identical consumers of one parametric function.
// Both graduate through the same assumption, so both must be tainted.
package paramtaint

import "example.com/paramtaint/helper"

func A(m map[int]int) { helper.Run(func() {}, m) }

func B(m map[int]int) { helper.Run(func() {}, m) }
