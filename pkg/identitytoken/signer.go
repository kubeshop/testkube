package identitytoken

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const (
	// TokenTTL is how long a token is valid: enough to exchange it with a
	// cloud provider at the start of a step, short enough that a leaked token
	// is of little use.
	TokenTTL = 10 * time.Minute

	// minKeyBits is the smallest RSA key cloud providers accept for RS256.
	minKeyBits = 2048

	// maxAudienceLength bounds the audience the agent sends; real audiences
	// (URLs, ARNs, Azure "api://AzureADTokenExchange") are far shorter.
	maxAudienceLength = 512

	DiscoveryPath = "/.well-known/openid-configuration"
	JWKSPath      = "/.well-known/jwks.json"
)

var ErrInvalidAudience = errors.New("audience must be a non-empty string of at most 512 characters")

// Subject identifies the execution a token is minted for.
type Subject struct {
	OrganizationID string
	EnvironmentID  string
	WorkflowName   string
	ExecutionID    string
	RunnerID       string
}

func (s Subject) String() string {
	return fmt.Sprintf("organization:%s:environment:%s:workflow:%s", s.OrganizationID, s.EnvironmentID, s.WorkflowName)
}

type claims struct {
	jwt.RegisteredClaims
	OrganizationID string `json:"organization_id"`
	EnvironmentID  string `json:"environment_id"`
	Workflow       string `json:"workflow"`
	ExecutionID    string `json:"execution_id"`
	RunnerID       string `json:"runner_id,omitempty"`
}

// Signer mints identity tokens and serves the documents that verify them.
type Signer struct {
	issuer string
	key    *rsa.PrivateKey
	keyID  string
	// executionKey authenticates callers of a standalone control plane; see ExecutionToken.
	executionKey []byte
	now          func() time.Time
}

// NewSignerFromFile reads the PEM private key at keyFile.
func NewSignerFromFile(issuer, keyFile string) (*Signer, error) {
	data, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("reading identity token private key: %w", err)
	}
	key, err := parsePrivateKey(data)
	if err != nil {
		return nil, err
	}
	return NewSigner(issuer, key)
}

func NewSigner(issuer string, key *rsa.PrivateKey) (*Signer, error) {
	u, err := url.Parse(issuer)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return nil, fmt.Errorf("identity token issuer must be an absolute URL, got %q", issuer)
	}
	// Discovery appends the well-known paths to the issuer, which would land
	// inside a query or fragment; even an empty "?" or "#" breaks jwks_uri.
	if strings.ContainsAny(issuer, "?#") {
		return nil, fmt.Errorf("identity token issuer must not have a query or fragment, got %q", issuer)
	}
	if key == nil || key.N.BitLen() < minKeyBits {
		return nil, fmt.Errorf("identity token private key must be RSA with at least %d bits", minKeyBits)
	}
	return &Signer{
		issuer:       strings.TrimSuffix(issuer, "/"),
		key:          key,
		keyID:        thumbprint(&key.PublicKey),
		executionKey: deriveExecutionKey(key),
		now:          time.Now,
	}, nil
}

func parsePrivateKey(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("identity token private key is not PEM encoded")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing identity token private key: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("identity token private key must be an RSA key")
	}
	return key, nil
}

func (s *Signer) Issuer() string {
	return s.issuer
}

// Sign mints a token for the subject, scoped to the audience.
func (s *Signer) Sign(subject Subject, audience string) (string, error) {
	if audience == "" || len(audience) > maxAudienceLength {
		return "", ErrInvalidAudience
	}
	jti := make([]byte, 16)
	if _, err := rand.Read(jti); err != nil {
		return "", fmt.Errorf("generating token ID: %w", err)
	}
	now := s.now()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   subject.String(),
			Audience:  jwt.ClaimStrings{audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(TokenTTL)),
			ID:        hex.EncodeToString(jti),
		},
		OrganizationID: subject.OrganizationID,
		EnvironmentID:  subject.EnvironmentID,
		Workflow:       subject.WorkflowName,
		ExecutionID:    subject.ExecutionID,
		RunnerID:       subject.RunnerID,
	})
	token.Header["kid"] = s.keyID
	return token.SignedString(s.key)
}

type jwk struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func publicJWK(key *rsa.PublicKey) (n, e string) {
	return base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())
}

// thumbprint is the RFC 7638 JWK thumbprint, a key ID that stays the same for
// the same key across restarts and replicas.
func thumbprint(key *rsa.PublicKey) string {
	n, e := publicJWK(key)
	// RFC 7638 fixes the members and their order, so the string is built by hand.
	sum := sha256.Sum256([]byte(`{"e":"` + e + `","kty":"RSA","n":"` + n + `"}`))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// JWKS returns the JWK set with the public signing key.
func (s *Signer) JWKS() ([]byte, error) {
	n, e := publicJWK(&s.key.PublicKey)
	return json.Marshal(map[string][]jwk{
		"keys": {{Kty: "RSA", Use: "sig", Alg: "RS256", Kid: s.keyID, N: n, E: e}},
	})
}

// Discovery returns the OIDC discovery document for the issuer.
func (s *Signer) Discovery() ([]byte, error) {
	return json.Marshal(map[string]any{
		"issuer":                                s.issuer,
		"jwks_uri":                              s.issuer + JWKSPath,
		"response_types_supported":              []string{"id_token"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"claims_supported": []string{
			"iss", "sub", "aud", "exp", "iat", "nbf", "jti",
			"organization_id", "environment_id", "workflow", "execution_id", "runner_id",
		},
	})
}

// executionKeyLabel separates the execution token key from any other use of
// the signing key.
const executionKeyLabel = "testkube identity-token execution-token v1"

func deriveExecutionKey(key *rsa.PrivateKey) []byte {
	mac := hmac.New(sha256.New, x509.MarshalPKCS1PrivateKey(key))
	mac.Write([]byte(executionKeyLabel))
	return mac.Sum(nil)
}

// ExecutionToken is the secret a standalone control plane hands to the
// execution's pods, and that GetIdentityToken requires back. It is derived
// rather than stored, so every replica loading the key agrees on it and it
// survives restarts; it stops working when the execution finishes, because
// tokens are only minted for running executions.
func (s *Signer) ExecutionToken(environmentID, executionID string) string {
	mac := hmac.New(sha256.New, s.executionKey)
	// The NUL separator keeps the two IDs from running into each other.
	mac.Write([]byte(environmentID + "\x00" + executionID))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// VerifyExecutionToken reports whether token is the execution's token.
func (s *Signer) VerifyExecutionToken(environmentID, executionID, token string) bool {
	if token == "" {
		return false
	}
	return hmac.Equal([]byte(token), []byte(s.ExecutionToken(environmentID, executionID)))
}
