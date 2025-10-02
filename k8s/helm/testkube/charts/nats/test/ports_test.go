package test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestPorts(t *testing.T) {
	t.Parallel()
	test := DefaultTest()
	test.Values = `
config:
  cluster:
    enabled: true
    port: 1005
  nats:
    port: 1001
  leafnodes:
    enabled: true
    port: 1002
  websocket:
    enabled: true
    port: 1003
  mqtt:
    enabled: true
    port: 1004
  gateway:
    enabled: true
    port: 1006
  monitor:
    port: 1007
  profiling:
    enabled: true
    port: 1008

container:
  ports:
    nats:
      hostPort: 2001
    leafnodes:
      hostPort: 2002
    websocket:
      hostPort: 2003
    mqtt:
      hostPort: 2004
    cluster:
      hostPort: 2005
    gateway:
      hostPort: 2006
    monitor:
      hostPort: 2007
    profiling:
      hostPort: 2008

service:
  merge:
    spec:
      type: NodePort
  ports:
    nats:
      enabled: true
      port: 3001
      nodePort: 4001
    leafnodes:
      enabled: true
      port: 3002
      nodePort: 4002
    websocket:
      enabled: true
      port: 3003
      nodePort: 4003
    mqtt:
      enabled: true
      port: 3004
      nodePort: 4004
    cluster:
      enabled: true
      port: 3005
      nodePort: 4005
    gateway:
      enabled: true
      port: 3006
      nodePort: 4006
    monitor:
      enabled: true
      port: 3007
      nodePort: 4007
    profiling:
      enabled: true
      port: 3008
      nodePort: 4008
`
	expected := DefaultResources(t, test)
	expected.Conf.Value["port"] = int64(1001)
	expected.Conf.Value["leafnodes"] = map[string]any{
		"port":         int64(1002),
		"no_advertise": true,
	}
	expected.Conf.Value["websocket"] = map[string]any{
		"port":   int64(1003),
		"no_tls": true,
	}
	expected.Conf.Value["mqtt"] = map[string]any{
		"port": int64(1004),
	}
	expected.Conf.Value["cluster"] = map[string]any{
		"name":         "nats",
		"no_advertise": true,
		"port":         int64(1005),
		"routes": []any{
			"nats://nats-0.nats-headless:1005",
			"nats://nats-1.nats-headless:1005",
			"nats://nats-2.nats-headless:1005",
		},
	}
	expected.Conf.Value["gateway"] = map[string]any{
		"port": int64(1006),
		"name": "nats",
	}
	expected.Conf.Value["http_port"] = int64(1007)
	expected.Conf.Value["prof_port"] = int64(1008)

	replicas3 := int32(3)
	expected.StatefulSet.Value.Spec.Replicas = &replicas3

	expected.StatefulSet.Value.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{
		{
			Name:          "nats",
			ContainerPort: 1001,
			HostPort:      2001,
		},
		{
			Name:          "leafnodes",
			ContainerPort: 1002,
			HostPort:      2002,
		},
		{
			Name:          "websocket",
			ContainerPort: 1003,
			HostPort:      2003,
		},
		{
			Name:          "mqtt",
			ContainerPort: 1004,
			HostPort:      2004,
		},
		{
			Name:          "cluster",
			ContainerPort: 1005,
			HostPort:      2005,
		},
		{
			Name:          "gateway",
			ContainerPort: 1006,
			HostPort:      2006,
		},
		{
			Name:          "monitor",
			ContainerPort: 1007,
			HostPort:      2007,
		},
		{
			Name:          "profiling",
			ContainerPort: 1008,
			HostPort:      2008,
		},
	}

	expected.HeadlessService.Value.Spec.Ports = []corev1.ServicePort{
		{
			Name:        "nats",
			Port:        1001,
			TargetPort:  intstr.FromString("nats"),
			AppProtocol: &appProtocolTCP,
		},
		{
			Name:        "leafnodes",
			Port:        1002,
			TargetPort:  intstr.FromString("leafnodes"),
			AppProtocol: &appProtocolTCP,
		},
		{
			Name:        "websocket",
			Port:        1003,
			TargetPort:  intstr.FromString("websocket"),
			AppProtocol: &appProtocolHTTP,
		},
		{
			Name:        "mqtt",
			Port:        1004,
			TargetPort:  intstr.FromString("mqtt"),
			AppProtocol: &appProtocolTCP,
		},
		{
			Name:        "cluster",
			Port:        1005,
			TargetPort:  intstr.FromString("cluster"),
			AppProtocol: &appProtocolTCP,
		},
		{
			Name:        "gateway",
			Port:        1006,
			TargetPort:  intstr.FromString("gateway"),
			AppProtocol: &appProtocolTCP,
		},
		{
			Name:        "monitor",
			Port:        1007,
			TargetPort:  intstr.FromString("monitor"),
			AppProtocol: &appProtocolHTTP,
		},
		{
			Name:        "profiling",
			Port:        1008,
			TargetPort:  intstr.FromString("profiling"),
			AppProtocol: &appProtocolTCP,
		},
	}

	expected.Service.Value.Spec.Type = "NodePort"
	expected.Service.Value.Spec.Ports = []corev1.ServicePort{
		{
			Name:        "nats",
			Port:        3001,
			NodePort:    4001,
			TargetPort:  intstr.FromString("nats"),
			AppProtocol: &appProtocolTCP,
		},
		{
			Name:        "leafnodes",
			Port:        3002,
			NodePort:    4002,
			TargetPort:  intstr.FromString("leafnodes"),
			AppProtocol: &appProtocolTCP,
		},
		{
			Name:        "websocket",
			Port:        3003,
			NodePort:    4003,
			TargetPort:  intstr.FromString("websocket"),
			AppProtocol: &appProtocolHTTP,
		},
		{
			Name:        "mqtt",
			Port:        3004,
			NodePort:    4004,
			TargetPort:  intstr.FromString("mqtt"),
			AppProtocol: &appProtocolTCP,
		},
		{
			Name:        "cluster",
			Port:        3005,
			NodePort:    4005,
			TargetPort:  intstr.FromString("cluster"),
			AppProtocol: &appProtocolTCP,
		},
		{
			Name:        "gateway",
			Port:        3006,
			NodePort:    4006,
			TargetPort:  intstr.FromString("gateway"),
			AppProtocol: &appProtocolTCP,
		},
		{
			Name:        "monitor",
			Port:        3007,
			NodePort:    4007,
			TargetPort:  intstr.FromString("monitor"),
			AppProtocol: &appProtocolHTTP,
		},
		{
			Name:        "profiling",
			Port:        3008,
			NodePort:    4008,
			TargetPort:  intstr.FromString("profiling"),
			AppProtocol: &appProtocolTCP,
		},
	}

	RenderAndCheck(t, test, expected)
}
