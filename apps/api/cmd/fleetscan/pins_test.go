package main

import "testing"

func TestPackageJSONPinsReadTheRefAfterTheHash(t *testing.T) {
	data := []byte(`{"dependencies":{"@facile/muse":"github:FacileStudio/muse#v0.10.0","svelte":"^5"},"devDependencies":{"vite":"^7"}}`)

	pins := packageJSONPins("apps/client/package.json", data)

	if len(pins) != 1 || pins[0].Package != "@facile/muse" || pins[0].Resolved != "v0.10.0" || pins[0].Path != "apps/client/package.json" {
		t.Fatalf("pins = %+v", pins)
	}
}

func TestAnUnpinnedGitHubDependencyResolvesToNothing(t *testing.T) {
	pins := packageJSONPins("package.json", []byte(`{"dependencies":{"@facile/muse":"github:FacileStudio/muse"}}`))

	if len(pins) != 1 || pins[0].Resolved != "" {
		t.Fatalf("pins = %+v, want one unpinned", pins)
	}
}

func TestPackageJSONThatIsNotJSONYieldsNoPins(t *testing.T) {
	if pins := packageJSONPins("package.json", []byte("{not json")); len(pins) != 0 {
		t.Fatalf("pins = %+v", pins)
	}
}

func TestGoModPinsTakeBlockAndSingleLineRequiresButNotNestedModules(t *testing.T) {
	data := []byte("module x\n\ngo 1.26.4\n\nrequire (\n\tgithub.com/FacileStudio/tronc v0.17.1\n\tgithub.com/FacileStudio/tronc/migrate v0.1.0\n\tgithub.com/FacileStudio/porte v0.13.0 // indirect\n\tgithub.com/other/lib v1.0.0\n)\n\nrequire github.com/FacileStudio/filet v0.18.2\n")

	pins, directive := goModPins("apps/api/go.mod", data)

	got := map[string]string{}
	for _, pin := range pins {
		got[pin.Package] = pin.Resolved
	}
	want := map[string]string{
		"github.com/FacileStudio/tronc": "v0.17.1",
		"github.com/FacileStudio/porte": "v0.13.0",
		"github.com/FacileStudio/filet": "v0.18.2",
	}
	if len(got) != len(want) {
		t.Fatalf("pins = %v, want %v", got, want)
	}
	for pkg, version := range want {
		if got[pkg] != version {
			t.Errorf("%s = %q, want %q", pkg, got[pkg], version)
		}
	}
	if directive != "1.26.4" {
		t.Errorf("directive = %q", directive)
	}
}

func TestHighestTagIgnoresPrereleasesAndNonSemver(t *testing.T) {
	got := highestTag([]string{"v0.9.0", "v0.16.1", "v0.16.2-rc1", "latest", "v0.16.0", "v0.100.0"})

	if got != "v0.100.0" {
		t.Fatalf("highest = %q, want v0.100.0 (numeric, not lexical)", got)
	}
	if highestTag([]string{"nightly", "v1-beta"}) != "" {
		t.Fatal("a list with no release tag must yield nothing")
	}
}

func TestFiletStateFindsTheWiringAndThePin(t *testing.T) {
	wired, pin := filetState([][]byte{[]byte("jobs:\n  go:\n    steps:\n      - run: go test\n"), []byte("      - uses: FacileStudio/filet@v0.3.0\n")})
	if !wired || pin != "v0.3.0" {
		t.Fatalf("wired = %v, pin = %q", wired, pin)
	}
	if wired, pin := filetState([][]byte{[]byte("run: go test")}); wired || pin != "" {
		t.Fatalf("no filet anywhere reported wired = %v, pin = %q", wired, pin)
	}
	if wired, _ := filetState([][]byte{[]byte("# TODO: add filet here\nsteps:\n  - run: go test\n")}); wired {
		t.Fatal("a comment mentioning filet counted as wiring it")
	}
	if wired, pin := filetState([][]byte{[]byte("steps:\n  - run: filet check .\n")}); !wired || pin != "" {
		t.Fatalf("a filet check step: wired = %v, pin = %q", wired, pin)
	}
}

func TestGoModPinsIgnoreReplaceDirectivesAndAcceptMajorVersions(t *testing.T) {
	data := []byte("module x\n\ngo 1.26\n\nrequire github.com/FacileStudio/tronc/v2 v2.1.0\n\nrequire github.com/FacileStudio/porte v0.13.0\n\nreplace (\n\tgithub.com/FacileStudio/porte v0.12.0 => ../porte\n)\n\nreplace github.com/FacileStudio/filet v0.3.0 => github.com/fork/filet v0.3.1\n")

	pins, _ := goModPins("go.mod", data)

	got := map[string]string{}
	for _, pin := range pins {
		got[pin.Package] = pin.Resolved
	}
	if len(pins) != 2 || got["github.com/FacileStudio/tronc"] != "v2.1.0" || got["github.com/FacileStudio/porte"] != "v0.13.0" {
		t.Fatalf("pins = %v: the /v2 module counts, replace lines never do", got)
	}
}

func TestPackageJSONPinsStripSemverAndReadPeerDependencies(t *testing.T) {
	pins := packageJSONPins("package.json", []byte(`{"peerDependencies":{"@facile/muse":"github:FacileStudio/muse#semver:^0.16"}}`))

	if len(pins) != 1 || pins[0].Resolved != "^0.16" {
		t.Fatalf("pins = %+v", pins)
	}
}
