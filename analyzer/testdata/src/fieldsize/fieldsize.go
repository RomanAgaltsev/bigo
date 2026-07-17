// Package fieldsize pins the field-path size islands (Rules V and P) and
// every rejection that guards the entry-value semantics.
package fieldsize

type Config struct{ items []int }

type S struct {
	items []int
	limit int
	cfg   Config
}

func work(x int) int { return x }

func mutate(p *[]int) { *p = append(*p, 0) }

// Rule V: value receiver — the copy is unreachable from callees, so calls in
// the body are harmless.
//
//bigo:max O(n) where n=len(s.items)
func (s S) SumValue() int {
	t := 0
	for i := 0; i < len(s.items); i++ {
		t += work(s.items[i])
	}
	return t
}

// Rule P: hoisted length in a call-free preamble.
//
//bigo:max O(n) where n=len(s.items)
func (s *S) SumHoisted() int {
	n := len(s.items)
	t := 0
	for i := 0; i < n; i++ {
		t += i
	}
	return t
}

// Rule P: inline re-read with a call-free, store-free loop body.
//
//bigo:max O(n) where n=len(s.items)
func (s *S) SumInline() int {
	t := 0
	for i := 0; i < len(s.items); i++ {
		t += s.items[i]
	}
	return t
}

// Rule P rejection: a call in the loop body could mutate *s between
// iterations, so the re-read length is not the entry value.
//
//bigo:max O(n) where n=len(s.items)
func (s *S) SumCalls() int { // want `cannot verify budget O\(len\(s\.items\)\)`
	t := 0
	for i := 0; i < len(s.items); i++ {
		t += work(s.items[i])
	}
	return t
}

// Store rejection — the wrong-bound reproducer: the function grows the field
// before the loop, so a bound in entry-value terms would be wrong.
//
//bigo:max O(n) where n=len(s.items)
func (s *S) GrowThenScan(v int) int { // want `cannot verify budget O\(len\(s\.items\)\)`
	s.items = append(s.items, v)
	t := 0
	for i := 0; i < len(s.items); i++ {
		t++
	}
	return t
}

// Escape rejection: &s.items hands mutation ability to unknown code (and
// forces the copy into an Alloc, which Rule V refuses).
//
//bigo:max O(n) where n=len(s.items)
func (s S) EscapedField() int { // want `cannot verify budget O\(len\(s\.items\)\)`
	mutate(&s.items)
	t := 0
	for i := 0; i < len(s.items); i++ {
		t++
	}
	return t
}

// Numeric field bound (Rule P, clean).
//
//bigo:max O(n) where n=s.limit
func (s *S) CountToLimit() int {
	t := 0
	for i := 0; i < s.limit; i++ {
		t++
	}
	return t
}

// Depth-2 path through an embedded struct (Rule P, clean).
//
//bigo:max O(n) where n=len(s.cfg.items)
func (s *S) SumNested() int {
	t := 0
	for i := 0; i < len(s.cfg.items); i++ {
		t += s.cfg.items[i]
	}
	return t
}

func sumItems(s *S) int {
	t := 0
	for i := 0; i < len(s.items); i++ {
		t += s.items[i]
	}
	return t
}

// Call-site containment: sumItems is bounded in ITS frame, but its field var
// is meaningless here — the call must stay unverifiable (spec non-goal:
// re-rooting is deferred).
//
//bigo:max O(n)
func CallsFieldBounded(xs []int, s *S) int { // want `cannot verify budget O\(len\(xs\)\)`
	t := 0
	for i := 0; i < len(xs); i++ {
		t += sumItems(s)
	}
	return t
}

// The propagation gap is announced at the annotation, not discovered at
// distant call sites.
//
//bigo:cost O(k) where k=len(s.items)
func opaqueFieldCost(s *S) int // want `//bigo:cost with field-path sizes does not propagate`

// A field-rooted local slice (v1.28.0 review, F2). len(s.items[1:]) == len - 1
// <= len(s.items). Through v1.28.0 this was top: lenOf resolved the operand's
// length via Stab.VarFor, which names len/cap calls, never a collection. Now
// LenVarFor names the collection's length directly.
//
//bigo:max O(n) where n=len(s.items)
func (s *S) SumFieldTail() int {
	ys := s.items[1:]
	t := 0
	for i := 0; i < len(ys); i++ {
		t += ys[i]
	}
	return t
}

// The append copy idiom over a field-rooted spread: len(append(nil, s.items...))
// == len(s.items). Exercises LenVarFor through lenExtent's append branch.
//
//bigo:max O(n) where n=len(s.items)
func (s *S) SumFieldCopy() int {
	ys := append([]int(nil), s.items...)
	t := 0
	for i := 0; i < len(ys); i++ {
		t += ys[i]
	}
	return t
}
