//go:build linux

package huge_test

import (
	"testing"

	"github.com/bboreham/go-huge"
)

func TestMarkAll(t *testing.T) {
	count, err := huge.MarkAll()
	if err != nil {
		t.Logf("first non-fatal error: %v", err)
	}
	if count == 0 {
		t.Fatal("expected at least one region to be advised")
	}
	t.Logf("advised %d regions", count)
}
