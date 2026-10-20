// Package testutil holds small helpers shared by tests.
package testutil

import "testing"

// Check fails the test when err is non-nil.
func Check(tb testing.TB, err error) {
	tb.Helper()
	if err != nil {
		tb.Fatal(err)
	}
}

// CheckCleanup reports err without stopping the rest of a cleanup.
func CheckCleanup(tb testing.TB, err error) {
	tb.Helper()
	if err != nil {
		tb.Error(err)
	}
}
