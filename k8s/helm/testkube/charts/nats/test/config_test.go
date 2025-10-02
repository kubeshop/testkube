package test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestConfigDisable(t *testing.T) {
	t.Parallel()
	test := DefaultTest()
	test.Values = `
config:
  monitor:
    enabled: false
`
	expected := DefaultResources(t, test)
	delete(expected.Conf.Value, "http_port")

	expected.StatefulSet.Value.Spec.Template.Spec.Containers[0].LivenessProbe = nil
	expected.StatefulSet.Value.Spec.Template.Spec.Containers[0].ReadinessProbe = nil
	expected.StatefulSet.Value.Spec.Template.Spec.Containers[0].StartupProbe = nil

	cp := expected.StatefulSet.Value.Spec.Template.Spec.Containers[0].Ports
	expected.StatefulSet.Value.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{cp[0]}

	hsp := expected.HeadlessService.Value.Spec.Ports
	expected.HeadlessService.Value.Spec.Ports = []corev1.ServicePort{hsp[0]}

	RenderAndCheck(t, test, expected)
}

func TestConfigJetStreamCluster(t *testing.T) {
	t.Parallel()
	test := DefaultTest()
	test.Values = `
config:
  cluster:
    enabled: true
  jetstream:
    enabled: true
`
	expected := DefaultResources(t, test)

	expected.Conf.Value["cluster"] = map[string]any{
		"name":         "nats",
		"no_advertise": true,
		"port":         int64(6222),
		"routes": []any{
			"nats://nats-0.nats-headless:6222",
			"nats://nats-1.nats-headless:6222",
			"nats://nats-2.nats-headless:6222",
		},
	}
	expected.Conf.Value["jetstream"] = map[string]any{
		"max_file_store":   int64(10737418240),
		"max_memory_store": int64(0),
		"store_dir":        "/data",
	}

	replicas3 := int32(3)
	expected.StatefulSet.Value.Spec.Replicas = &replicas3

	vm := expected.StatefulSet.Value.Spec.Template.Spec.Containers[0].VolumeMounts
	expected.StatefulSet.Value.Spec.Template.Spec.Containers[0].VolumeMounts = append(vm, corev1.VolumeMount{
		MountPath: "/data",
		Name:      test.FullName + "-js",
	})

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
			Name:          "cluster",
			ContainerPort: 6222,
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
			Name:        "cluster",
			Port:        6222,
			TargetPort:  intstr.FromString("cluster"),
			AppProtocol: &appProtocolTCP,
		},
		{
			Name:        "monitor",
			Port:        8222,
			TargetPort:  intstr.FromString("monitor"),
			AppProtocol: &appProtocolHTTP,
		},
	}

	RenderAndCheck(t, test, expected)
}

func TestConfigOptions(t *testing.T) {
	t.Parallel()
	test := DefaultTest()
	test.Values = `
config:
  jetstream:
    enabled: true
    fileStore:
      dir: /mnt
      pvc:
        size: 5Gi
        storageClassName: gp3
      maxSize: 1Gi
    memoryStore:
      enabled: true
      maxSize: 2Gi
  cluster:
    enabled: true
    replicas: 2
    routeURLs:
      user: foo
      password: bar
      useFQDN: true
      k8sClusterDomain: foo.bar.local
  resolver:
    enabled: true
    dir: /mnt/resolver
    pvc:
      size: 5Gi
      storageClassName: gp3
  serverNamePrefix: test_
`
	expected := DefaultResources(t, test)

	expected.Conf.Value["cluster"] = map[string]any{
		"authorization": map[string]any{
			"user":     "foo",
			"password": "bar",
		},
		"name":         "nats",
		"no_advertise": true,
		"port":         int64(6222),
		"routes": []any{
			"nats://foo:bar@nats-0.nats-headless.nats.svc.foo.bar.local:6222",
			"nats://foo:bar@nats-1.nats-headless.nats.svc.foo.bar.local:6222",
		},
	}
	expected.Conf.Value["jetstream"] = map[string]any{
		"max_file_store":   int64(1073741824),
		"max_memory_store": int64(2147483648),
		"store_dir":        "/mnt",
	}
	expected.Conf.Value["resolver"] = map[string]any{
		"dir": "/mnt/resolver",
	}

	replicas2 := int32(2)
	expected.StatefulSet.Value.Spec.Replicas = &replicas2

	resource5Gi, _ := resource.ParseQuantity("5Gi")
	storageClassGp3 := "gp3"
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
						"storage": resource5Gi,
					},
				},
				StorageClassName: &storageClassGp3,
			},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name: test.FullName + "-resolver",
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					"ReadWriteOnce",
				},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						"storage": resource5Gi,
					},
				},
				StorageClassName: &storageClassGp3,
			},
		},
	}

	ctr := &expected.StatefulSet.Value.Spec.Template.Spec.Containers[0]
	ctr.Env[1].Value = "test_$(POD_NAME)"

	ctr.VolumeMounts = append(ctr.VolumeMounts, corev1.VolumeMount{
		MountPath: "/mnt",
		Name:      test.FullName + "-js",
	}, corev1.VolumeMount{
		MountPath: "/mnt/resolver",
		Name:      test.FullName + "-resolver",
	})

	ctr.Ports = []corev1.ContainerPort{
		{
			Name:          "nats",
			ContainerPort: 4222,
		},
		{
			Name:          "cluster",
			ContainerPort: 6222,
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
			Name:        "cluster",
			Port:        6222,
			TargetPort:  intstr.FromString("cluster"),
			AppProtocol: &appProtocolTCP,
		},
		{
			Name:        "monitor",
			Port:        8222,
			TargetPort:  intstr.FromString("monitor"),
			AppProtocol: &appProtocolHTTP,
		},
	}

	RenderAndCheck(t, test, expected)
}

func TestConfigMergePatch(t *testing.T) {
	t.Parallel()
	test := DefaultTest()
	test.Values = `
config:
  merge:
    ping_interval: 5m
  patch: [{op: add, path: /ping_max, value: 3}]
  cluster:
    enabled: true
    merge:
      no_advertise: false
    patch: [{op: add, path: /advertise, value: "demo.nats.io:6222"}]
  jetstream:
    enabled: true
    merge:
      max_outstanding_catchup: "<< 64MB >>"
    patch: [{op: add, path: /max_file_store, value: "<< 1GB >>"}]
    fileStore:
      pvc:
        merge:
          spec:
            storageClassName: gp3
        patch: [{op: add, path: /spec/accessModes/-, value: ReadWriteMany}]
  leafnodes:
    enabled: true
    merge:
      no_advertise: false
    patch: [{op: add, path: /advertise, value: "demo.nats.io:7422"}]
  websocket:
    enabled: true
    merge:
      compression: true
    patch: [{op: add, path: /same_origin, value: true}]
  mqtt:
    enabled: true
    merge:
      ack_wait: 1m
    patch: [{op: add, path: /max_ack_pending, value: 100}]
  gateway:
    enabled: true
    merge:
      gateways:
      - name: nats
        url: nats://demo.nats.io:7222
    patch: [{op: add, path: /advertise, value: "demo.nats.io:7222"}]
  resolver:
    enabled: true
    merge:
      type: full
    patch: [{op: add, path: /allow_delete, value: true}]
    pvc:
      merge:
        spec:
          storageClassName: gp3
      patch: [{op: add, path: /spec/accessModes/-, value: ReadWriteMany}]
`
	expected := DefaultResources(t, test)
	expected.Conf.Value["ping_interval"] = "5m"
	expected.Conf.Value["ping_max"] = int64(3)
	expected.Conf.Value["cluster"] = map[string]any{
		"name":         "nats",
		"no_advertise": false,
		"advertise":    "demo.nats.io:6222",
		"port":         int64(6222),
		"routes": []any{
			"nats://nats-0.nats-headless:6222",
			"nats://nats-1.nats-headless:6222",
			"nats://nats-2.nats-headless:6222",
		},
	}
	expected.Conf.Value["jetstream"] = map[string]any{
		"max_memory_store":        int64(0),
		"store_dir":               "/data",
		"max_file_store":          int64(1073741824),
		"max_outstanding_catchup": int64(67108864),
	}
	expected.Conf.Value["leafnodes"] = map[string]any{
		"port":         int64(7422),
		"no_advertise": false,
		"advertise":    "demo.nats.io:7422",
	}
	expected.Conf.Value["websocket"] = map[string]any{
		"port":        int64(8080),
		"compression": true,
		"no_tls":      true,
		"same_origin": true,
	}
	expected.Conf.Value["mqtt"] = map[string]any{
		"port":            int64(1883),
		"ack_wait":        "1m",
		"max_ack_pending": int64(100),
	}
	expected.Conf.Value["gateway"] = map[string]any{
		"port":      int64(7222),
		"name":      "nats",
		"advertise": "demo.nats.io:7222",
		"gateways": []any{
			map[string]any{
				"name": "nats",
				"url":  "nats://demo.nats.io:7222",
			},
		},
	}
	expected.Conf.Value["resolver"] = map[string]any{
		"dir":          "/data/resolver",
		"type":         "full",
		"allow_delete": true,
	}

	replicas3 := int32(3)
	expected.StatefulSet.Value.Spec.Replicas = &replicas3

	vm := expected.StatefulSet.Value.Spec.Template.Spec.Containers[0].VolumeMounts
	expected.StatefulSet.Value.Spec.Template.Spec.Containers[0].VolumeMounts = append(vm, corev1.VolumeMount{
		MountPath: "/data",
		Name:      test.FullName + "-js",
	}, corev1.VolumeMount{
		MountPath: "/data/resolver",
		Name:      test.FullName + "-resolver",
	})

	resource1Gi, _ := resource.ParseQuantity("1Gi")
	resource10Gi, _ := resource.ParseQuantity("10Gi")
	storageClassGp3 := "gp3"
	expected.StatefulSet.Value.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
		{
			ObjectMeta: v1.ObjectMeta{
				Name: test.FullName + "-js",
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					"ReadWriteOnce",
					"ReadWriteMany",
				},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						"storage": resource10Gi,
					},
				},
				StorageClassName: &storageClassGp3,
			},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name: test.FullName + "-resolver",
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					"ReadWriteOnce",
					"ReadWriteMany",
				},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						"storage": resource1Gi,
					},
				},
				StorageClassName: &storageClassGp3,
			},
		},
	}

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
			AppProtocol: &appProtocolTCP,
		},
		{
			Name:        "leafnodes",
			Port:        7422,
			TargetPort:  intstr.FromString("leafnodes"),
			AppProtocol: &appProtocolTCP,
		},
		{
			Name:        "websocket",
			Port:        8080,
			TargetPort:  intstr.FromString("websocket"),
			AppProtocol: &appProtocolHTTP,
		},
		{
			Name:        "mqtt",
			Port:        1883,
			TargetPort:  intstr.FromString("mqtt"),
			AppProtocol: &appProtocolTCP,
		},
		{
			Name:        "cluster",
			Port:        6222,
			TargetPort:  intstr.FromString("cluster"),
			AppProtocol: &appProtocolTCP,
		},
		{
			Name:        "gateway",
			Port:        7222,
			TargetPort:  intstr.FromString("gateway"),
			AppProtocol: &appProtocolTCP,
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
			Name:        "leafnodes",
			Port:        7422,
			TargetPort:  intstr.FromString("leafnodes"),
			AppProtocol: &appProtocolTCP,
		},
		{
			Name:        "websocket",
			Port:        8080,
			TargetPort:  intstr.FromString("websocket"),
			AppProtocol: &appProtocolHTTP,
		},
		{
			Name:        "mqtt",
			Port:        1883,
			TargetPort:  intstr.FromString("mqtt"),
			AppProtocol: &appProtocolTCP,
		},
	}

	RenderAndCheck(t, test, expected)
}

func TestConfigInclude(t *testing.T) {
	t.Parallel()
	test := DefaultTest()
	test.Values = `
config:
  jetstream:
    enabled: true
    merge:
      zzz$include: "js.conf"
  merge:
    $include: "my-config.conf"
    zzz$include: "my-config-last.conf"
configMap:
  merge:
    data:
      js.conf: |
        max_file_store:  1GB
        max_outstanding_catchup: 64MB
      my-config.conf: |
        ping_interval: "5m"
      my-config-last.conf: |
        ping_max: 3
`
	expected := DefaultResources(t, test)
	expected.Conf.Value["ping_interval"] = "5m"
	expected.Conf.Value["ping_max"] = int64(3)
	expected.Conf.Value["jetstream"] = map[string]any{
		"max_file_store":          int64(1073741824),
		"max_memory_store":        int64(0),
		"max_outstanding_catchup": int64(67108864),
		"store_dir":               "/data",
	}

	expected.ConfigMap.Value.Data = map[string]string{
		"js.conf": `max_file_store:  1GB
max_outstanding_catchup: 64MB
`,
		"my-config.conf": `ping_interval: "5m"
`,
		"my-config-last.conf": `ping_max: 3
`,
	}

	reloaderArgs := expected.StatefulSet.Value.Spec.Template.Spec.Containers[1].Args
	reloaderArgs = append(reloaderArgs, "-config", "/etc/nats-config/my-config.conf")
	reloaderArgs = append(reloaderArgs, "-config", "/etc/nats-config/js.conf")
	reloaderArgs = append(reloaderArgs, "-config", "/etc/nats-config/my-config-last.conf")
	expected.StatefulSet.Value.Spec.Template.Spec.Containers[1].Args = reloaderArgs

	vm := expected.StatefulSet.Value.Spec.Template.Spec.Containers[0].VolumeMounts
	expected.StatefulSet.Value.Spec.Template.Spec.Containers[0].VolumeMounts = append(vm, corev1.VolumeMount{
		MountPath: "/data",
		Name:      test.FullName + "-js",
	})

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

	RenderAndCheck(t, test, expected)
}

func TestExtraResources(t *testing.T) {
	t.Parallel()
	test := DefaultTest()
	test.Values = `
extraResources:
- apiVersion: v1
  kind: Service
  metadata:
    name:
      $tplYaml: >
        {{ include "nats.fullname" $ }}-extra
    labels:
      $tplYaml: |
        {{ include "nats.labels" $ }}
  spec:
    selector:
      labels:
        $tplYamlSpread: |
          {{ include "nats.selectorLabels" $ | nindent 4 }}
    ports:
    - $tplYamlSpread: |
        - name: gateway
          port: 7222
          targetPort: gateway
          appProtocol: tcp
- $tplYaml: |
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: {{ include "nats.fullname" $ }}-extra
      labels:
        {{- include "nats.labels" $ | nindent 4 }}
    data:
      foo: bar
`

	expected := DefaultResources(t, test)

	expected.ExtraConfigMap.HasValue = true
	expected.ExtraConfigMap.Value.Data = map[string]string{
		"foo": "bar",
	}

	expected.ExtraService.HasValue = true
	expected.ExtraService.Value.Spec.Ports = []corev1.ServicePort{
		{
			Name:        "gateway",
			Port:        7222,
			TargetPort:  intstr.FromString("gateway"),
			AppProtocol: &appProtocolTCP,
		},
	}

	RenderAndCheck(t, test, expected)
}
