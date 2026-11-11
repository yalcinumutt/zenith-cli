package orderkey

import (
	"math/rand"
	"sort"
	"testing"
)

func mustBetween(t *testing.T, a, b string) string {
	t.Helper()
	k, err := Between(a, b)
	if err != nil {
		t.Fatalf("Between(%q,%q): %v", a, b, err)
	}
	if a != "" && k <= a {
		t.Fatalf("Between(%q,%q) = %q, not after a", a, b, k)
	}
	if b != "" && k >= b {
		t.Fatalf("Between(%q,%q) = %q, not before b", a, b, k)
	}
	if k[len(k)-1] == '0' {
		t.Fatalf("key %q ends in zero", k)
	}
	return k
}

func TestEdgesAndOrdering(t *testing.T) {
	first := mustBetween(t, "", "")
	if mustBetween(t, first, "") <= first || mustBetween(t, "", first) >= first {
		t.Error("edges")
	}
	if _, err := Between("b", "a"); err != ErrOrder {
		t.Errorf("reversed bounds: %v", err)
	}
	if _, err := Between("a", "a"); err != ErrOrder {
		t.Errorf("equal bounds: %v", err)
	}
	if _, err := Between("a0", ""); err == nil {
		t.Error("trailing zero accepted")
	}
}

func TestRepeatedInsertionAtTheSamePosition(t *testing.T) {
	lo, hi := mustBetween(t, "", ""), ""
	hi = mustBetween(t, lo, "")
	// Always insert right after lo: the interval keeps halving.
	keys := []string{lo, hi}
	cur := hi
	for i := 0; i < 500; i++ {
		k := mustBetween(t, lo, cur)
		keys = append(keys, k)
		cur = k
	}
	sorted := append([]string(nil), keys...)
	sort.Strings(sorted)
	seen := map[string]bool{}
	for _, k := range sorted {
		if seen[k] {
			t.Fatalf("duplicate key %q", k)
		}
		seen[k] = true
	}
	// Always insert right before hi.
	cur = lo
	for i := 0; i < 500; i++ {
		cur = mustBetween(t, cur, hi)
	}
}

func TestAppendAndPrependScale(t *testing.T) {
	k := ""
	for i := 0; i < 1000; i++ {
		next := mustBetween(t, k, "")
		k = next
	}
	if len(k) > 40 {
		t.Errorf("1000 appends produced a %d-char key", len(k))
	}
	k = "V"
	for i := 0; i < 300; i++ {
		k = mustBetween(t, "", k)
	}
	if len(k) > 40 {
		t.Errorf("300 prepends produced a %d-char key", len(k))
	}
}

func TestRandomInsertionsStaySorted(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	list := []string{}
	for i := 0; i < 2000; i++ {
		pos := rng.Intn(len(list) + 1)
		lo, hi := "", ""
		if pos > 0 {
			lo = list[pos-1]
		}
		if pos < len(list) {
			hi = list[pos]
		}
		k := mustBetween(t, lo, hi)
		list = append(list, "")
		copy(list[pos+1:], list[pos:])
		list[pos] = k
	}
	if !sort.StringsAreSorted(list) {
		t.Fatal("list not sorted after random insertions")
	}
}

func TestInitialIsIncreasingAndLeavesRoom(t *testing.T) {
	for _, n := range []int{0, 1, 5, 61, 62, 63, 5000} {
		keys := Initial(n)
		if len(keys) != n || !sort.StringsAreSorted(keys) {
			t.Fatalf("n=%d: not sorted", n)
		}
		for i := 1; i < len(keys); i++ {
			if keys[i] == keys[i-1] {
				t.Fatalf("n=%d: duplicate", n)
			}
			mustBetween(t, keys[i-1], keys[i])
		}
		if n > 0 {
			mustBetween(t, "", keys[0])
			mustBetween(t, keys[n-1], "")
		}
	}
}
