package shadow

import (
	"errors"
	"sync"
	"testing"

	"github.com/dyl-03/shadow/internal/clock"
	"github.com/dyl-03/shadow/internal/metrics"
	"github.com/dyl-03/shadow/internal/wal"
)

// tempWAL returns a WAL backed by the test's temp directory.
func tempWAL(t *testing.T) *wal.Manager {
	t.Helper()
	dir := t.TempDir()
	m, err := wal.NewManager(dir, 1<<20, metrics.New())
	if err != nil {
		t.Fatalf("new wal: %v", err)
	}
	t.Cleanup(func() { _ = m.Close() })
	return m
}

// TestUpdateDesired_VersionCheck_RejectsStaleWriter reproduces the concurrent
// clobber bug: two callers both read the same base version and race to write.
// The second (stale) write must be rejected with ErrVersionConflict, leaving
// the winner's state intact and the version advanced exactly once.
func TestUpdateDesired_VersionCheck_RejectsStaleWriter(t *testing.T) {
	store := New(tempWAL(t), clock.Wall(), metrics.New())

	// First write seeds the device at desired version 1.
	sh1, err := store.UpdateDesired("dev1", 0, map[string]string{"mode": "auto"})
	if err != nil {
		t.Fatalf("seed write: %v", err)
	}
	if sh1.DesiredVer != 1 {
		t.Fatalf("seed version = %d, want 1", sh1.DesiredVer)
	}

	// A second caller that still holds the stale base version (0) attempts a
	// concurrent write. It must be rejected, not silently applied.
	_, err = store.UpdateDesired("dev1", 0, map[string]string{"mode": "manual"})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale write err = %v, want ErrVersionConflict", err)
	}

	// The winner's value and version must be unchanged by the rejected write.
	got := store.Get("dev1")
	if got.Desired["mode"] != "auto" {
		t.Fatalf("desired[mode] = %q, want %q (stale write clobbered state)", got.Desired["mode"], "auto")
	}
	if got.DesiredVer != 1 {
		t.Fatalf("version = %d, want 1 (version did not stay put)", got.DesiredVer)
	}

	// A write carrying the correct in-sync version succeeds and advances again.
	sh3, err := store.UpdateDesired("dev1", 1, map[string]string{"mode": "manual"})
	if err != nil {
		t.Fatalf("in-sync write: %v", err)
	}
	if sh3.DesiredVer != 2 {
		t.Fatalf("post-update version = %d, want 2", sh3.DesiredVer)
	}
	if sh3.Desired["mode"] != "manual" {
		t.Fatalf("post-update desired[mode] = %q, want %q", sh3.Desired["mode"], "manual")
	}
}

// TestUpdateDesired_ConcurrentNoClobber runs many concurrent writers that all
// think they are based on version 0. Exactly one must win; every other writer
// must observe ErrVersionConflict, and the final state must reflect only the
// single winning write with version 1.
func TestUpdateDesired_ConcurrentNoClobber(t *testing.T) {
	store := New(tempWAL(t), clock.Wall(), metrics.New())

	const n = 32
	var wg sync.WaitGroup
	errs := make([]error, n)
	start := make(chan struct{})

	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			<-start
			_, err := store.UpdateDesired("devN", 0, map[string]string{"writer": string(rune('a' + i%26))})
			errs[i] = err
		}()
	}
	close(start)
	wg.Wait()

	conflicts := 0
	for _, err := range errs {
		if errors.Is(err, ErrVersionConflict) {
			conflicts++
		} else if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if conflicts != n-1 {
		t.Fatalf("conflicts = %d, want %d (exactly one writer should win)", conflicts, n-1)
	}

	got := store.Get("devN")
	if got.DesiredVer != 1 {
		t.Fatalf("final version = %d, want 1", got.DesiredVer)
	}
	if _, ok := got.Desired["writer"]; !ok {
		t.Fatal("final desired has no writer field (winner not recorded)")
	}
}
