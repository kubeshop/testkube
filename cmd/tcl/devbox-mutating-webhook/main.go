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
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/wI2L/jsondiff"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

func main() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/mutate", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "invalid content type", http.StatusBadRequest)
			return
		}

		initImage := os.Args[1]
		toolkitImage := os.Args[2]

		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		body := buf.Bytes()

		if len(body) == 0 {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		var review admissionv1.AdmissionReview
		if err := json.Unmarshal(body, &review); err != nil {
			http.Error(w, fmt.Sprintf("invalid request: %s", err), http.StatusBadRequest)
			return
		}

		if review.Request == nil {
			http.Error(w, "invalid request: empty", http.StatusBadRequest)
			return
		}
		if review.Request.Kind.Kind != "Pod" {
			http.Error(w, fmt.Sprintf("invalid resource: %s", review.Request.Kind.Kind), http.StatusBadRequest)
			return
		}

		pod := corev1.Pod{}
		if err := json.Unmarshal(review.Request.Object.Raw, &pod); err != nil {
			http.Error(w, fmt.Sprintf("invalid pod provided: %s", err), http.StatusBadRequest)
			return
		}
		originalPod := pod.DeepCopy()

		// Apply changes
		if pod.Labels[constants.ResourceIdLabelName] != "" {
			usesToolkit := false
			for _, c := range append(pod.Spec.InitContainers, pod.Spec.Containers...) {
				if c.Image == toolkitImage {
					usesToolkit = true
				}
			}

			pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
				Name:         "devbox",
				VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
			})

			script := `
				set -e
				/.tktw-bin/wget -O /.tk-devbox/init http://devbox-binary:8080/init || exit 1
				chmod 777 /.tk-devbox/init
				ls -lah /.tk-devbox`
			if usesToolkit {
				script = `
					set -e
					/.tktw-bin/wget -O /.tk-devbox/init http://devbox-binary:8080/init || exit 1
					/.tktw-bin/wget -O /.tk-devbox/toolkit http://devbox-binary:8080/toolkit || exit 1
					chmod 777 /.tk-devbox/init
					chmod 777 /.tk-devbox/toolkit
					ls -lah /.tk-devbox`
			}

			pod.Spec.InitContainers = append([]corev1.Container{{
				Name:            "devbox-init",
				Image:           initImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/bin/sh", "-c"},
				Args:            []string{script},
			}}, pod.Spec.InitContainers...)

			// TODO: Handle it better, to not be ambiguous
			pod.Annotations[constants.SpecAnnotationName] = strings.ReplaceAll(pod.Annotations[constants.SpecAnnotationName], "\"/toolkit\"", "\"/.tk-devbox/toolkit\"")
			pod.Annotations[constants.SpecAnnotationName] = strings.ReplaceAll(pod.Annotations[constants.SpecAnnotationName], "\"/.tktw/toolkit\"", "\"/.tk-devbox/toolkit\"")

			for i := range pod.Spec.InitContainers {
				if (pod.Spec.InitContainers[i].Image == toolkitImage || pod.Spec.InitContainers[i].Image == initImage) && pod.Spec.InitContainers[i].Command[0] == "/init" {
					pod.Spec.InitContainers[i].Command[0] = "/.tk-devbox/init"
				}
				if pod.Spec.InitContainers[i].Command[0] == "/.tktw/init" {
					pod.Spec.InitContainers[i].Command[0] = "/.tk-devbox/init"
				}
			}
			for i := range pod.Spec.Containers {
				if (pod.Spec.Containers[i].Image == toolkitImage || pod.Spec.Containers[i].Image == initImage) && pod.Spec.Containers[i].Command[0] == "/init" {
					pod.Spec.Containers[i].Command[0] = "/.tk-devbox/init"
				}
				if pod.Spec.Containers[i].Command[0] == "/.tktw/init" {
					pod.Spec.Containers[i].Command[0] = "/.tk-devbox/init"
				}
			}

			for i := range pod.Spec.InitContainers {
				pod.Spec.InitContainers[i].VolumeMounts = append(pod.Spec.InitContainers[i].VolumeMounts, corev1.VolumeMount{
					Name:      "devbox",
					MountPath: "/.tk-devbox",
				})
			}
			for i := range pod.Spec.Containers {
				pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
					Name:      "devbox",
					MountPath: "/.tk-devbox",
				})
			}
		}

		patch, err := jsondiff.Compare(originalPod, pod)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to build patch for changes: %s", err), http.StatusInternalServerError)
			return
		}

		serializedPatch, err := json.Marshal(patch)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to serialize patch for changes: %s", err), http.StatusInternalServerError)
			return
		}

		review.Response = &admissionv1.AdmissionResponse{
			UID:       review.Request.UID,
			Allowed:   true,
			PatchType: common.Ptr(admissionv1.PatchTypeJSONPatch),
			Patch:     serializedPatch,
		}

		serializedResponse, err := json.Marshal(review)
		if err != nil {
			http.Error(w, fmt.Sprintf("cannot marshal result: %s", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "%s", serializedResponse)
	})

	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stopSignal
		os.Exit(0)
	}()

	fmt.Println("Starting server...")

	panic(http.ListenAndServeTLS(":8443", "/certs/tls.crt", "/certs/tls.key", nil))
}
