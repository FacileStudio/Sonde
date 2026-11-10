package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/fleet"
)

func TestAFindingKeepsItsKeyWhenItsCVEIsAssignedLater(t *testing.T) {
	router, db := fleetRouter(t)
	ingest(t, router, nightly(covered(1, "a", flaw("GHSA-zzzz-zzzz-zzzz"))))
	var before time.Time
	db.Table("fleet_findings").Select("first_seen").Scan(&before)

	later := ingest(t, router, nightly(covered(1, "a", flaw("GHSA-zzzz-zzzz-zzzz", "CVE-2026-9"))))

	var after time.Time
	db.Table("fleet_findings").Select("first_seen").Scan(&after)
	if later.NewFindings != 0 || later.Resolved != 0 || later.FindingsOpen != 1 || !after.Equal(before) {
		t.Fatalf("result = %+v, first_seen %v -> %v: a new alias must not reopen the flaw", later, before, after)
	}
}

func TestACompleteRunResolvesAProjectItNoLongerLists(t *testing.T) {
	router, db := fleetRouter(t)
	ingest(t, router, nightly(append(fleetOf(9), covered(2, "gone", flaw("CVE-2026-1")))...))

	result := ingest(t, router, nightly(fleetOf(9)...))

	var covered bool
	var reason string
	db.Table("fleet_projects").Select("covered").Where("github_repo_id = 2").Scan(&covered)
	db.Table("fleet_projects").Select("not_covered").Where("github_repo_id = 2").Scan(&reason)
	if result.Resolved != 1 || result.FindingsOpen != 9 || covered || reason != "not in the latest scan" {
		t.Fatalf("result = %+v, covered = %v, reason = %q", result, covered, reason)
	}
}

func TestAnIncompleteRunLeavesUnlistedProjectsAlone(t *testing.T) {
	router, _ := fleetRouter(t)
	ingest(t, router, nightly(covered(1, "a"), covered(2, "b", flaw("CVE-2026-1"))))
	partial := nightly(covered(1, "a"))
	partial.Run.ReposExpected = 2

	if got := ingest(t, router, partial); got.Resolved != 0 || got.FindingsOpen != 1 || got.Complete {
		t.Fatalf("result = %+v: an incomplete run must not resolve what it did not list", got)
	}
}

func TestAReportNoNewerThanTheLastIsRefused(t *testing.T) {
	router, _ := fleetRouter(t)
	older := nightly(covered(1, "a"))
	newer := nightly(covered(1, "a"))
	newer.Run.FinishedAt = older.Run.FinishedAt.Add(time.Minute)
	ingest(t, router, newer)

	if got := deliver(t, router, fleetToken, gzipJSON(t, older)).Code; got != http.StatusConflict {
		t.Errorf("older report = %d, want 409", got)
	}
	if got := deliver(t, router, fleetToken, gzipJSON(t, newer)).Code; got != http.StatusConflict {
		t.Errorf("the same report again = %d, want 409", got)
	}
}

func TestARunThatExpectedNothingIsNotComplete(t *testing.T) {
	router, _ := fleetRouter(t)
	empty := nightly()

	if got := ingest(t, router, empty); got.Complete {
		t.Fatalf("result = %+v: a run that expected nothing proves nothing", got)
	}
	if (fleet.RunInfo{ReposExpected: 3, ReposScanned: 3}).Complete() != true {
		t.Fatal("a run that scanned all it expected is complete")
	}
}
