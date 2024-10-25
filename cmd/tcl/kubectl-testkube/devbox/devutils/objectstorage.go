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
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	minio2 "github.com/minio/minio-go/v7"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/artifacts"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/storage/minio"
)

type ObjectStorage struct {
	pod            *PodObject
	localPort      int
	hashes         map[string]string
	hashMu         sync.RWMutex
	cachedClient   *minio2.Client
	cachedClientMu sync.Mutex
}

func NewObjectStorage(pod *PodObject) *ObjectStorage {
	return &ObjectStorage{
		pod:    pod,
		hashes: make(map[string]string),
	}
}

func (r *ObjectStorage) Is(path string, hash string) bool {
	r.hashMu.RLock()
	defer r.hashMu.RUnlock()
	return r.hashes[path] == hash
}

func (r *ObjectStorage) SetHash(path string, hash string) {
	r.hashMu.Lock()
	defer r.hashMu.Unlock()
	r.hashes[path] = hash
}

func (r *ObjectStorage) Create(ctx context.Context) error {
	err := r.pod.Create(ctx, &corev1.Pod{
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: common.Ptr(int64(1)),
			Containers: []corev1.Container{
				{
					Name:            "minio",
					Image:           "minio/minio:RELEASE.2024-10-13T13-34-11Z",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Args:            []string{"server", "/data", "--console-address", ":9090"},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							TCPSocket: &corev1.TCPSocketAction{
								Port: intstr.FromInt32(9000),
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
	err = r.pod.CreateService(ctx, corev1.ServicePort{
		Name:       "api",
		Protocol:   "TCP",
		Port:       9000,
		TargetPort: intstr.FromInt32(9000),
	})
	if err != nil {
		return err
	}

	err = r.pod.WaitForContainerStarted(ctx)
	if err != nil {
		return err
	}

	r.localPort = GetFreePort()
	err = r.pod.Forward(ctx, 9000, r.localPort, true)
	if err != nil {
		fmt.Println("Forward error")
		return err
	}

	c, err := r.Client()
	if err != nil {
		fmt.Println("Creating client")
		return err
	}

	// Handle a case when port forwarder is not ready
	for i := 0; i < 10; i++ {
		makeBucketCtx, ctxCancel := context.WithTimeout(ctx, 2*time.Second)
		err = c.MakeBucket(makeBucketCtx, "devbox", minio2.MakeBucketOptions{})
		if err == nil {
			ctxCancel()
			return nil
		}
		if ctx.Err() != nil {
			ctxCancel()
			return ctx.Err()
		}
		ctxCancel()
	}
	return nil
}

func (r *ObjectStorage) Client() (*minio2.Client, error) {
	r.cachedClientMu.Lock()
	defer r.cachedClientMu.Unlock()
	if r.cachedClient != nil {
		return r.cachedClient, nil
	}
	connecter := minio.NewConnecter(
		fmt.Sprintf("localhost:%d", r.localPort),
		"minioadmin",
		"minioadmin",
		"",
		"",
		"devbox",
		log.DefaultLogger,
	)
	cl, err := connecter.GetClient()
	if err != nil {
		return nil, err
	}
	r.cachedClient = cl
	return cl, nil
}

func (r *ObjectStorage) WaitForReady(ctx context.Context) error {
	return r.pod.WaitForReady(ctx)
}

// TODO: Compress on-fly
func (r *ObjectStorage) Upload(ctx context.Context, path string, fsPath string, hash string) error {
	c, err := r.Client()
	if err != nil {
		return err
	}
	if hash != "" && r.Is(path, hash) {
		return nil
	}
	//putUrl, err := c.PresignedPutObject(ctx, "devbox", path, 15*time.Minute)
	putUrl, err := c.PresignHeader(ctx, "PUT", "devbox", path, 15*time.Minute, nil, http.Header{
		"X-Amz-Meta-Snowball-Auto-Extract": {"true"},
		"X-Amz-Meta-Minio-Snowball-Prefix": {filepath.Dir(path)},
		"Content-Type":                     {"application/gzip"},
		"Content-Encoding":                 {"gzip"},
	})
	if err != nil {
		return err
	}

	file, err := os.Open(fsPath)
	if err != nil {
		return err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	tarStream := artifacts.NewTarStream()
	go func() {
		tarStream.Add(filepath.Base(path), file, stat)
		tarStream.Close()
	}()
	io.Copy(buf, tarStream)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, putUrl.String(), buf)
	if err != nil {
		return err
	}
	req.ContentLength = int64(buf.Len())
	req.Header.Set("X-Amz-Meta-Snowball-Auto-Extract", "true")
	req.Header.Set("X-Amz-Meta-Minio-Snowball-Prefix", filepath.Dir(path))
	req.Header.Set("Content-Type", "application/gzip")
	req.Header.Set("Content-Encoding", "gzip")

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{Transport: tr}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed saving file: status code: %d / message: %s", res.StatusCode, string(b))
	}
	r.SetHash(path, hash)
	return nil
}
