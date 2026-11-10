package fleet

import "testing"

func TestTheKeyPrefersACVEOverAMirrorAliasThatSortsFirst(t *testing.T) {
	f := FindingReport{Ecosystem: "Go", Package: "stdlib", Installed: "1.22", ID: "GO-2026-1", Aliases: []string{"BIT-golang-2026-1", "GHSA-aaaa", "CVE-2026-1"}}

	if got := f.Key(); got != "Go|stdlib|1.22|CVE-2026-1" {
		t.Fatalf("key = %q: a BIT- alias must never become the key", got)
	}
	if got := canonicalName("GO-2026-1", "BIT-golang-2026-1,GHSA-aaaa"); got != "GHSA-aaaa" {
		t.Fatalf("canonical = %q: without a CVE the GitHub advisory wins", got)
	}
	if got := canonicalName("GO-2026-1", "BIT-golang-2026-1"); got != "GO-2026-1" {
		t.Fatalf("canonical = %q: the ecosystem's own id outranks a mirror", got)
	}
}
