package env

import (
	"os"
	"regexp"
	"testing"
)

var envRead = regexp.MustCompile(`(?:Getenv|OrDefault)\("([A-Z][A-Z0-9_]+)"`)

func TestComposeForwardsEveryVariableTheServerReads(t *testing.T) {
	source, err := os.ReadFile("env.go")
	if err != nil {
		t.Fatalf("read env.go: %v", err)
	}
	compose, err := os.ReadFile("../../../../docker-compose.yml")
	if err != nil {
		t.Fatalf("read docker-compose.yml: %v", err)
	}
	reads := envRead.FindAllSubmatch(source, -1)
	if len(reads) < 10 {
		t.Fatalf("found only %d variables in env.go: the pattern no longer matches how it reads them", len(reads))
	}
	for _, match := range reads {
		name := string(match[1])
		forwarded := regexp.MustCompile(`(?m)^\s+` + name + `:`)
		if !forwarded.Match(compose) {
			t.Errorf("env.go reads %s but docker-compose.yml does not forward it, so production never sees it", name)
		}
	}
}
