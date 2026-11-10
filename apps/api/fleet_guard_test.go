package main

import (
	"testing"

	"github.com/FacileStudio/Sonde/apps/api/internal/fleet"
)

func fleetOf(n int) []fleet.ProjectReport {
	projects := make([]fleet.ProjectReport, 0, n)
	for i := range n {
		projects = append(projects, covered(int64(100+i), "repo-"+itoa(int64(i)), flaw("CVE-2026-"+itoa(int64(i)))))
	}
	return projects
}

func TestAMuchShorterCompleteRunResolvesNothingAndIsStoredIncomplete(t *testing.T) {
	router, db := fleetRouter(t)
	ingest(t, router, nightly(fleetOf(88)...))

	narrowed := ingest(t, router, nightly(fleetOf(1)...))

	var stored bool
	db.Table("fleet_runs").Select("complete").Order("id DESC").Limit(1).Scan(&stored)
	if narrowed.Resolved != 0 || narrowed.FindingsOpen != 88 || narrowed.Complete || stored || !narrowed.CoverageDropped {
		t.Fatalf("result = %+v, stored complete = %v: a narrowed listing must not resolve the fleet", narrowed, stored)
	}
}

func TestAConfirmedShrinkResolvesTheProjectsThatLeft(t *testing.T) {
	router, _ := fleetRouter(t)
	ingest(t, router, nightly(fleetOf(88)...))
	report := nightly(fleetOf(1)...)
	report.Run.AcceptShrink = true

	confirmed := ingest(t, router, report)

	if confirmed.Resolved != 87 || !confirmed.Complete || confirmed.CoverageDropped {
		t.Fatalf("result = %+v: accept_shrink lets a real departure resolve", confirmed)
	}
}

func TestAStoredKeyIsReusedWhenABetterNameArrives(t *testing.T) {
	router, _ := fleetRouter(t)
	ingest(t, router, nightly(covered(1, "a", flaw("BIT-golang-2026-1"))))

	later := ingest(t, router, nightly(covered(1, "a", flaw("CVE-2026-1", "BIT-golang-2026-1"))))

	if later.NewFindings != 0 || later.Resolved != 0 || later.FindingsOpen != 1 {
		t.Fatalf("result = %+v: a finding filed under its BIT- name keeps that key", later)
	}
}
