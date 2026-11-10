package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func countingServer(t *testing.T, handler func(call int32, w http.ResponseWriter, r *http.Request)) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler(calls.Add(1), w, r)
	}))
	t.Cleanup(server.Close)
	return server, &calls
}

func TestAThrottledRequestIsRetriedAndAPlainRefusalIsNot(t *testing.T) {
	cases := []struct {
		name    string
		headers map[string]string
		status  int
		calls   int32
	}{
		{"429 with Retry-After", map[string]string{"Retry-After": "0"}, http.StatusTooManyRequests, 2},
		{"secondary limit 403", map[string]string{"Retry-After": "0"}, http.StatusForbidden, 2},
		{"exhausted primary limit", map[string]string{"X-RateLimit-Remaining": "0", "X-RateLimit-Reset": strconv.FormatInt(time.Now().Unix()-1, 10)}, http.StatusForbidden, 2},
		{"plain 403", nil, http.StatusForbidden, 1},
		{"404", nil, http.StatusNotFound, 1},
	}
	for _, c := range cases {
		server, calls := countingServer(t, func(call int32, w http.ResponseWriter, _ *http.Request) {
			if call == 1 {
				for key, value := range c.headers {
					w.Header().Set(key, value)
				}
				w.WriteHeader(c.status)
				return
			}
			_, _ = w.Write([]byte("{}"))
		})
		_, _ = newGithubClient(server.URL, "t", time.Millisecond).get(context.Background(), "/x", "application/json")
		if calls.Load() != c.calls {
			t.Errorf("%s: calls = %d, want %d", c.name, calls.Load(), c.calls)
		}
	}
}

func TestTheLastFailedAttemptDoesNotSleep(t *testing.T) {
	server, calls := countingServer(t, func(_ int32, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	client := newGithubClient(server.URL, "t", 100*time.Millisecond)

	start := time.Now()
	_, err := client.get(context.Background(), "/x", "application/json")

	if err == nil || calls.Load() != maxAttempts {
		t.Fatalf("err = %v, calls = %d", err, calls.Load())
	}
	if elapsed := time.Since(start); elapsed >= 500*time.Millisecond {
		t.Fatalf("took %v: backoff after the final attempt (100ms + 200ms is all it may wait)", elapsed)
	}
}

func TestAResponseOverTheCapFailsInsteadOfTruncating(t *testing.T) {
	server, _ := countingServer(t, func(_ int32, w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("0123456789A"))
	})
	client := newGithubClient(server.URL, "t", time.Millisecond)
	client.maxBody = 10

	if _, err := client.get(context.Background(), "/x", "application/json"); err == nil {
		t.Fatal("an 11-byte body under a 10-byte cap was accepted")
	}
}

func TestTagsAreReadFromEveryPage(t *testing.T) {
	server, _ := countingServer(t, func(_ int32, w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "1" {
			page := make([]map[string]string, pageSize)
			for i := range page {
				page[i] = map[string]string{"name": "v0.1." + strconv.Itoa(i)}
			}
			writeJSON(w, page)
			return
		}
		writeJSON(w, []map[string]string{{"name": "v0.17.0"}})
	})

	tags, err := newGithubClient(server.URL, "t", time.Millisecond).tags(context.Background(), "FacileStudio/muse")

	if err != nil || len(tags) != pageSize+1 || highestTag(tags) != "v0.17.0" {
		t.Fatalf("tags = %d, highest = %q, err = %v", len(tags), highestTag(tags), err)
	}
}

func TestClassifyAndSelectFollowDouaneAndStayInsideTheTree(t *testing.T) {
	cases := map[string]fileKind{
		"go.mod": kindLockfile, "apps/client/bun.lock": kindLockfile, "yarn.lock": kindLockfile,
		"web/pnpm-lock.yaml": kindLockfile, "src/App.csproj": kindLockfile, "apps/client/package.json": kindManifest,
		".github/workflows/ci.yml": kindWorkflow, "Dockerfile": kindBuild, "deploy/api.Dockerfile": kindBuild,
		"node_modules/x/package.json": kindNone, "apps/api/vendor/m/go.mod": kindNone, "../escape/go.mod": kindNone,
		"/etc/go.mod": kindNone, "README.md": kindNone,
	}
	for name, want := range cases {
		if got := classify(name); got != want {
			t.Errorf("classify(%q) = %d, want %d", name, got, want)
		}
	}
	chosen := selectFiles([]string{"go.mod", "go.sum", "tools/go.sum", "README.md"})
	if chosen.lockfiles != 2 || len(chosen.paths) != 2 || chosen.paths[1] != "tools/go.sum" {
		t.Fatalf("selection = %+v: a go.sum beside its go.mod is skipped, a lone one is kept", chosen)
	}
}

func TestWriteTreeRefusesAPathOutsideTheRepository(t *testing.T) {
	root := t.TempDir()
	err := writeTree(root, "11", map[string][]byte{"../../outside": []byte("x")})
	if err == nil {
		t.Fatal("a path escaping the repository was written")
	}
	if _, statErr := os.Stat(filepath.Join(root, "..", "outside")); statErr == nil {
		t.Fatal("the file landed outside the root")
	}
}
