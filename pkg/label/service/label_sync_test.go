package label_service

import (
	"errors"
	"testing"
	"time"
)

func TestRunWithFullSyncEventsEnablesOnlyDuringRun(t *testing.T) {
	flag := false
	var seen bool

	err := runWithFullSyncEvents("instance-a", &flag, func() error {
		seen = flag
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !seen {
		t.Fatal("events must be enabled while the full sync runs")
	}
	if flag {
		t.Fatal("events must be disabled again after the full sync")
	}
}

func TestRunWithFullSyncEventsRestoresPreviousValueOnError(t *testing.T) {
	for _, previous := range []bool{false, true} {
		flag := previous
		want := errors.New("sync failed")

		err := runWithFullSyncEvents("instance-a", &flag, func() error { return want })

		if !errors.Is(err, want) {
			t.Fatalf("expected the sync error to be returned, got %v", err)
		}
		if flag != previous {
			t.Fatalf("flag must be restored to %v, got %v", previous, flag)
		}
	}
}

func TestRunWithFullSyncEventsLocksPerInstance(t *testing.T) {
	var flagA, flagB, flagA2 bool
	release := make(chan struct{})
	started := make(chan struct{})
	doneA := make(chan struct{})

	go func() {
		defer close(doneA)
		_ = runWithFullSyncEvents("instance-a", &flagA, func() error {
			close(started)
			<-release
			return nil
		})
	}()
	<-started

	// Another instance is not blocked by the running resync.
	otherDone := make(chan struct{})
	go func() {
		_ = runWithFullSyncEvents("instance-b", &flagB, func() error { return nil })
		close(otherDone)
	}()
	select {
	case <-otherDone:
	case <-time.After(2 * time.Second):
		t.Fatal("a resync of another instance must not wait for instance-a")
	}

	// The same instance waits for the running resync.
	sameDone := make(chan struct{})
	go func() {
		_ = runWithFullSyncEvents("instance-a", &flagA2, func() error { return nil })
		close(sameDone)
	}()
	select {
	case <-sameDone:
		t.Fatal("a second resync of instance-a must wait for the first one")
	case <-time.After(100 * time.Millisecond):
	}

	close(release)
	<-doneA
	select {
	case <-sameDone:
	case <-time.After(2 * time.Second):
		t.Fatal("the second resync of instance-a must run after the first one")
	}
}
