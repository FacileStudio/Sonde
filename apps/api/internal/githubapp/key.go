// Package githubapp acts as a GitHub App installation: it signs the App's JWT,
// mints installation tokens scoped to the repositories and permissions one call
// needs, and answers what a GitHub account may do on a repository. Nothing here
// logs a token or the private key.
package githubapp

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

const (
	jwtBackdate = time.Minute
	jwtLifetime = 9 * time.Minute
)

// ParseKey reads the App's private key from base64 of its PEM, the form the
// environment carries it in. GitHub hands out PKCS#1 keys; PKCS#8 is accepted
// too so a converted key works.
func ParseKey(encoded string) (*rsa.PrivateKey, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return nil, fmt.Errorf("the GitHub App private key is not base64")
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("the GitHub App private key is not a PEM block")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("the GitHub App private key is not an RSA key")
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("the GitHub App private key is not an RSA key")
	}
	return key, nil
}

// appJWT signs the short-lived RS256 token that authenticates as the App
// itself. GitHub takes the App's client id as the issuer. The issue time is
// backdated a minute against clock drift, and the token lives under GitHub's
// ten-minute ceiling.
func appJWT(clientID string, key *rsa.PrivateKey, now time.Time) (string, error) {
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: key},
		(&jose.SignerOptions{}).WithType("JWT"),
	)
	if err != nil {
		return "", err
	}
	claims := jwt.Claims{
		Issuer:   clientID,
		IssuedAt: jwt.NewNumericDate(now.Add(-jwtBackdate)),
		Expiry:   jwt.NewNumericDate(now.Add(jwtLifetime)),
	}
	return jwt.Signed(signer).Claims(claims).Serialize()
}
