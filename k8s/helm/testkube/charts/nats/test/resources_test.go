package test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestResourceOptions(t *testing.T) {
	t.Parallel()
	test := DefaultTest()
	test.Values = `
global:
  image:
    pullPolicy: Always
    pullSecretNames:
    - testPullSecret
    registry: docker.io
  labels:
    global: global
namespaceOverride: foo
config:
  jetstream:
    enabled: true
  websocket:
    enabled: true
    ingress:
      enabled: true
      hosts:
      - demo.nats.io
      tlsSecretName: ws-tls
container:
  image:
    pullPolicy: IfNotPresent
    registry: gcr.io
  env:
    GOMEMLIMIT: 1GiB
    TOKEN:
      valueFrom:
        secretKeyRef:
          name: token
          key: token
reloader:
  env:
    GOMEMLIMIT: 1GiB
    TOKEN:
      valueFrom:
        secretKeyRef:
          name: token
          key: token
  natsVolumeMountPrefixes:
  - /etc/
  - /data
promExporter:
  enabled: true
  port: 7778
  env:
    GOMEMLIMIT: 1GiB
    TOKEN:
      valueFrom:
        secretKeyRef:
          name: token
          key: token
podTemplate:
  configChecksumAnnotation: false
  topologySpreadConstraints:
    kubernetes.io/hostname:
      maxSkew: 1
natsBox:
  contexts:
    loadedSecret:
      creds:
        secretName: loaded-creds
        key: nats.creds
      nkey:
        secretName: loaded-nkey
        key: nats.nk
      tls:
        secretName: loaded-tls
      merge:
        ca: /etc/my-ca/ca.crt
    loadedContents:
      creds:
        contents: aabbcc
      nkey:
        contents: ddeeff
    token:
      merge:
        token: foo
  container:
    env:
      GOMEMLIMIT: 1GiB
      TOKEN:
        valueFrom:
          secretKeyRef:
            name: token
            key: token
`
	expected := DefaultResources(t, test)

	expected.Conf.Value["jetstream"] = map[string]any{
		"max_file_store":   int64(10737418240),
		"max_memory_store": int64(0),
		"store_dir":        "/data",
	}

	expected.Conf.Value["websocket"] = map[string]any{
		"port":   int64(8080),
		"no_tls": true,
	}

	env := []corev1.EnvVar{
		{
			Name:  "GOMEMLIMIT",
			Value: "1GiB",
		},
		{
			Name: "TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "token",
					},
					Key: "token",
				},
			},
		},
	}

	expected.StatefulSet.Value.ObjectMeta.Labels["global"] = "global"
	expected.StatefulSet.Value.ObjectMeta.Namespace = "foo"
	expected.StatefulSet.Value.Spec.Template.ObjectMeta.Labels["global"] = "global"
	expected.StatefulSet.Value.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
		{
			Name: "testPullSecret",
		},
	}

	dd := ddg.Get(t)
	ctr := expected.StatefulSet.Value.Spec.Template.Spec.Containers

	// nats
	ctr[0].Env = append(ctr[0].Env, env...)
	ctr[0].Image = "gcr.io/" + ctr[0].Image
	ctr[0].ImagePullPolicy = "IfNotPresent"
	ctr[0].VolumeMounts = append(ctr[0].VolumeMounts, corev1.VolumeMount{
		Name:      test.FullName + "-js",
		MountPath: "/data",
	})

	// reloader
	ctr[1].Env = env
	ctr[1].Image = "docker.io/" + ctr[1].Image
	ctr[1].ImagePullPolicy = "Always"
	ctr[1].VolumeMounts = append(ctr[1].VolumeMounts, corev1.VolumeMount{
		Name:      test.FullName + "-js",
		MountPath: "/data",
	})

	// promExporter
	ctr = append(ctr, corev1.Container{
		Args: []string{
			"-port=7778",
			"-connz",
			"-routez",
			"-subz",
			"-varz",
			"-prefix=nats",
			"-use_internal_server_id",
			"-jsz=all",
			"http://localhost:8222/",
		},
		Env:             env,
		Image:           "docker.io/" + dd.PromExporterImage,
		ImagePullPolicy: "Always",
		Name:            "prom-exporter",
		Ports: []corev1.ContainerPort{
			{
				Name:          "prom-metrics",
				ContainerPort: 7778,
			},
		},
	})

	expected.StatefulSet.Value.Spec.Template.Spec.Containers = ctr
	expected.StatefulSet.Value.Spec.Template.Spec.TopologySpreadConstraints = []corev1.TopologySpreadConstraint{
		{
			MaxSkew:       1,
			TopologyKey:   "kubernetes.io/hostname",
			LabelSelector: expected.StatefulSet.Value.Spec.Selector,
		},
	}

	expected.NatsBoxDeployment.Value.ObjectMeta.Labels["global"] = "global"
	expected.NatsBoxDeployment.Value.ObjectMeta.Namespace = "foo"
	expected.NatsBoxDeployment.Value.Spec.Template.ObjectMeta.Labels["global"] = "global"
	expected.NatsBoxDeployment.Value.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
		{
			Name: "testPullSecret",
		},
	}

	nbCtr := expected.NatsBoxDeployment.Value.Spec.Template.Spec.Containers[0]
	// nats-box
	nbCtr.Env = env
	nbCtr.Image = "docker.io/" + nbCtr.Image
	nbCtr.ImagePullPolicy = "Always"
	nbCtr.VolumeMounts = append(nbCtr.VolumeMounts,
		corev1.VolumeMount{
			MountPath: "/etc/nats-contents",
			Name:      "contents",
		},
		corev1.VolumeMount{
			Name:      "ctx-loadedSecret-creds",
			MountPath: "/etc/nats-creds/loadedSecret",
		},
		corev1.VolumeMount{
			Name:      "ctx-loadedSecret-nkey",
			MountPath: "/etc/nats-nkeys/loadedSecret",
		},
		corev1.VolumeMount{
			Name:      "ctx-loadedSecret-tls",
			MountPath: "/etc/nats-certs/loadedSecret",
		},
	)
	expected.NatsBoxDeployment.Value.Spec.Template.Spec.Containers[0] = nbCtr

	nbVol := expected.NatsBoxDeployment.Value.Spec.Template.Spec.Volumes
	nbVol = append(nbVol,
		corev1.Volume{
			Name: "contents",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "nats-box-contents",
				},
			},
		},
		corev1.Volume{
			Name: "ctx-loadedSecret-creds",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "loaded-creds",
				},
			},
		},
		corev1.Volume{
			Name: "ctx-loadedSecret-nkey",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "loaded-nkey",
				},
			},
		},
		corev1.Volume{
			Name: "ctx-loadedSecret-tls",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "loaded-tls",
				},
			},
		},
	)
	expected.NatsBoxDeployment.Value.Spec.Template.Spec.Volumes = nbVol

	expected.NatsBoxContextsSecret.Value.ObjectMeta.Labels["global"] = "global"
	expected.NatsBoxContextsSecret.Value.ObjectMeta.Namespace = "foo"
	expected.NatsBoxContextsSecret.Value.StringData["loadedSecret.json"] = `{
  "ca": "/etc/my-ca/ca.crt",
  "cert": "/etc/nats-certs/loadedSecret/tls.crt",
  "creds": "/etc/nats-creds/loadedSecret/nats.creds",
  "key": "/etc/nats-certs/loadedSecret/tls.key",
  "nkey": "/etc/nats-nkeys/loadedSecret/nats.nk",
  "url": "nats://` + test.FullName + `"
}
`
	expected.NatsBoxContextsSecret.Value.StringData["loadedContents.json"] = `{
  "creds": "/etc/nats-contents/loadedContents.creds",
  "nkey": "/etc/nats-contents/loadedContents.nk",
  "url": "nats://` + test.FullName + `"
}
`
	expected.NatsBoxContextsSecret.Value.StringData["token.json"] = `{
  "token": "foo",
  "url": "nats://` + test.FullName + `"
}
`

	expected.NatsBoxContentsSecret.HasValue = true
	expected.NatsBoxContentsSecret.Value.ObjectMeta.Labels["global"] = "global"
	expected.NatsBoxContentsSecret.Value.ObjectMeta.Namespace = "foo"
	expected.NatsBoxContentsSecret.Value.StringData = map[string]string{
		"loadedContents.creds": "aabbcc",
		"loadedContents.nk":    "ddeeff",
	}

	expected.Ingress.HasValue = true
	expected.Ingress.Value.ObjectMeta.Labels["global"] = "global"
	expected.Ingress.Value.ObjectMeta.Namespace = "foo"
	expected.Ingress.Value.Spec.TLS = []networkingv1.IngressTLS{
		{
			Hosts:      []string{"demo.nats.io"},
			SecretName: "ws-tls",
		},
	}

	resource10Gi, _ := resource.ParseQuantity("10Gi")
	expected.StatefulSet.Value.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
		{
			ObjectMeta: v1.ObjectMeta{
				Name: test.FullName + "-js",
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					"ReadWriteOnce",
				},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						"storage": resource10Gi,
					},
				},
			},
		},
	}

	expected.StatefulSet.Value.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{
		{
			Name:          "nats",
			ContainerPort: 4222,
		},
		{
			Name:          "websocket",
			ContainerPort: 8080,
		},
		{
			Name:          "monitor",
			ContainerPort: 8222,
		},
	}

	expected.HeadlessService.Value.ObjectMeta.Labels["global"] = "global"
	expected.HeadlessService.Value.ObjectMeta.Namespace = "foo"
	expected.HeadlessService.Value.Spec.Ports = []corev1.ServicePort{
		{
			Name:        "nats",
			Port:        4222,
			TargetPort:  intstr.FromString("nats"),
			AppProtocol: &appProtocolTCP,
		},
		{
			Name:        "websocket",
			Port:        8080,
			TargetPort:  intstr.FromString("websocket"),
			AppProtocol: &appProtocolHTTP,
		},
		{
			Name:        "monitor",
			Port:        8222,
			TargetPort:  intstr.FromString("monitor"),
			AppProtocol: &appProtocolHTTP,
		},
	}

	expected.Service.Value.ObjectMeta.Labels["global"] = "global"
	expected.Service.Value.ObjectMeta.Namespace = "foo"
	expected.Service.Value.Spec.Ports = []corev1.ServicePort{
		{
			Name:        "nats",
			Port:        4222,
			TargetPort:  intstr.FromString("nats"),
			AppProtocol: &appProtocolTCP,
		},
		{
			Name:        "websocket",
			Port:        8080,
			TargetPort:  intstr.FromString("websocket"),
			AppProtocol: &appProtocolHTTP,
		},
	}

	expected.ConfigMap.Value.ObjectMeta.Labels["global"] = "global"
	expected.ConfigMap.Value.ObjectMeta.Namespace = "foo"

	RenderAndCheck(t, test, expected)
}

func TestResourcesMergePatch(t *testing.T) {
	t.Parallel()
	test := DefaultTest()
	test.Values = `
config:
  websocket:
    enabled: true
    ingress:
      enabled: true
      hosts:
      - demo.nats.io
      merge:
        metadata:
          annotations:
            test: test
      patch: [{op: add, path: /metadata/labels/test, value: "test"}]
container:
  merge:
    stdin: true
  patch: [{op: add, path: /tty, value: true}]
reloader:
  merge:
    stdin: true
  patch: [{op: add, path: /tty, value: true}]
promExporter:
  enabled: true
  merge:
    stdin: true
  patch: [{op: add, path: /tty, value: true}]
  podMonitor:
    enabled: true
    merge:
      metadata:
        annotations:
          test: test
    patch: [{op: add, path: /metadata/labels/test, value: "test"}]
service:
  enabled: true
  merge:
    metadata:
      annotations:
        test: test
  patch: [{op: add, path: /metadata/labels/test, value: "test"}]
statefulSet:
  merge:
    metadata:
      annotations:
        test: test
  patch: [{op: add, path: /metadata/labels/test, value: "test"}]
podTemplate:
  merge:
    metadata:
      annotations:
        test: test
  patch: [{op: add, path: /metadata/labels/test, value: "test"}]
headlessService:
  merge:
    metadata:
      annotations:
        test: test
  patch: [{op: add, path: /metadata/labels/test, value: "test"}]
configMap:
  merge:
    metadata:
      annotations:
        test: test
  patch: [{op: add, path: /metadata/labels/test, value: "test"}]
podDisruptionBudget:
  merge:
    metadata:
      annotations:
        test: test
  patch: [{op: add, path: /metadata/labels/test, value: "test"}]
serviceAccount:
  enabled: true
  merge:
    metadata:
      annotations:
        test: test
  patch: [{op: add, path: /metadata/labels/test, value: "test"}]
natsBox:
  contexts:
    default:
      merge:
        user: foo
      patch: [{op: add, path: /password, value: "bar"}]
  container:
    merge:
      stdin: true
    patch: [{op: add, path: /tty, value: true}]
  podTemplate:
    merge:
      metadata:
        annotations:
          test: test
    patch: [{op: add, path: /metadata/labels/test, value: "test"}]
  deployment:
    merge:
      metadata:
        annotations:
          test: test
    patch: [{op: add, path: /metadata/labels/test, value: "test"}]
  contextsSecret:
    merge:
      metadata:
        annotations:
          test: test
    patch: [{op: add, path: /metadata/labels/test, value: "test"}]
  contentsSecret:
    merge:
      metadata:
        annotations:
          test: test
    patch: [{op: add, path: /metadata/labels/test, value: "test"}]
  serviceAccount:
    enabled: true
    merge:
      metadata:
        annotations:
          test: test
    patch: [{op: add, path: /metadata/labels/test, value: "test"}]
`
	expected := DefaultResources(t, test)

	expected.Conf.Value["websocket"] = map[string]any{
		"port":   int64(8080),
		"no_tls": true,
	}

	annotations := func() map[string]string {
		return map[string]string{
			"test": "test",
		}
	}

	dd := ddg.Get(t)
	ctr := expected.StatefulSet.Value.Spec.Template.Spec.Containers
	ctr[0].Stdin = true
	ctr[0].TTY = true
	ctr[1].Stdin = true
	ctr[1].TTY = true
	ctr = append(ctr, corev1.Container{
		Args: []string{
			"-port=7777",
			"-connz",
			"-routez",
			"-subz",
			"-varz",
			"-prefix=nats",
			"-use_internal_server_id",
			"http://localhost:8222/",
		},
		Image: dd.PromExporterImage,
		Name:  "prom-exporter",
		Ports: []corev1.ContainerPort{
			{
				Name:          "prom-metrics",
				ContainerPort: 7777,
			},
		},
		Stdin: true,
		TTY:   true,
	})
	expected.StatefulSet.Value.Spec.Template.Spec.Containers = ctr
	expected.StatefulSet.Value.Spec.Template.Spec.ServiceAccountName = test.FullName

	expected.NatsBoxDeployment.Value.Spec.Template.Spec.Containers[0].Stdin = true
	expected.NatsBoxDeployment.Value.Spec.Template.Spec.Containers[0].TTY = true

	expected.StatefulSet.Value.ObjectMeta.Annotations = annotations()
	expected.StatefulSet.Value.ObjectMeta.Labels["test"] = "test"

	expected.StatefulSet.Value.Spec.Template.ObjectMeta.Annotations = annotations()
	expected.StatefulSet.Value.Spec.Template.ObjectMeta.Labels["test"] = "test"

	expected.NatsBoxDeployment.Value.ObjectMeta.Annotations = annotations()
	expected.NatsBoxDeployment.Value.ObjectMeta.Labels["test"] = "test"

	expected.NatsBoxDeployment.Value.Spec.Template.ObjectMeta.Annotations = annotations()
	expected.NatsBoxDeployment.Value.Spec.Template.ObjectMeta.Labels["test"] = "test"
	expected.NatsBoxDeployment.Value.Spec.Template.Spec.ServiceAccountName = test.FullName + "-box"

	expected.PodMonitor.HasValue = true
	expected.PodMonitor.Value.ObjectMeta.Annotations = annotations()
	expected.PodMonitor.Value.ObjectMeta.Labels["test"] = "test"

	expected.Ingress.HasValue = true
	expected.Ingress.Value.ObjectMeta.Annotations = annotations()
	expected.Ingress.Value.ObjectMeta.Labels["test"] = "test"

	expected.NatsBoxContextsSecret.Value.ObjectMeta.Annotations = annotations()
	expected.NatsBoxContextsSecret.Value.ObjectMeta.Labels["test"] = "test"
	expected.NatsBoxContextsSecret.Value.StringData["default.json"] = `{
  "password": "bar",
  "url": "nats://` + test.FullName + `",
  "user": "foo"
}
`

	expected.NatsBoxContentsSecret.Value.ObjectMeta.Annotations = annotations()
	expected.NatsBoxContentsSecret.Value.ObjectMeta.Labels["test"] = "test"

	expected.NatsBoxServiceAccount.HasValue = true
	expected.NatsBoxServiceAccount.Value.ObjectMeta.Annotations = annotations()
	expected.NatsBoxServiceAccount.Value.ObjectMeta.Labels["test"] = "test"

	expected.Service.Value.ObjectMeta.Annotations = annotations()
	expected.Service.Value.ObjectMeta.Labels["test"] = "test"

	expected.PodDisruptionBudget.Value.ObjectMeta.Annotations = annotations()
	expected.PodDisruptionBudget.Value.ObjectMeta.Labels["test"] = "test"

	expected.ServiceAccount.HasValue = true
	expected.ServiceAccount.Value.ObjectMeta.Annotations = annotations()
	expected.ServiceAccount.Value.ObjectMeta.Labels["test"] = "test"

	expected.HeadlessService.Value.ObjectMeta.Annotations = annotations()
	expected.HeadlessService.Value.ObjectMeta.Labels["test"] = "test"

	expected.ConfigMap.Value.ObjectMeta.Annotations = annotations()
	expected.ConfigMap.Value.ObjectMeta.Labels["test"] = "test"

	expected.StatefulSet.Value.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{
		{
			Name:          "nats",
			ContainerPort: 4222,
		},
		{
			Name:          "websocket",
			ContainerPort: 8080,
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
			AppProtocol: &appProtocolTCP,
		},
		{
			Name:        "websocket",
			Port:        8080,
			TargetPort:  intstr.FromString("websocket"),
			AppProtocol: &appProtocolHTTP,
		},
		{
			Name:        "monitor",
			Port:        8222,
			TargetPort:  intstr.FromString("monitor"),
			AppProtocol: &appProtocolHTTP,
		},
	}

	expected.Service.Value.Spec.Ports = []corev1.ServicePort{
		{
			Name:        "nats",
			Port:        4222,
			TargetPort:  intstr.FromString("nats"),
			AppProtocol: &appProtocolTCP,
		},
		{
			Name:        "websocket",
			Port:        8080,
			TargetPort:  intstr.FromString("websocket"),
			AppProtocol: &appProtocolHTTP,
		},
	}

	RenderAndCheck(t, test, expected)
}
