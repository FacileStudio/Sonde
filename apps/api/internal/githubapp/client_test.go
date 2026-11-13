package githubapp

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FacileStudio/tronc/errors"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

type fakeGitHub struct {
	key      *rsa.PrivateKey
	mints    atomic.Int32
	lookups  atomic.Int32
	lastMint map[string]any
	refuse   bool
}

func (f *fakeGitHub) appAuth(t *testing.T, r *http.Request) {
	parsed, err := jwt.ParseSigned(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "),
		[]jose.SignatureAlgorithm{jose.RS256})
	if err != nil {
		t.Errorf("app JWT: %v", err)
		return
	}
	var claims jwt.Claims
	if err := parsed.Claims(&f.key.PublicKey, &claims); err != nil || claims.Issuer != "Iv23test" {
		t.Errorf("app JWT claims %+v: %v", claims, err)
	}
}

func (f *fakeGitHub) mint(w http.ResponseWriter, r *http.Request) {
	f.mints.Add(1)
	_ = json.NewDecoder(r.Body).Decode(&f.lastMint)
	if f.refuse {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	w.WriteHeader(http.StatusCreated)
	expires := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	_, _ = w.Write([]byte(`{"token":"ghs_installation","expires_at":"` + expires + `"}`))
}

func (f *fakeGitHub) handler(t *testing.T) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /app/installations", func(w http.ResponseWriter, r *http.Request) {
		f.appAuth(t, r)
		_, _ = w.Write([]byte(`[{"id":7,"account":{"login":"FacileStudio"}}]`))
	})
	mux.HandleFunc("POST /app/installations/7/access_tokens", func(w http.ResponseWriter, r *http.Request) {
		f.appAuth(t, r)
		f.mint(w, r)
	})
	mux.HandleFunc("GET /user/42", func(w http.ResponseWriter, r *http.Request) {
		f.lookups.Add(1)
		_, _ = w.Write([]byte(`{"login":"octo"}`))
	})
	mux.HandleFunc("GET /repos/FacileStudio/{repo}/collaborators/octo/permission", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer ghs_installation" {
			t.Errorf("permission check sent %q", r.Header.Get("Authorization"))
		}
		if r.PathValue("repo") == "Secret" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`{"permission":"write"}`))
	})
	return mux
}

func newFake(t *testing.T) (*fakeGitHub, *Client) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	fake := &fakeGitHub{key: key}
	server := httptest.NewServer(fake.handler(t))
	t.Cleanup(server.Close)
	encoded := base64.StdEncoding.EncodeToString(pem.EncodeToMemory(&pem.Block{
		Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key),
	}))
	client, err := New(Config{ClientID: "Iv23test", PrivateKey: encoded, BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	return fake, client
}

func TestATokenIsScopedAndCachedUntilNearItsExpiry(t *testing.T) {
	fake, client := newFake(t)
	scope := Scope{Owner: "FacileStudio", Repositories: []string{"Sonde"}, Permissions: map[string]string{"actions": "write"}}
	for range 2 {
		got, err := client.Token(context.Background(), scope)
		if err != nil || got != "ghs_installation" {
			t.Fatalf("token %q: %v", got, err)
		}
	}
	if fake.mints.Load() != 1 {
		t.Fatalf("minted %d tokens, want 1", fake.mints.Load())
	}
	repos, _ := json.Marshal(fake.lastMint["repositories"])
	perms, _ := json.Marshal(fake.lastMint["permissions"])
	if string(repos) != `["Sonde"]` || string(perms) != `{"actions":"write"}` {
		t.Fatalf("mint body repositories %s permissions %s", repos, perms)
	}
	client.now = func() time.Time { return time.Now().Add(56 * time.Minute) }
	if _, err := client.Token(context.Background(), scope); err != nil || fake.mints.Load() != 2 {
		t.Fatalf("a token five minutes from expiry was reused: mints %d, %v", fake.mints.Load(), err)
	}
}

func TestAPermissionTheInstallationLacksIsAPlainPrecondition(t *testing.T) {
	fake, client := newFake(t)
	fake.refuse = true
	_, err := client.Token(context.Background(), Scope{Owner: "FacileStudio", Permissions: map[string]string{"actions": "write"}})
	if errors.Status(err) != http.StatusPreconditionFailed || !strings.Contains(err.Error(), "approve the new permission") {
		t.Fatalf("err = %v", err)
	}
	if _, err := client.Token(context.Background(), Scope{Owner: "Elsewhere"}); errors.Status(err) != http.StatusPreconditionFailed {
		t.Fatalf("an owner without the App: %v", err)
	}
}

func TestPermissionResolvesTheLoginAndCachesTheAnswer(t *testing.T) {
	fake, client := newFake(t)
	ctx := context.Background()
	for range 2 {
		if level, err := client.Permission(ctx, "FacileStudio", "Sonde", "42"); err != nil || level != Write {
			t.Fatalf("Sonde = %q, %v", level, err)
		}
	}
	if level, err := client.Permission(ctx, "FacileStudio", "Secret", "42"); err != nil || level != None {
		t.Fatalf("Secret = %q, %v", level, err)
	}
	if fake.lookups.Load() != 1 || fake.mints.Load() != 1 {
		t.Fatalf("lookups %d mints %d, want one of each", fake.lookups.Load(), fake.mints.Load())
	}
	if !AtLeast(Write, Read) || AtLeast(Read, Write) || AtLeast(None, None) || AtLeast("maintain", Read) {
		t.Fatal("AtLeast ranks the levels wrong")
	}
}

func TestAnUnconfiguredAppIsOffAndAHalfConfiguredOneIsRefused(t *testing.T) {
	client, err := New(Config{})
	if err != nil || client.Enabled() {
		t.Fatalf("empty config: %v, enabled %v", err, client.Enabled())
	}
	if _, err := client.Permission(context.Background(), "o", "r", "1"); errors.Status(err) != http.StatusPreconditionFailed {
		t.Fatalf("unconfigured permission: %v", err)
	}
	if _, err := New(Config{ClientID: "Iv23"}); err == nil {
		t.Fatal("a client id without a key was accepted")
	}
	if _, err := New(Config{ClientID: "Iv23", PrivateKey: base64.StdEncoding.EncodeToString([]byte("nope"))}); err == nil {
		t.Fatal("a key that is not PEM was accepted")
	}
}
