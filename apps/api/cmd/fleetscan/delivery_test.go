package main

import (
	"context"
	"errors"
	"testing"
)

const coveringSweep = `{"repos":[{"name":"11","packages":5,"findings":[` + fleetFinding + `],"gaps":[]}],"gaps":[]}`

func completeFleet() []fakeRepo {
	repos := fleetRepos()
	return []fakeRepo{repos[0], repos[1], repos[2]}
}

func scanComplete(t *testing.T, failures, conflicts int32) (*fakeSonde, error) {
	t.Helper()
	github, sonde := fakeGithub(t, completeFleet()), newFakeSonde(t, failures)
	sonde.conflicts = conflicts
	err := scan(context.Background(), testConfig(github, sonde, fakeDouane(t, douaneEmitting(coveringSweep, 1))))
	return sonde, err
}

func TestACompleteRunSucceedsAndLeavesArchivedRepositoriesOut(t *testing.T) {
	quietLogs(t)
	sonde, err := scanComplete(t, 0, 0)
	if err != nil {
		t.Fatalf("scan: %v: a complete run must exit cleanly", err)
	}
	report := onlyReport(t, sonde)

	if report.Run.ReposExpected != 2 || report.Run.ReposScanned != 2 || len(report.Projects) != 2 {
		t.Fatalf("run = %+v with %d projects: an archived repository is not reported at all", report.Run, len(report.Projects))
	}
}

func TestADuplicateRefusalOnARetryCountsAsDelivered(t *testing.T) {
	quietLogs(t)
	sonde, err := scanComplete(t, 1, 1)
	if err != nil || sonde.calls.Load() != 2 {
		t.Fatalf("err = %v, calls = %d: a lost reply then a 409 means the report was stored", err, sonde.calls.Load())
	}
}

func TestADuplicateRefusalOnTheFirstAttemptIsAFailure(t *testing.T) {
	quietLogs(t)
	_, err := scanComplete(t, 0, 1)
	var refused deliveryError
	if !errors.As(err, &refused) || refused.status != 409 {
		t.Fatalf("err = %v: a first-attempt 409 is a stale report, not a delivery", err)
	}
}

func TestTheAcceptShrinkFlagReachesTheReport(t *testing.T) {
	quietLogs(t)
	github, sonde := fakeGithub(t, completeFleet()), newFakeSonde(t, 0)
	cfg := testConfig(github, sonde, fakeDouane(t, douaneEmitting(coveringSweep, 1)))
	cfg.acceptShrink = true
	if err := scan(context.Background(), cfg); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !onlyReport(t, sonde).Run.AcceptShrink {
		t.Fatal("-accept-shrink did not reach the report")
	}
}
