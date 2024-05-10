// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package transfer

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/artifacts"
)

type server struct {
	files       map[string]struct{}
	storagePath string
	host        string
	port        int
}

type Server interface {
	Count() int
	Has(dirPath string, files []string) bool
	Include(dirPath string, files []string) (Entry, error)
	Listen() (func(), error)
}

type Entry struct {
	Id  string `json:"id"`
	Url string `json:"url"`
}

func NewServer(storagePath string, host string, port int) Server {
	return &server{
		files:       make(map[string]struct{}),
		storagePath: storagePath,
		host:        host,
		port:        port,
	}
}

func (t *server) Count() int {
	return len(t.files)
}

func (t *server) Has(dirPath string, files []string) bool {
	_, ok := t.files[SourceID(dirPath, files)]
	return ok
}

func (t *server) GetUrl(id string) string {
	return fmt.Sprintf("http://%s:%d/%s.tar.gz", t.host, t.port, id)
}

func (t *server) Include(dirPath string, files []string) (Entry, error) {
	id := SourceID(dirPath, files)

	if !filepath.IsAbs(dirPath) {
		var err error
		dirPath, err = filepath.Abs(dirPath)
		if err != nil {
			return Entry{}, errors.Wrap(err, "failed to build absolute path for inclusion")
		}
	}

	// Ensure that is not prepared already
	if _, ok := t.files[id]; ok {
		return Entry{Id: id, Url: t.GetUrl(id)}, nil
	}

	// Access the file on the disk
	fileStream, err := os.Create(filepath.Join(t.storagePath, fmt.Sprintf("%s.tar.gz", id)))
	if err != nil {
		return Entry{}, err
	}
	defer fileStream.Close()

	// Prepare files archive
	gzipStream := gzip.NewWriter(fileStream)
	tarStream := tar.NewWriter(gzipStream)
	defer gzipStream.Close()
	defer tarStream.Close()

	// Append all the files
	walker, err := artifacts.CreateWalker(files, []string{dirPath}, dirPath)
	if err != nil {
		return Entry{}, err
	}
	err = walker.Walk(os.DirFS("/"), func(path string, file fs.File, stat fs.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Warning: '%s' has been ignored, as there was a problem reading it: %s\n", path, err.Error())
			return nil
		}

		// Append the file to the archive
		name := stat.Name()
		link := name
		isSymlink := stat.Mode()&fs.ModeSymlink != 0
		if isSymlink {
			link, err = os.Readlink(filepath.Join(dirPath, path))
			if err != nil {
				fmt.Printf("Warning: '%s' has been ignored, as there was a problem reading link: %s\n", path, err.Error())
				return nil
			}
		}

		// Build the data
		header, err := tar.FileInfoHeader(stat, link)
		if err != nil {
			return err
		}
		header.Name = path
		err = tarStream.WriteHeader(header)
		if err != nil {
			return err
		}

		// Copy the contents for regular files
		if !isSymlink {
			_, err = io.Copy(tarStream, file)
		}

		return err
	})
	if err != nil {
		return Entry{}, err
	}

	t.files[id] = struct{}{}
	return Entry{Id: id, Url: t.GetUrl(id)}, nil
}

func (t *server) Listen() (func(), error) {
	handler := http.FileServer(http.Dir(t.storagePath))
	addr := fmt.Sprintf(":%d", t.port)
	srv := http.Server{Addr: addr, Handler: handler}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	stop := func() {
		_ = srv.Shutdown(context.Background())
	}
	go func() {
		_ = srv.Serve(listener)
	}()
	return stop, err
}
