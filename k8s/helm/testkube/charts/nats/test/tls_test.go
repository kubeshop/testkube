package test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestConfigTls(t *testing.T) {
	t.Parallel()
	test := DefaultTest()
	test.Values = `
config:
  cluster:
    enabled: true
    tls:
      enabled: true
      secretName: cluster-tls
  nats:
    tls:
      enabled: true
      secretName: nats-tls
      merge:
        ca_file: /etc/my-ca/ca.crt
        verify_cert_and_check_known_urls: true
      patch: [{op: add, path: /verify_and_map, value: true}]
  leafnodes:
    enabled: true
    tls:
      enabled: true
      secretName: leafnodes-tls
  websocket:
    enabled: true
    tls:
      enabled: true
      secretName: websocket-tls
  mqtt:
    enabled: true
    tls:
      enabled: true
      secretName: mqtt-tls
  gateway:
    enabled: true
    tls:
      enabled: true
      secretName: gateway-tls
  monitor:
    tls:
      enabled: true
`
	expected := DefaultResources(t, test)
	expected.Conf.Value["cluster"] = map[string]any{
		"name":         "nats",
		"no_advertise": true,
		"port":         int64(6222),
		"routes": []any{
			"tls://nats-0.nats-headless:6222",
			"tls://nats-1.nats-headless:6222",
			"tls://nats-2.nats-headless:6222",
		},
	}
	expected.Conf.Value["leafnodes"] = map[string]any{
		"port":         int64(7422),
		"no_advertise": true,
	}
	expected.Conf.Value["websocket"] = map[string]any{
		"port": int64(8080),
	}
	expected.Conf.Value["mqtt"] = map[string]any{
		"port": int64(1883),
	}
	expected.Conf.Value["gateway"] = map[string]any{
		"port": int64(7222),
		"name": "nats",
	}
	expected.Conf.Value["https_port"] = expected.Conf.Value["http_port"]
	delete(expected.Conf.Value, "http_port")

	replicas3 := int32(3)
	expected.StatefulSet.Value.Spec.Replicas = &replicas3

	volumes := expected.StatefulSet.Value.Spec.Template.Spec.Volumes
	natsVm := expected.StatefulSet.Value.Spec.Template.Spec.Containers[0].VolumeMounts
	reloaderVm := expected.StatefulSet.Value.Spec.Template.Spec.Containers[1].VolumeMounts
	for _, protocol := range []string{"nats", "leafnodes", "websocket", "mqtt", "cluster", "gateway"} {
		tls := map[string]any{
			"cert_file": "/etc/nats-certs/" + protocol + "/tls.crt",
			"key_file":  "/etc/nats-certs/" + protocol + "/tls.key",
		}
		if protocol == "nats" {
			tls["ca_file"] = "/etc/my-ca/ca.crt"
			tls["verify_cert_and_check_known_urls"] = true
			tls["verify_and_map"] = true
			expected.Conf.Value["tls"] = tls
		} else {
			expected.Conf.Value[protocol].(map[string]any)["tls"] = tls
		}

		volumes = append(volumes, corev1.Volume{
			Name: protocol + "-tls",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: protocol + "-tls",
				},
			},
		})

		natsVm = append(natsVm, corev1.VolumeMount{
			MountPath: "/etc/nats-certs/" + protocol,
			Name:      protocol + "-tls",
		})

		reloaderVm = append(reloaderVm, corev1.VolumeMount{
			MountPath: "/etc/nats-certs/" + protocol,
			Name:      protocol + "-tls",
		})
	}

	expected.StatefulSet.Value.Spec.Template.Spec.Containers[0].StartupProbe.HTTPGet.Scheme = corev1.URISchemeHTTPS
	expected.StatefulSet.Value.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Scheme = corev1.URISchemeHTTPS
	expected.StatefulSet.Value.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Scheme = corev1.URISchemeHTTPS

	expected.StatefulSet.Value.Spec.Template.Spec.Volumes = volumes
	expected.StatefulSet.Value.Spec.Template.Spec.Containers[0].VolumeMounts = natsVm
	expected.StatefulSet.Value.Spec.Template.Spec.Containers[1].VolumeMounts = reloaderVm

	// reloader certs are alphabetized
	reloaderArgs := expected.StatefulSet.Value.Spec.Template.Spec.Containers[1].Args
	for _, protocol := range []string{"cluster", "gateway", "leafnodes", "mqtt", "nats", "websocket"} {
		if protocol == "nats" {
			reloaderArgs = append(reloaderArgs, "-config", "/etc/my-ca/ca.crt")
		}
		reloaderArgs = append(reloaderArgs, "-config", "/etc/nats-certs/"+protocol+"/tls.crt", "-config", "/etc/nats-certs/"+protocol+"/tls.key")
	}

	expected.StatefulSet.Value.Spec.Template.Spec.Containers[1].Args = reloaderArgs

	expected.StatefulSet.Value.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{
		{
			Name:          "nats",
			ContainerPort: 4222,
		},
		{
			Name:          "leafnodes",
			ContainerPort: 7422,
		},
		{
			Name:          "websocket",
			ContainerPort: 8080,
		},
		{
			Name:          "mqtt",
			ContainerPort: 1883,
		},
		{
			Name:          "cluster",
			ContainerPort: 6222,
		},
		{
			Name:          "gateway",
			ContainerPort: 7222,
		},
		{
			Name:          "monitor",
			ContainerPort: 8222,
		},
	}

	expected.HeadlessService.Value.Spec.Ports = []corev1.ServicePort{
		{
			Name:        "nats",
			Port:        4222,
			TargetPort:  intstr.FromString("nats"),
			AppProtocol: &appProtocolTLS,
		},
		{
			Name:        "leafnodes",
			Port:        7422,
			TargetPort:  intstr.FromString("leafnodes"),
			AppProtocol: &appProtocolTLS,
		},
		{
			Name:        "websocket",
			Port:        8080,
			TargetPort:  intstr.FromString("websocket"),
			AppProtocol: &appProtocolHTTPS,
		},
		{
			Name:        "mqtt",
			Port:        1883,
			TargetPort:  intstr.FromString("mqtt"),
			AppProtocol: &appProtocolTLS,
		},
		{
			Name:        "cluster",
			Port:        6222,
			TargetPort:  intstr.FromString("cluster"),
			AppProtocol: &appProtocolTLS,
		},
		{
			Name:        "gateway",
			Port:        7222,
			TargetPort:  intstr.FromString("gateway"),
			AppProtocol: &appProtocolTLS,
		},
		{
			Name:        "monitor",
			Port:        8222,
			TargetPort:  intstr.FromString("monitor"),
			AppProtocol: &appProtocolHTTPS,
		},
	}

	expected.Service.Value.Spec.Ports = []corev1.ServicePort{
		{
			Name:        "nats",
			Port:        4222,
			TargetPort:  intstr.FromString("nats"),
			AppProtocol: &appProtocolTLS,
		},
		{
			Name:        "leafnodes",
			Port:        7422,
			TargetPort:  intstr.FromString("leafnodes"),
			AppProtocol: &appProtocolTLS,
		},
		{
			Name:        "websocket",
			Port:        8080,
			TargetPort:  intstr.FromString("websocket"),
			AppProtocol: &appProtocolHTTPS,
		},
		{
			Name:        "mqtt",
			Port:        1883,
			TargetPort:  intstr.FromString("mqtt"),
			AppProtocol: &appProtocolTLS,
		},
	}

	RenderAndCheck(t, test, expected)
}

func TestTlsCA(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		key    string
		dir    string
		secret bool
	}{
		{
			name:   "ConfigMap",
			secret: false,
		},
		{
			name:   "Secret",
			secret: true,
			key:    "my-ca.crt",
			dir:    "/etc/nats-ca-cert-custom",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			test := DefaultTest()
			test.Values = `
config:
  nats:
    tls:
      enabled: true
      secretName: nats-tls
tlsCA:
  enabled: true`
			if tt.secret {
				test.Values += `
  secretName: nats-ca`
			} else {
				test.Values += `
  configMapName: nats-ca`
			}
			if tt.key != "" {
				test.Values += `
  key: ` + tt.key
			}
			if tt.dir != "" {
				test.Values += `
  dir: ` + tt.dir
			}
			expected := DefaultResources(t, test)

			key := tt.key
			if key == "" {
				key = "ca.crt"
			}
			dir := tt.dir
			if dir == "" {
				dir = "/etc/nats-ca-cert"
			}
			expected.Conf.Value["tls"] = map[string]any{
				"cert_file": "/etc/nats-certs/nats/tls.crt",
				"key_file":  "/etc/nats-certs/nats/tls.key",
				"ca_file":   dir + "/" + key,
			}

			expected.NatsBoxContextsSecret.Value.StringData["default.json"] = `{
  "ca": "` + dir + "/" + key + `",
  "url": "nats://` + test.FullName + `"
}
`
			expected.Service.Value.Spec.Ports[0].AppProtocol = &appProtocolTLS
			expected.HeadlessService.Value.Spec.Ports[0].AppProtocol = &appProtocolTLS

			// reloader certs are alphabetized
			reloaderArgs := expected.StatefulSet.Value.Spec.Template.Spec.Containers[1].Args
			reloaderArgs = append(reloaderArgs,
				"-config", dir+"/"+key,
				"-config", "/etc/nats-certs/nats/tls.crt",
				"-config", "/etc/nats-certs/nats/tls.key")
			expected.StatefulSet.Value.Spec.Template.Spec.Containers[1].Args = reloaderArgs

			tlsCAVol := corev1.Volume{
				Name: "tls-ca",
			}
			if tt.secret {
				tlsCAVol.VolumeSource = corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "nats-ca",
					},
				}
			} else {
				tlsCAVol.VolumeSource = corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "nats-ca",
						},
					},
				}
			}

			tlsCAVm := corev1.VolumeMount{
				Name:      "tls-ca",
				MountPath: dir,
			}

			stsVols := expected.StatefulSet.Value.Spec.Template.Spec.Volumes
			natsVm := expected.StatefulSet.Value.Spec.Template.Spec.Containers[0].VolumeMounts
			reloaderVm := expected.StatefulSet.Value.Spec.Template.Spec.Containers[1].VolumeMounts

			stsVols = append(stsVols, tlsCAVol, corev1.Volume{
				Name: "nats-tls",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "nats-tls",
					},
				},
			})

			natsVm = append(natsVm, tlsCAVm, corev1.VolumeMount{
				MountPath: "/etc/nats-certs/nats",
				Name:      "nats-tls",
			})

			reloaderVm = append(reloaderVm, tlsCAVm, corev1.VolumeMount{
				MountPath: "/etc/nats-certs/nats",
				Name:      "nats-tls",
			})

			expected.StatefulSet.Value.Spec.Template.Spec.Volumes = stsVols
			expected.StatefulSet.Value.Spec.Template.Spec.Containers[0].VolumeMounts = natsVm
			expected.StatefulSet.Value.Spec.Template.Spec.Containers[1].VolumeMounts = reloaderVm

			natsBoxVols := expected.NatsBoxDeployment.Value.Spec.Template.Spec.Volumes
			natsBoxVms := expected.NatsBoxDeployment.Value.Spec.Template.Spec.Containers[0].VolumeMounts

			natsBoxVols = append(natsBoxVols, tlsCAVol)
			natsBoxVms = append(natsBoxVms, tlsCAVm)

			expected.NatsBoxDeployment.Value.Spec.Template.Spec.Volumes = natsBoxVols
			expected.NatsBoxDeployment.Value.Spec.Template.Spec.Containers[0].VolumeMounts = natsBoxVms

			RenderAndCheck(t, test, expected)
		})
	}
}
