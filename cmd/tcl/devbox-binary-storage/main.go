// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/dustin/go-humanize"

	"github.com/kubeshop/testkube/cmd/tcl/kubectl-testkube/devbox/devutils"
)

var (
	locks     = make(map[string]*sync.RWMutex)
	locksMu   sync.Mutex
	hashCache = make(map[string]string)
)

func getLock(filePath string) *sync.RWMutex {
	locksMu.Lock()
	defer locksMu.Unlock()
	if locks[filePath] == nil {
		locks[filePath] = new(sync.RWMutex)
	}
	return locks[filePath]
}

func rebuildHash(filePath string) {
	hashCache[filePath] = ""
	f, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err == nil {
		hashCache[filePath] = fmt.Sprintf("%x", h.Sum(nil))
	}
}

func getHash(filePath string) string {
	if hashCache[filePath] == "" {
		rebuildHash(filePath)
	}
	return hashCache[filePath]
}

func main() {
	storagePath := "/storage"
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		filePath := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		if filePath == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		localPath := filepath.Join(storagePath, filePath)
		switch r.Method {
		case http.MethodGet:
			getLock(filePath).RLock()
			defer getLock(filePath).RUnlock()

			file, err := os.Open(localPath)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			stat, err := file.Stat()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))
			w.WriteHeader(http.StatusOK)
			io.Copy(w, file)
			return
		case http.MethodPost:
			getLock(filePath).Lock()
			defer getLock(filePath).Unlock()

			body, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Println("failed reading body", err)
				return
			}
			if r.ContentLength != int64(len(body)) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if r.Header.Get("Content-Encoding") == "gzip" {
				gz, err := gzip.NewReader(bytes.NewBuffer(body))
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Println("failed reading body into gzip", err)
					return
				}
				body, err = io.ReadAll(gz)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Println("failed reading back data from gzip stream", err)
					return
				}
			}

			err = os.WriteFile(localPath, body, 0666)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Println("failed to write file", err)
				return
			}

			h := sha256.New()
			if _, err := io.Copy(h, bytes.NewBuffer(body)); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Println("failed to build hash", err)
			}
			hashCache[filePath] = fmt.Sprintf("%x", h.Sum(nil))

			fmt.Println("saved file", filePath, humanize.Bytes(uint64(len(body))))
			w.WriteHeader(http.StatusOK)
			return
		case http.MethodPatch:
			getLock(filePath).Lock()
			defer getLock(filePath).Unlock()

			body, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Println("failed reading body", err)
				return
			}
			if r.ContentLength != int64(len(body)) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if r.Header.Get("Content-Encoding") == "gzip" {
				gz, err := gzip.NewReader(bytes.NewBuffer(body))
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Println("failed reading body into gzip", err)
					return
				}
				body, err = io.ReadAll(gz)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Println("failed reading back data from gzip stream", err)
					return
				}
			}

			// Verify if patch can be applied
			if r.Header.Get("X-Prev-Hash") != getHash(filePath) {
				w.WriteHeader(http.StatusConflict)
				return
			}

			// Apply patch
			prevFile, err := os.ReadFile(localPath)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Println("failed reading existing file", err)
				return
			}
			patch := devutils.NewBinaryPatchFromBytes(body)
			file := patch.Apply(prevFile)

			h := sha256.New()
			if _, err := io.Copy(h, bytes.NewBuffer(file)); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Println("failed to build hash", err)
				return
			}

			// Validate hash
			nextHash := fmt.Sprintf("%x", h.Sum(nil))
			if r.Header.Get("X-Hash") != nextHash {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Println("after applying patch result has different hash than expected", err)
				return
			}
			fmt.Println("Expected hash", r.Header.Get("X-Hash"), "got", nextHash)
			err = os.WriteFile(localPath, file, 0666)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Println("failed to write file", err)
				return
			}
			hashCache[filePath] = nextHash
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stopSignal
		os.Exit(0)
	}()

	fmt.Println("Starting server...")

	panic(http.ListenAndServe(":8080", nil))
}
