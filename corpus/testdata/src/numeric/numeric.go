// Package numeric is the canonical-corpus number-algorithms family. Numeric
// parameters bind by value (where a=a), so bounds are in the argument's
// magnitude, matching the literature's convention.
package numeric

// GCD computes the greatest common divisor, iterative Euclid. The literature
// bound is O(log min(a,b)); O(log a) is the expressible worst case.
//
//oracle:time O(log a) where a=a
//oracle:space O(1) where a=a
//oracle:source CLRS §31.2 (Lamé); en.wikipedia.org/wiki/Euclidean_algorithm
func GCD(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// FastPow computes x^b (b ≥ 0) by iterative binary exponentiation.
//
//oracle:time O(log b) where b=b
//oracle:space O(1) where b=b
//oracle:source CLRS §31.6; ru.algorithmica.org (binary exponentiation)
func FastPow(x, b int) int {
	result := 1
	for b > 0 {
		if b&1 == 1 {
			result *= x
		}
		x *= x
		b >>= 1
	}
	return result
}

// FibIter returns the n-th Fibonacci number iteratively.
//
//oracle:time O(n) where n=n
//oracle:space O(1) where n=n
//oracle:source en.wikipedia.org/wiki/Fibonacci_sequence (iterative)
func FibIter(n int) int {
	a, b := 0, 1
	for i := 0; i < n; i++ {
		a, b = b, a+b
	}
	return a
}

// FibMemo returns the n-th Fibonacci number with map memoization — a
// deliberate known-⊤ shape (memoized recursion): the true bound is O(n),
// inference is expected to say ⊤. Evidence row for annotate-or-trust.
//
//oracle:time O(n) where n=n
//oracle:space O(n) where n=n
//oracle:source CLRS §15.1 (memoization); en.wikipedia.org/wiki/Memoization
func FibMemo(n int, memo map[int]int) int {
	if n < 2 {
		return n
	}
	if v, ok := memo[n]; ok {
		return v
	}
	v := FibMemo(n-1, memo) + FibMemo(n-2, memo)
	memo[n] = v
	return v
}

// Sieve returns the primality table up to n. The literature bound is
// O(n log log n); log log is inexpressible in the poly-log algebra, so the
// pin is the DELIBERATELY CONSERVATIVE O(n log n) (spec §5.1) — an `exact`
// against this pin would still be loose vs the true bound.
//
//oracle:time O(n log n) where n=n
//oracle:space O(n) where n=n
//oracle:source CLRS ex. 31-3; en.wikipedia.org/wiki/Sieve_of_Eratosthenes (conservative pin)
func Sieve(n int) []bool {
	composite := make([]bool, n+1)
	for p := 2; p*p <= n; p++ {
		if composite[p] {
			continue
		}
		for q := p * p; q <= n; q += p {
			composite[q] = true
		}
	}
	return composite
}

// TrialDivision reports whether n ≥ 2 is prime by trial division. The
// literature bound is O(√n); √ is inexpressible in the poly-log algebra, so
// the pin is the DELIBERATELY CONSERVATIVE O(n) (spec §5.1).
//
//oracle:time O(n) where n=n
//oracle:space O(1) where n=n
//oracle:source en.wikipedia.org/wiki/Trial_division (conservative pin)
func TrialDivision(n int) bool {
	if n < 2 {
		return false
	}
	for d := 2; d*d <= n; d++ {
		if n%d == 0 {
			return false
		}
	}
	return true
}

// DigitSum sums the base-10 digits of n ≥ 0.
//
//oracle:time O(log n) where n=n
//oracle:space O(1) where n=n
//oracle:source www.geeksforgeeks.org/program-for-sum-of-the-digits-of-a-given-number/ (bound reference)
func DigitSum(n int) int {
	sum := 0
	for n > 0 {
		sum += n % 10
		n /= 10
	}
	return sum
}
