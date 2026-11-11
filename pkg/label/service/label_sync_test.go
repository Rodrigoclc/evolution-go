package label_service

import (
	"errors"
	"testing"
)

func TestRunWithFullSyncEventsEnablesOnlyDuringRun(t *testing.T) {
	flag := false
	var seen bool

	err := runWithFullSyncEvents(&flag, func() error {
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

		err := runWithFullSyncEvents(&flag, func() error { return want })

		if !errors.Is(err, want) {
			t.Fatalf("expected the sync error to be returned, got %v", err)
		}
		if flag != previous {
			t.Fatalf("flag must be restored to %v, got %v", previous, flag)
		}
	}
}
