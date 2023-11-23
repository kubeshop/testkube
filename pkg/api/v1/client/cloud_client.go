package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func NewCloudClient[A All](httpClient *http.Client, apiURI, apiPathPrefix string) CloudClient[A] {
	if apiPathPrefix == "" {
		apiPathPrefix = "/" + Version
	}

	return CloudClient[A]{
		client:        httpClient,
		sseClient:     httpClient,
		apiURI:        apiURI,
		apiPathPrefix: apiPathPrefix,
		DirectClient:  NewDirectClient[A](httpClient, apiURI, apiPathPrefix),
	}
}

// CLoudClient is almost the same as Direct client, but has different GetFile method
// which returns a download URL for the artifact instead of downloading it.
type CloudClient[A All] struct {
	client        *http.Client
	sseClient     *http.Client
	apiURI        string
	apiPathPrefix string
	DirectClient[A]
}

type ArtifactURL struct {
	// Download URL for the artifact.
	URL string `json:"url"`
}

// GetFile, in cloud we need to call non
func (t CloudClient[A]) GetFile(uri, fileName, destination string, params map[string][]string) (name string, err error) {

	cloudURI := strings.Replace(uri, "/agent", "", -1)
	req, err := http.NewRequest(http.MethodGet, cloudURI, nil)
	if err != nil {
		return "", err
	}

	q := req.URL.Query()
	for key, values := range params {
		for _, value := range values {
			if value != "" {
				q.Add(key, value)
			}
		}
	}
	req.URL.RawQuery = q.Encode()

	resp, err := t.client.Do(req)
	if err != nil {
		return name, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return name, fmt.Errorf("error: %d", resp.StatusCode)
	}

	var artifactURL ArtifactURL
	err = json.NewDecoder(resp.Body).Decode(&artifactURL)
	if err != nil {
		return "", err
	}

	req, err = http.NewRequest(http.MethodGet, artifactURL.URL, nil)
	if err != nil {
		return "", err
	}
	resp, err = t.client.Do(req)
	if err != nil {
		return name, err
	}
	defer resp.Body.Close()

	target := filepath.Join(destination, fileName)
	dir := filepath.Dir(target)
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		if err = os.MkdirAll(dir, os.ModePerm); err != nil {
			return name, err
		}
	} else if err != nil {
		return name, err
	}

	f, err := os.Create(target)
	if err != nil {
		return name, err
	}

	if _, err = io.Copy(f, resp.Body); err != nil {
		return name, err
	}

	if err = t.responseError(resp); err != nil {
		return name, fmt.Errorf("api/download-file returned error: %w", err)
	}

	return f.Name(), nil
}
