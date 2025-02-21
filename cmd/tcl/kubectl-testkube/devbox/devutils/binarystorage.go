// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package devutils

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	errors2 "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/artifacts"
	"github.com/kubeshop/testkube/internal/common"
)

type BinaryStorage struct {
	pod       *PodObject
	binary    *Binary
	localPort int
	hashes    map[string]string
	hashMu    sync.RWMutex
}

func NewBinaryStorage(pod *PodObject, binary *Binary) *BinaryStorage {
	return &BinaryStorage{
		pod:    pod,
		binary: binary,
		hashes: make(map[string]string),
	}
}

func (r *BinaryStorage) Create(ctx context.Context) error {
	if r.binary.Hash() == "" {
		return errors2.New("binary storage server binary is not built")
	}

	// Deploy Pod
	err := r.pod.Create(ctx, &corev1.Pod{
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: common.Ptr(int64(1)),
			Volumes: []corev1.Volume{
				{Name: "server", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
				{Name: "storage", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
			},
			Containers: []corev1.Container{
				{
					Name:            "binary-storage",
					Image:           "busybox:1.36.1-musl",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"/bin/sh", "-c", fmt.Sprintf("while [ ! -f /app/server-ready ]; do sleep 1; done\n/app/server")},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "server", MountPath: "/app"},
						{Name: "storage", MountPath: "/storage"},
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/health",
								Port:   intstr.FromInt32(8080),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						PeriodSeconds: 1,
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	// Wait for the container to be started
	err = r.pod.WaitForContainerStarted(ctx)
	if err != nil {
		return err
	}

	// Deploy Service
	err = r.pod.CreateService(ctx, corev1.ServicePort{
		Name:       "api",
		Protocol:   "TCP",
		Port:       8080,
		TargetPort: intstr.FromInt32(8080),
	})
	if err != nil {
		return err
	}

	// TODO: Move transfer utilities to *PodObject
	// Apply the binary
	req := r.pod.ClientSet().CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(r.pod.Name()).
		Namespace(r.pod.Namespace()).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: "binary-storage",
			Command:   []string{"tar", "-xzf", "-", "-C", "/app"},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(r.pod.RESTConfig(), "POST", req.URL())
	if err != nil {
		return errors2.Wrap(err, "failed to create spdy executor")
	}

	os.WriteFile("/tmp/flag", []byte{1}, 0777)
	flagFile, err := os.Open("/tmp/flag")
	if err != nil {
		return errors2.Wrap(err, "failed to open flag file")
	}
	defer flagFile.Close()
	flagFileStat, err := flagFile.Stat()
	if err != nil {
		return err
	}

	file, err := os.Open(r.binary.Path())
	if err != nil {
		return err
	}
	defer file.Close()
	fileStat, err := file.Stat()
	if err != nil {
		return err
	}

	tarStream := artifacts.NewTarStream()
	go func() {
		defer tarStream.Close()
		tarStream.Add("server", file, fileStat)
		tarStream.Add("server-ready", flagFile, flagFileStat)
	}()

	reader, writer := io.Pipe()
	var buf []byte
	var bufMu sync.Mutex
	go func() {
		bufMu.Lock()
		defer bufMu.Unlock()
		buf, _ = io.ReadAll(reader)
	}()
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  tarStream,
		Stdout: writer,
		Stderr: writer,
		Tty:    false,
	})
	if err != nil {
		writer.Close()
		bufMu.Lock()
		defer bufMu.Unlock()
		return fmt.Errorf("failed to stream binary: %s: %s", err.Error(), string(buf))
	}
	writer.Close()

	err = r.pod.WaitForReady(ctx)
	if err != nil {
		return err
	}

	r.localPort = GetFreePort()
	err = r.pod.Forward(ctx, 8080, r.localPort, true)
	if err != nil {
		return err
	}

	return nil
}

func (r *BinaryStorage) WaitForReady(ctx context.Context) error {
	return r.pod.WaitForReady(ctx)
}

func (r *BinaryStorage) Is(path string, hash string) bool {
	r.hashMu.RLock()
	defer r.hashMu.RUnlock()
	return r.hashes[path] == hash
}

func (r *BinaryStorage) SetHash(path string, hash string) {
	r.hashMu.Lock()
	defer r.hashMu.Unlock()
	r.hashes[path] = hash
}

func (r *BinaryStorage) Upload(ctx context.Context, name string, binary *Binary) (cached bool, size int, err error) {
	binary.buildMu.RLock()
	defer binary.buildMu.RUnlock()
	if binary.hash != "" && r.Is(name, binary.hash) {
		return true, 0, nil
	}
	for i := 0; i < 5; i++ {
		size, err = r.upload(ctx, name, binary)
		if err == nil {
			return
		}
		if ctx.Err() != nil {
			return false, 0, err
		}
		time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
	}
	return false, size, err
}

func (r *BinaryStorage) upload(ctx context.Context, name string, binary *Binary) (int, error) {
	file, err := os.Open(binary.outputPath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{Transport: tr}

	if binary.hash != "" && binary.prevHash != "" && r.Is(name, binary.prevHash) {
		contents, err := binary.patch()
		if err == nil {
			gzipContents := bytes.NewBuffer(nil)
			gz := gzip.NewWriter(gzipContents)
			io.Copy(gz, bytes.NewBuffer(contents))
			gz.Flush()
			gz.Close()

			gzipContentsLen := gzipContents.Len()
			req, err := http.NewRequestWithContext(ctx, http.MethodPatch, fmt.Sprintf("http://localhost:%d/%s", r.localPort, name), gzipContents)
			if err != nil {
				if ctx.Err() != nil {
					return 0, err
				}
				fmt.Printf("error while sending %s patch, fallback to full stream: %s\n", name, err)
			} else {
				req.ContentLength = int64(gzipContentsLen)
				req.Header.Set("Content-Encoding", "gzip")
				req.Header.Set("X-Prev-Hash", binary.prevHash)
				req.Header.Set("X-Hash", binary.hash)
				res, err := client.Do(req)
				if err != nil {
					fmt.Printf("error while sending %s patch, fallback to full stream: %s\n", name, err)
				} else if res.StatusCode != http.StatusOK {
					b, _ := io.ReadAll(res.Body)
					fmt.Printf("error while sending %s patch, fallback to full stream: status code: %s, message: %s\n", name, res.Status, string(b))
				} else {
					r.SetHash(name, binary.hash)
					return gzipContentsLen, nil
				}
			}
		}
	}

	buf := bytes.NewBuffer(nil)
	gz := gzip.NewWriter(buf)
	io.Copy(gz, file)
	gz.Flush()
	gz.Close()
	bufLen := buf.Len()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("http://localhost:%d/%s", r.localPort, name), buf)
	if err != nil {
		return bufLen, err
	}
	req.ContentLength = int64(bufLen)
	req.Header.Set("Content-Encoding", "gzip")

	res, err := client.Do(req)
	if err != nil {
		return bufLen, err
	}
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		return bufLen, fmt.Errorf("failed saving file: status code: %d / message: %s", res.StatusCode, string(b))
	}
	r.SetHash(name, binary.hash)
	return bufLen, nil
}
