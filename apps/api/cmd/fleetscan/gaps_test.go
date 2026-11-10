package main

import "testing"

func TestOnlyGapsThatCanHideAFindingWithholdCoverage(t *testing.T) {
	cases := []struct {
		name     string
		gap      douaneGap
		packages int
		blocks   bool
	}{
		{"advisory fetch failed", douaneGap{Kind: "unresolved", Subject: "GHSA-aaaa-bbbb-cccc"}, 40, true},
		{"dockerfile module unnamed", douaneGap{Kind: "unresolved", Subject: "Dockerfile"}, 40, false},
		{"nested dockerfile module unnamed", douaneGap{Kind: "unresolved", Subject: "apps/api/Dockerfile.prod"}, 40, false},
		{"unsupported beside read packages", douaneGap{Kind: "unsupported", Subject: "pnpm-lock.yaml"}, 40, false},
		{"unsupported and nothing read", douaneGap{Kind: "unsupported", Subject: "yarn.lock"}, 0, true},
		{"unreadable lockfile", douaneGap{Kind: "unreadable", Subject: "go.mod"}, 40, true},
		{"kev feed down", douaneGap{Kind: "upstream", Subject: "kev"}, 40, false},
		{"osv down", douaneGap{Kind: "upstream", Subject: "osv"}, 40, true},
		{"severity only", douaneGap{Kind: "severity", Subject: "GO-2026-1"}, 40, false},
	}
	for _, c := range cases {
		got := len(blocking([]douaneGap{c.gap}, c.packages)) > 0
		if got != c.blocks {
			t.Errorf("%s: blocks = %v, want %v", c.name, got, c.blocks)
		}
	}
}
