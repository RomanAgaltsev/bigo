package fixture

func g(int) int // bodyless -> unverifiable/nobody

func bounded1(xs []int) int {
	s := 0
	for i := 0; i < len(xs); i++ {
		s += xs[i]
	}
	return s
}

func bounded2() int { return 1 }

func topCall(xs []int) int {
	s := 0
	for i := 0; i < len(xs); i++ {
		s += g(xs[i]) // unresolved callee -> unverifiable/call
	}
	return s
}

func topLoop(n, j int) int {
	s := 0
	for i := 0; i+j < n; i++ { // variable offset -> unverifiable/loop
		s++
	}
	return s
}
