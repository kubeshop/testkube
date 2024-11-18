// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package devutils

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

type FsWatcher struct {
	watcher *fsnotify.Watcher
}

// TODO: support masks like **/*.go
func NewFsWatcher(paths ...string) (*FsWatcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &FsWatcher{
		watcher: fsWatcher,
	}
	for i := range paths {
		if err = w.add(paths[i]); err != nil {
			fsWatcher.Close()
			return nil, err
		}
	}
	return w, nil
}

func (w *FsWatcher) Close() error {
	return w.watcher.Close()
}

func (w *FsWatcher) addRecursive(dirPath string) error {
	if err := w.watcher.Add(dirPath); err != nil {
		return err
	}
	return filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if filepath.Base(path)[0] == '.' {
			// Ignore dot-files
			return nil
		}
		if path == dirPath {
			return nil
		}
		return w.addRecursive(path)
	})
}

func (w *FsWatcher) add(path string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	return w.addRecursive(path)
}

func (w *FsWatcher) Next(ctx context.Context) (string, error) {
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case event, ok := <-w.watcher.Events:
			if !ok {
				return "", io.EOF
			}
			fileinfo, err := os.Stat(event.Name)
			if err != nil {
				continue
			}
			if fileinfo.IsDir() {
				if event.Has(fsnotify.Create) {
					if err = w.addRecursive(event.Name); err != nil {
						return "", err
					}
				}
				continue
			}
			if !event.Has(fsnotify.Create) && !event.Has(fsnotify.Write) && !event.Has(fsnotify.Remove) {
				continue
			}
			return event.Name, nil
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return "", io.EOF
			}
			return "", err
		}
	}
}
