package fleet

import (
	"testing"
	"time"
)

func TestDriftSeparatesCurrentBehindAndHeld(t *testing.T) {
	now := time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)
	pins := []Pin{
		{ProjectID: 1, ProjectName: "a", Package: "muse", Resolved: "v0.16.1"},
		{ProjectID: 2, ProjectName: "b", Package: "muse", Resolved: "v0.10.0"},
		{ProjectID: 3, ProjectName: "c", Package: "muse", Resolved: "v0.10.0"},
		{ProjectID: 4, ProjectName: "d", Package: "muse", Resolved: ""},
		{ProjectID: 5, ProjectName: "e", Package: "untracked", Resolved: "v1"},
	}
	holdB := int64(2)
	policies := []Policy{{Package: "muse", ProjectID: &holdB, HeldVersion: "v0.10.0", Reason: "needs the old tokens", ExpiresAt: now.Add(time.Hour)}}

	rows := BuildDrift(pins, map[string]string{"muse": "v0.16.1"}, policies, now)

	if len(rows) != 1 || rows[0].Current != 1 {
		t.Fatalf("rows = %+v", rows)
	}
	behind := rows[0].Behind
	if len(behind) != 2 || behind[0].Resolved != "v0.10.0" || len(behind[0].Projects) != 2 {
		t.Fatalf("behind = %+v", behind)
	}
	if !behind[0].Projects[0].Held || behind[0].Projects[1].Held {
		t.Fatalf("only project b is held: %+v", behind[0].Projects)
	}
	if behind[1].Resolved != unpinned {
		t.Fatalf("an empty pin must read as %q, got %q", unpinned, behind[1].Resolved)
	}
}

func TestAnExpiredHoldNoLongerHides(t *testing.T) {
	now := time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)
	pins := []Pin{{ProjectID: 2, ProjectName: "b", Package: "muse", Resolved: "v0.10.0"}}
	policies := []Policy{{Package: "muse", HeldVersion: "v0.10.0", Reason: "x", ExpiresAt: now.Add(-time.Second)}}

	rows := BuildDrift(pins, map[string]string{"muse": "v0.16.1"}, policies, now)

	if rows[0].Behind[0].Projects[0].Held {
		t.Fatal("a lapsed hold still hides the project")
	}
}

func TestAHoldOnAnotherVersionDoesNotApply(t *testing.T) {
	now := time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)
	pins := []Pin{{ProjectID: 2, ProjectName: "b", Package: "muse", Resolved: "v0.9.0"}}
	policies := []Policy{{Package: "muse", HeldVersion: "v0.10.0", Reason: "x", ExpiresAt: now.Add(time.Hour)}}

	rows := BuildDrift(pins, map[string]string{"muse": "v0.16.1"}, policies, now)

	if rows[0].Behind[0].Projects[0].Held {
		t.Fatal("a hold on v0.10.0 hid a project on v0.9.0")
	}
}

func TestAProjectPinningOneVersionFromSeveralManifestsCountsOnce(t *testing.T) {
	now := time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)
	pins := []Pin{
		{ProjectID: 1, ProjectName: "fusible", Package: "muse", Resolved: "v0.16.1"},
		{ProjectID: 1, ProjectName: "fusible", Package: "muse", Resolved: "v0.16.1"},
		{ProjectID: 2, ProjectName: "sonde", Package: "muse", Resolved: "v0.17.0"},
		{ProjectID: 2, ProjectName: "sonde", Package: "muse", Resolved: "v0.17.0"},
	}

	rows := BuildDrift(pins, map[string]string{"muse": "v0.17.0"}, nil, now)

	if rows[0].Current != 1 || len(rows[0].Behind) != 1 || len(rows[0].Behind[0].Projects) != 1 {
		t.Fatalf("drift = %+v: each project must appear once per version", rows[0])
	}
}
