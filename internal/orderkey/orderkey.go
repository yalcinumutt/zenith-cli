// Package orderkey generates fractional-index keys for manual ordering
// (ADR 0007). A key is a short base-62 string; rows sort by key (ties broken by
// uuid). Placing a row between two neighbours needs one new key and a single-row
// write, so reorders never touch other rows and merge safely under per-field
// last-write-wins.
//
// Keys never end in the zero digit, which keeps every key strictly between its
// neighbours' fractions.
package orderkey

import (
	"errors"
	"fmt"
	"strings"
)

const digits = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

const base = len(digits)

// ErrOrder is returned when the bounds are not in order.
var ErrOrder = errors.New("orderkey: lower bound is not before upper bound")

func idx(c byte) int { return strings.IndexByte(digits, c) }

func valid(k string) bool {
	if k == "" {
		return true
	}
	if k[len(k)-1] == digits[0] {
		return false
	}
	for i := 0; i < len(k); i++ {
		if idx(k[i]) < 0 {
			return false
		}
	}
	return true
}

// Between returns a key strictly between a and b. An empty a means "before
// everything" and an empty b means "after everything".
func Between(a, b string) (string, error) {
	if !valid(a) || !valid(b) {
		return "", fmt.Errorf("orderkey: invalid key in (%q, %q)", a, b)
	}
	if a != "" && b != "" && a >= b {
		return "", ErrOrder
	}
	switch {
	case a == "" && b == "":
		return "V", nil
	case b == "":
		return after(a), nil
	case a == "":
		return before(b), nil
	}
	return midpoint(a, b), nil
}

// after returns a short key greater than a: bump the last digit, or extend by
// one digit once it is maxed out. Runs of appends therefore grow slowly.
func after(a string) string {
	last := a[len(a)-1]
	if i := idx(last); i < base-1 {
		return a[:len(a)-1] + string(digits[i+1])
	}
	return a + "V"
}

// before returns a short key smaller than b, symmetric to after.
func before(b string) string {
	last := b[len(b)-1]
	if i := idx(last); i > 1 {
		return b[:len(b)-1] + string(digits[i-1])
	}
	return b[:len(b)-1] + "0V" // b ended in 1: ...0V sorts just before ...1
}

func at(s string, i int) byte {
	if i < len(s) {
		return s[i]
	}
	return digits[0]
}

func from(s string, n int) string {
	if n >= len(s) {
		return ""
	}
	return s[n:]
}

func midpoint(a, b string) string {
	if b != "" {
		// Strip the common prefix, treating a as zero-padded.
		n := 0
		for n < len(b) && at(a, n) == b[n] {
			n++
		}
		if n > 0 {
			return b[:n] + midpoint(from(a, n), b[n:])
		}
	}
	da := 0
	if a != "" {
		da = idx(a[0])
	}
	db := base
	if b != "" {
		db = idx(b[0])
	}
	if db-da > 1 {
		return string(digits[(da+db+1)/2])
	}
	// The first digits are consecutive.
	if len(b) > 1 {
		return b[:1]
	}
	return string(digits[da]) + midpoint(from(a, 1), "")
}

// Initial returns n increasing keys with room between each, for numbering rows
// that have no key yet in their existing order.
func Initial(n int) []string {
	width := 1
	for pow := base; pow <= n; pow *= base {
		width++
	}
	keys := make([]string, n)
	for i := range keys {
		v := i + 1
		buf := make([]byte, width)
		for p := width - 1; p >= 0; p-- {
			buf[p] = digits[v%base]
			v /= base
		}
		// A trailing mid-digit keeps room after every key and avoids a trailing zero.
		keys[i] = string(buf) + "V"
	}
	return keys
}
