package monitors

import (
	"strings"
	"testing"
)

func validHTTPInput() Input {
	return Input{
		Slug:            "api-gateway",
		Name:            "API Gateway",
		Type:            TypeHTTP,
		Target:          "https://example.com/health",
		IntervalSeconds: DefaultIntervalSeconds,
		TimeoutSeconds:  DefaultTimeoutSeconds,
		ExpectedStatus:  DefaultExpectedStatus,
	}
}

func TestValidateAcceptsWellFormedMonitors(t *testing.T) {
	cases := map[string]func(*Input){
		"http":    func(in *Input) {},
		"tcp":     func(in *Input) { in.Type = TypeTCP; in.Target = "example.com:443"; in.ExpectedStatus = 0 },
		"push":    func(in *Input) { in.Type = TypePush; in.Target = ""; in.ExpectedStatus = 0 },
		"https":   func(in *Input) { in.Target = "http://example.com" },
		"keyword": func(in *Input) { in.ExpectedKeyword = "OK" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			in := validHTTPInput()
			mutate(&in)
			if err := Validate(&in); err != nil {
				t.Fatalf("Validate() = %v, want nil", err)
			}
		})
	}
}

func TestValidateRejectsBadMonitors(t *testing.T) {
	cases := map[string]func(*Input){
		"bad slug":         func(in *Input) { in.Slug = "Not A Slug" },
		"empty slug":       func(in *Input) { in.Slug = "" },
		"empty name":       func(in *Input) { in.Name = "" },
		"interval too low": func(in *Input) { in.IntervalSeconds = MinIntervalSeconds - 1 },
		"interval too high": func(in *Input) {
			in.IntervalSeconds = MaxIntervalSeconds + 1
		},
		"timeout above interval": func(in *Input) { in.TimeoutSeconds = in.IntervalSeconds + 1 },
		"unknown type":           func(in *Input) { in.Type = "icmp" },
		"http target not a url":  func(in *Input) { in.Target = "example.com/health" },
		"http target ftp":        func(in *Input) { in.Target = "ftp://example.com" },
		"status out of range":    func(in *Input) { in.ExpectedStatus = 999 },
		"tcp without port":       func(in *Input) { in.Type = TypeTCP; in.Target = "example.com" },
		"tcp port out of range":  func(in *Input) { in.Type = TypeTCP; in.Target = "example.com:99999" },
		"push with target":       func(in *Input) { in.Type = TypePush; in.Target = "https://elsewhere" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			in := validHTTPInput()
			mutate(&in)
			if err := Validate(&in); err == nil {
				t.Fatal("Validate() = nil, want an error")
			}
		})
	}
}

func TestValidateAppliesDefaults(t *testing.T) {
	in := Input{Slug: "x", Name: "X", Type: TypeTCP, Target: "h:1"}
	if err := Validate(&in); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
	if in.IntervalSeconds != DefaultIntervalSeconds || in.TimeoutSeconds != DefaultTimeoutSeconds {
		t.Fatalf("defaults not applied: interval=%d timeout=%d", in.IntervalSeconds, in.TimeoutSeconds)
	}
}

func TestSlugPatternBoundaries(t *testing.T) {
	tooLong := strings.Repeat("a", 101)
	if err := Validate(&Input{Slug: tooLong, Name: "X", Type: TypePush}); err == nil {
		t.Fatal("over-long slug accepted")
	}
}
