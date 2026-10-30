package identitytoken

import (
	"net/http"
)

// cacheMaxAge lets providers cache the documents while a rotated key still
// shows up within minutes.
const cacheMaxAge = "public, max-age=300"

// HandleDiscovery serves DiscoveryPath.
func (s *Signer) HandleDiscovery(w http.ResponseWriter, _ *http.Request) {
	writeDocument(w, s.Discovery)
}

// HandleJWKS serves JWKSPath.
func (s *Signer) HandleJWKS(w http.ResponseWriter, _ *http.Request) {
	writeDocument(w, s.JWKS)
}

func writeDocument(w http.ResponseWriter, build func() ([]byte, error)) {
	body, err := build()
	if err != nil {
		http.Error(w, "failed to build document", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", cacheMaxAge)
	_, _ = w.Write(body)
}
