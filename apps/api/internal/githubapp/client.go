package githubapp

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/FacileStudio/tronc/errors"
)

const (
	defaultAPI     = "https://api.github.com"
	requestTimeout = 15 * time.Second
	tokenMargin    = 5 * time.Minute
)

// NotConfigured is the reason every GitHub feature gives when the App's
// environment is missing.
const NotConfigured = "GitHub is not configured: set SONDE_GITHUB_APP_CLIENT_ID and SONDE_GITHUB_APP_PRIVATE_KEY"

// Config names the App. PrivateKey is base64 of the PEM. BaseURL and HTTP are
// for tests and default to api.github.com and a 15-second client.
type Config struct {
	ClientID   string
	PrivateKey string
	BaseURL    string
	HTTP       *http.Client
}

// Scope is what one installation token may touch: the owner's installation,
// the named repositories of that owner (all of them when empty), and the
// permissions, such as {"actions": "write"}.
type Scope struct {
	Owner        string
	Repositories []string
	Permissions  map[string]string
}

func (s Scope) key() string {
	perms := make([]string, 0, len(s.Permissions))
	for name, level := range s.Permissions {
		perms = append(perms, name+":"+level)
	}
	slices.Sort(perms)
	repos := slices.Sorted(slices.Values(s.Repositories))
	return strings.ToLower(s.Owner) + "|" + strings.Join(repos, ",") + "|" + strings.Join(perms, ",")
}

type token struct {
	value   string
	expires time.Time
}

// Client is the App. A nil *Client is the unconfigured App: Enabled reports
// false and every call answers failed_precondition with NotConfigured.
type Client struct {
	clientID string
	key      *rsa.PrivateKey
	baseURL  string
	http     *http.Client
	now      func() time.Time
	mu       sync.Mutex
	installs map[string]int64
	tokens   map[string]token
	access   map[string]cached
}

// New builds the App from its configuration. Both fields empty gives a nil
// client, the App switched off; one without the other, or a key that does not
// parse, is an error so a half-configured deploy fails at boot.
func New(cfg Config) (*Client, error) {
	if cfg.ClientID == "" && cfg.PrivateKey == "" {
		return nil, nil
	}
	if cfg.ClientID == "" || cfg.PrivateKey == "" {
		return nil, fmt.Errorf("set both SONDE_GITHUB_APP_CLIENT_ID and SONDE_GITHUB_APP_PRIVATE_KEY, or neither")
	}
	key, err := ParseKey(cfg.PrivateKey)
	if err != nil {
		return nil, err
	}
	client := &Client{
		clientID: cfg.ClientID, key: key, baseURL: strings.TrimRight(cfg.BaseURL, "/"), http: cfg.HTTP, now: time.Now,
		installs: map[string]int64{}, tokens: map[string]token{}, access: map[string]cached{},
	}
	if client.baseURL == "" {
		client.baseURL = defaultAPI
	}
	if client.http == nil {
		client.http = &http.Client{Timeout: requestTimeout}
	}
	return client, nil
}

// Enabled reports whether the App is configured.
func (c *Client) Enabled() bool { return c != nil }

// API is the GitHub REST base URL the client calls.
func (c *Client) API() string { return c.baseURL }

// Token returns an installation token limited to scope, minting one when the
// cached token is missing or expires within five minutes.
func (c *Client) Token(ctx context.Context, scope Scope) (string, error) {
	if !c.Enabled() {
		return "", errors.Failed(NotConfigured)
	}
	key := scope.key()
	c.mu.Lock()
	cachedToken, ok := c.tokens[key]
	c.mu.Unlock()
	if ok && c.now().Add(tokenMargin).Before(cachedToken.expires) {
		return cachedToken.value, nil
	}
	installation, err := c.installation(ctx, scope.Owner)
	if err != nil {
		return "", err
	}
	minted, err := c.mint(ctx, installation, scope)
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	c.tokens[key] = minted
	c.mu.Unlock()
	return minted.value, nil
}
