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
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

type client struct {
	addr string
}

type Client interface {
	Fetch(id string, mountPath string) error
}

func NewClient(addr string) Client {
	return &client{
		addr: addr,
	}
}

func (c *client) Fetch(id string, mountPath string) error {
	// Start downloading the file
	resp, err := http.Get(fmt.Sprintf("http://%s/%s", c.addr, id))
	if err != nil {
		return errors.Wrapf(err, "failed to download the transferred contents")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download the transferred contents: status code %d", resp.StatusCode)
	}

	// Process the files
	uncompressedStream, err := gzip.NewReader(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "failed to start reading gzip")
	}
	tarReader := tar.NewReader(uncompressedStream)

	// Unpack them
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "failed to get next entry from archive")
		}
		filePath := filepath.Join(mountPath, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(filePath, 0755); err != nil {
				return errors.Wrapf(err, "failed to create directory: %s", filePath)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				return errors.Wrapf(err, "failed to create directory tree for: %s", filePath)
			}
			outFile, err := os.Create(filePath)
			if err != nil {
				return errors.Wrapf(err, "failed to create file: %s", filePath)
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return errors.Wrapf(err, "failed to write file: %s", filePath)
			}
			outFile.Close()
		default:
			return fmt.Errorf("unknown entry type in the transferred archive: '%x' in %s", header.Typeflag, filePath)
		}
	}

	return nil
}
