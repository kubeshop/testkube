package identitytoken

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testIssuer = "https://api.example.com"

func newTestKey(t *testing.T, bits int) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, bits)
	require.NoError(t, err)
	return key
}

var testSubject = Subject{
	OrganizationID: "tkcorg_1",
	EnvironmentID:  "tkcenv_1",
	WorkflowName:   "deploy",
	ExecutionID:    "exec-1",
	RunnerID:       "tkcroot_1",
}

// publicKeyFromJWKS rebuilds the key a provider would verify with, so the test
// checks the published JWK set rather than the signer's own key.
func publicKeyFromJWKS(t *testing.T, s *Signer, kid string) *rsa.PublicKey {
	t.Helper()
	body, err := s.JWKS()
	require.NoError(t, err)
	var set struct {
		Keys []jwk `json:"keys"`
	}
	require.NoError(t, json.Unmarshal(body, &set))
	require.Len(t, set.Keys, 1)
	k := set.Keys[0]
	require.Equal(t, kid, k.Kid)
	assert.Equal(t, "RSA", k.Kty)
	assert.Equal(t, "RS256", k.Alg)
	n, err := base64.RawURLEncoding.DecodeString(k.N)
	require.NoError(t, err)
	e, err := base64.RawURLEncoding.DecodeString(k.E)
	require.NoError(t, err)
	return &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
}

// A provider verifies the token with the published key and trusts it by
// issuer, audience and subject.
func TestSign_VerifiesWithPublishedKey(t *testing.T) {
	signer, err := NewSigner(testIssuer+"/", newTestKey(t, 2048))
	require.NoError(t, err)
	now := time.Now().Truncate(time.Second)
	signer.now = func() time.Time { return now }

	raw, err := signer.Sign(testSubject, "sts.amazonaws.com")
	require.NoError(t, err)

	var got claims
	parser := jwt.NewParser(jwt.WithValidMethods([]string{"RS256"}))
	token, err := parser.ParseWithClaims(raw, &got, func(token *jwt.Token) (any, error) {
		return publicKeyFromJWKS(t, signer, token.Header["kid"].(string)), nil
	})
	require.NoError(t, err)
	require.True(t, token.Valid)
	assert.Equal(t, testIssuer, got.Issuer)
	assert.True(t, got.VerifyAudience("sts.amazonaws.com", true))

	assert.Equal(t, "organization:tkcorg_1:environment:tkcenv_1:workflow:deploy", got.Subject)
	assert.Equal(t, now.Add(TokenTTL), got.ExpiresAt.Time)
	assert.NotEmpty(t, got.ID)
	assert.Equal(t, "tkcorg_1", got.OrganizationID)
	assert.Equal(t, "tkcenv_1", got.EnvironmentID)
	assert.Equal(t, "deploy", got.Workflow)
	assert.Equal(t, "exec-1", got.ExecutionID)
	assert.Equal(t, "tkcroot_1", got.RunnerID)
}

// Each token carries its own ID, so providers that reject replays accept the
// next call.
func TestSign_UniqueTokenIDs(t *testing.T) {
	signer, err := NewSigner(testIssuer, newTestKey(t, 2048))
	require.NoError(t, err)
	a, err := signer.Sign(testSubject, "aud")
	require.NoError(t, err)
	b, err := signer.Sign(testSubject, "aud")
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
}

func TestSign_RejectsInvalidAudience(t *testing.T) {
	signer, err := NewSigner(testIssuer, newTestKey(t, 2048))
	require.NoError(t, err)
	for name, audience := range map[string]string{
		"empty":    "",
		"too long": string(make([]byte, maxAudienceLength+1)),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := signer.Sign(testSubject, audience)
			assert.ErrorIs(t, err, ErrInvalidAudience)
		})
	}
}

func TestNewSigner_Validation(t *testing.T) {
	key := newTestKey(t, 2048)
	tests := []struct {
		name   string
		issuer string
		key    *rsa.PrivateKey
	}{
		{name: "empty issuer", issuer: "", key: key},
		{name: "relative issuer", issuer: "api.example.com", key: key},
		{name: "non-http issuer", issuer: "ftp://api.example.com", key: key},
		{name: "missing key", issuer: testIssuer, key: nil},
		{name: "short key", issuer: testIssuer, key: newTestKey(t, 1024)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewSigner(tc.issuer, tc.key)
			assert.Error(t, err)
		})
	}
}

// Both PEM encodings that openssl produces are accepted, and the key ID only
// depends on the key, so replicas loading the same file publish the same kid.
func TestNewSignerFromFile_PEMFormats(t *testing.T) {
	key := newTestKey(t, 2048)
	pkcs8, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	blocks := map[string]*pem.Block{
		"pkcs1": {Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)},
		"pkcs8": {Type: "PRIVATE KEY", Bytes: pkcs8},
	}
	var kids []string
	for name, block := range blocks {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "key.pem")
			require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(block), 0o600))
			signer, err := NewSignerFromFile(testIssuer, path)
			require.NoError(t, err)
			kids = append(kids, signer.keyID)
		})
	}
	require.Len(t, kids, 2)
	assert.Equal(t, kids[0], kids[1])

	t.Run("not PEM", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "key.pem")
		require.NoError(t, os.WriteFile(path, []byte("not a key"), 0o600))
		_, err := NewSignerFromFile(testIssuer, path)
		assert.Error(t, err)
	})
}

// RFC 7638 section 3.1 gives the thumbprint of this key.
func TestThumbprint_RFC7638Example(t *testing.T) {
	n, err := base64.RawURLEncoding.DecodeString("0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw")
	require.NoError(t, err)
	key := &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: 65537}
	assert.Equal(t, "NzbLsXh8uDCcd-6MNwXF4W_7noWXFZAfHkxZsRGC9Xs", thumbprint(key))
}

func TestHandlers(t *testing.T) {
	signer, err := NewSigner(testIssuer, newTestKey(t, 2048))
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	signer.HandleDiscovery(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, DiscoveryPath, nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	var discovery map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &discovery))
	assert.Equal(t, testIssuer, discovery["issuer"])
	assert.Equal(t, testIssuer+JWKSPath, discovery["jwks_uri"])

	rec = httptest.NewRecorder()
	signer.HandleJWKS(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, JWKSPath, nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), signer.keyID)
}

// The execution token is the same for every replica loading the key and for a
// re-dispatched execution, and differs per execution and per key.
func TestExecutionToken(t *testing.T) {
	key := newTestKey(t, 2048)
	a, err := NewSigner(testIssuer, key)
	require.NoError(t, err)
	b, err := NewSigner(testIssuer, key)
	require.NoError(t, err)
	other, err := NewSigner(testIssuer, newTestKey(t, 2048))
	require.NoError(t, err)

	token := a.ExecutionToken("env-1", "exec-1")
	assert.NotEmpty(t, token)
	assert.Equal(t, token, b.ExecutionToken("env-1", "exec-1"))
	assert.True(t, a.VerifyExecutionToken("env-1", "exec-1", token))

	assert.False(t, a.VerifyExecutionToken("env-1", "exec-2", token), "another execution")
	assert.False(t, a.VerifyExecutionToken("env-2", "exec-1", token), "another environment")
	assert.False(t, other.VerifyExecutionToken("env-1", "exec-1", token), "another key")
	assert.False(t, a.VerifyExecutionToken("env-1", "exec-1", ""), "no token")
	// The separator keeps the IDs from running into each other.
	assert.NotEqual(t, a.ExecutionToken("env-1", "0exec"), a.ExecutionToken("env-10", "exec"))
}
