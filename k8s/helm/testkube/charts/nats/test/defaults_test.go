package test

import (
	"sync"
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	policyv1 "k8s.io/api/policy/v1"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type DynamicDefaults struct {
	VersionLabel      string
	HelmChartLabel    string
	NatsImage         string
	PromExporterImage string
	ReloaderImage     string
	NatsBoxImage      string
}

type DynamicDefaultsGetter struct {
	mu  sync.Mutex
	set bool
	dd  DynamicDefaults
}

var (
	ddg              DynamicDefaultsGetter
	appProtocolTCP   = "tcp"
	appProtocolTLS   = "tls"
	appProtocolHTTP  = "http"
	appProtocolHTTPS = "https"
)

func (d *DynamicDefaultsGetter) Get(t *testing.T) DynamicDefaults {
	t.Helper()

	d.mu.Lock()
	defer d.mu.Unlock()
	if d.set {
		return d.dd
	}

	test := DefaultTest()
	test.Values = `
promExporter:
  enabled: true
`
	r := HelmRender(t, test)

	require.True(t, r.StatefulSet.HasValue)

	var ok bool
	d.dd.VersionLabel, ok = r.StatefulSet.Value.Labels["app.kubernetes.io/version"]
	require.True(t, ok)
	d.dd.HelmChartLabel, ok = r.StatefulSet.Value.Labels["helm.sh/chart"]
	require.True(t, ok)

	containers := r.StatefulSet.Value.Spec.Template.Spec.Containers
	require.Len(t, containers, 3)
	d.dd.NatsImage = containers[0].Image
	d.dd.ReloaderImage = containers[1].Image
	d.dd.PromExporterImage = containers[2].Image

	require.True(t, r.NatsBoxDeployment.HasValue)
	containers = r.NatsBoxDeployment.Value.Spec.Template.Spec.Containers
	require.Len(t, containers, 1)
	d.dd.NatsBoxImage = containers[0].Image

	return d.dd
}

func DefaultResources(t *testing.T, test *Test) *Resources {
	fullName := test.FullName
	chartName := test.ChartName
	releaseName := test.ReleaseName

	dd := ddg.Get(t)
	dr := GenerateResources(fullName)

	natsLabels := func() map[string]string {
		return map[string]string{
			"app.kubernetes.io/component":  "nats",
			"app.kubernetes.io/instance":   releaseName,
			"app.kubernetes.io/managed-by": "Helm",
			"app.kubernetes.io/name":       chartName,
			"app.kubernetes.io/version":    dd.VersionLabel,
			"helm.sh/chart":                dd.HelmChartLabel,
		}
	}
	natsSelectorLabels := func() map[string]string {
		return map[string]string{
			"app.kubernetes.io/component": "nats",
			"app.kubernetes.io/instance":  releaseName,
			"app.kubernetes.io/name":      chartName,
		}
	}
	natsBoxLabels := func() map[string]string {
		return map[string]string{
			"app.kubernetes.io/component":  "nats-box",
			"app.kubernetes.io/instance":   releaseName,
			"app.kubernetes.io/managed-by": "Helm",
			"app.kubernetes.io/name":       chartName,
			"app.kubernetes.io/version":    dd.VersionLabel,
			"helm.sh/chart":                dd.HelmChartLabel,
		}
	}
	natsBoxSelectorLabels := func() map[string]string {
		return map[string]string{
			"app.kubernetes.io/component": "nats-box",
			"app.kubernetes.io/instance":  releaseName,
			"app.kubernetes.io/name":      chartName,
		}
	}

	replicas1 := int32(1)
	trueBool := true
	falseBool := false
	exactPath := networkingv1.PathTypeExact

	return &Resources{
		Conf: Resource[map[string]any]{
			ID:       dr.Conf.ID,
			HasValue: true,
			Value: map[string]any{
				"http_port":              int64(8222),
				"lame_duck_duration":     "30s",
				"lame_duck_grace_period": "10s",
				"pid_file":               "/var/run/nats/nats.pid",
				"port":                   int64(4222),
				"server_name":            "nats-0",
			},
		},
		ConfigMap: Resource[corev1.ConfigMap]{
			ID:       dr.ConfigMap.ID,
			HasValue: true,
			Value: corev1.ConfigMap{
				TypeMeta: v1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:   fullName + "-config",
					Labels: natsLabels(),
				},
			},
		},
		HeadlessService: Resource[corev1.Service]{
			ID:       dr.HeadlessService.ID,
			HasValue: true,
			Value: corev1.Service{
				TypeMeta: v1.TypeMeta{
					Kind:       "Service",
					APIVersion: "v1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:   fullName + "-headless",
					Labels: natsLabels(),
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name:        "nats",
							Port:        4222,
							TargetPort:  intstr.FromString("nats"),
							AppProtocol: &appProtocolTCP,
						},
						{
							Name:        "monitor",
							Port:        8222,
							TargetPort:  intstr.FromString("monitor"),
							AppProtocol: &appProtocolHTTP,
						},
					},
					Selector:                 natsSelectorLabels(),
					ClusterIP:                "None",
					PublishNotReadyAddresses: true,
				},
			},
		},
		Ingress: Resource[networkingv1.Ingress]{
			ID:       dr.Ingress.ID,
			HasValue: false,
			Value: networkingv1.Ingress{
				TypeMeta: v1.TypeMeta{
					Kind:       "Ingress",
					APIVersion: "networking.k8s.io/v1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:   fullName + "-ws",
					Labels: natsLabels(),
				},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{
							Host: "demo.nats.io",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path:     "/",
											PathType: &exactPath,
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: fullName,
													Port: networkingv1.ServiceBackendPort{
														Name: "websocket",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		NatsBoxContentsSecret: Resource[corev1.Secret]{
			ID:       dr.NatsBoxContentsSecret.ID,
			HasValue: false,
			Value: corev1.Secret{
				TypeMeta: v1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:   fullName + "-box-contents",
					Labels: natsBoxLabels(),
				},
				Type: "Opaque",
			},
		},
		NatsBoxContextsSecret: Resource[corev1.Secret]{
			ID:       dr.NatsBoxContextsSecret.ID,
			HasValue: true,
			Value: corev1.Secret{
				TypeMeta: v1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:   fullName + "-box-contexts",
					Labels: natsBoxLabels(),
				},
				Type: "Opaque",
				StringData: map[string]string{
					"default.json": `{
  "url": "nats://` + fullName + `"
}
`,
				},
			},
		},
		NatsBoxDeployment: Resource[appsv1.Deployment]{
			ID:       dr.NatsBoxDeployment.ID,
			HasValue: true,
			Value: appsv1.Deployment{
				TypeMeta: v1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:   fullName + "-box",
					Labels: natsBoxLabels(),
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicas1,
					Selector: &v1.LabelSelector{
						MatchLabels: natsBoxSelectorLabels(),
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{
							Labels: natsBoxLabels(),
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Args: []string{
										"sh",
										"-ec",
										"trap true INT TERM; sleep infinity & wait",
									},
									Command: []string{
										"sh",
										"-ec",
										`work_dir="$(pwd)"
mkdir -p "$XDG_CONFIG_HOME/nats"
cd "$XDG_CONFIG_HOME/nats"
if ! [ -s context ]; then
  ln -s /etc/nats-contexts context
fi
if ! [ -f context.txt ]; then
  echo -n "default" > context.txt
fi
cd "$work_dir"
exec /entrypoint.sh "$@"
`,
										"--",
									},
									Image: dd.NatsBoxImage,
									Name:  "nats-box",
									VolumeMounts: []corev1.VolumeMount{
										{
											MountPath: "/etc/nats-contexts",
											Name:      "contexts",
										},
									},
								},
							},
							EnableServiceLinks: &falseBool,
							Volumes: []corev1.Volume{
								{
									Name: "contexts",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName: "nats-box-contexts",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		NatsBoxServiceAccount: Resource[corev1.ServiceAccount]{
			ID:       dr.NatsBoxServiceAccount.ID,
			HasValue: false,
			Value: corev1.ServiceAccount{
				TypeMeta: v1.TypeMeta{
					Kind:       "ServiceAccount",
					APIVersion: "v1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:   fullName + "-box",
					Labels: natsBoxLabels(),
				},
			},
		},
		PodDisruptionBudget: Resource[policyv1.PodDisruptionBudget]{
			ID:       dr.PodDisruptionBudget.ID,
			HasValue: true,
			Value: policyv1.PodDisruptionBudget{
				TypeMeta: v1.TypeMeta{
					Kind:       "PodDisruptionBudget",
					APIVersion: "policy/v1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:   fullName,
					Labels: natsLabels(),
				},
				Spec: policyv1.PodDisruptionBudgetSpec{
					MaxUnavailable: &intstr.IntOrString{IntVal: 1},
					Selector: &v1.LabelSelector{
						MatchLabels: natsSelectorLabels(),
					},
				},
			},
		},
		PodMonitor: Resource[monitoringv1.PodMonitor]{
			ID:       dr.PodMonitor.ID,
			HasValue: false,
			Value: monitoringv1.PodMonitor{
				TypeMeta: v1.TypeMeta{
					Kind:       "PodMonitor",
					APIVersion: "monitoring.coreos.com/v1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:   fullName,
					Labels: natsLabels(),
				},
				Spec: monitoringv1.PodMonitorSpec{
					PodMetricsEndpoints: []monitoringv1.PodMetricsEndpoint{
						{
							Port: "prom-metrics",
						},
					},
					Selector: v1.LabelSelector{
						MatchLabels: natsSelectorLabels(),
					},
				},
			},
		},
		Service: Resource[corev1.Service]{
			ID:       dr.Service.ID,
			HasValue: true,
			Value: corev1.Service{
				TypeMeta: v1.TypeMeta{
					Kind:       "Service",
					APIVersion: "v1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:   fullName,
					Labels: natsLabels(),
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name:        "nats",
							Port:        4222,
							TargetPort:  intstr.FromString("nats"),
							AppProtocol: &appProtocolTCP,
						},
					},
					Selector: natsSelectorLabels(),
				},
			},
		},
		ServiceAccount: Resource[corev1.ServiceAccount]{
			ID:       dr.ServiceAccount.ID,
			HasValue: false,
			Value: corev1.ServiceAccount{
				TypeMeta: v1.TypeMeta{
					Kind:       "ServiceAccount",
					APIVersion: "v1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:   fullName,
					Labels: natsLabels(),
				},
			},
		},
		StatefulSet: Resource[appsv1.StatefulSet]{
			ID:       dr.StatefulSet.ID,
			HasValue: true,
			Value: appsv1.StatefulSet{
				TypeMeta: v1.TypeMeta{
					Kind:       "StatefulSet",
					APIVersion: "apps/v1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:   fullName,
					Labels: natsLabels(),
				},
				Spec: appsv1.StatefulSetSpec{
					Replicas: &replicas1,
					Selector: &v1.LabelSelector{
						MatchLabels: natsSelectorLabels(),
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{
							Labels: natsLabels(),
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Args: []string{
										"--config",
										"/etc/nats-config/nats.conf",
									},
									Env: []corev1.EnvVar{
										{
											Name: "POD_NAME",
											ValueFrom: &corev1.EnvVarSource{
												FieldRef: &corev1.ObjectFieldSelector{
													FieldPath: "metadata.name",
												},
											},
										},
										{
											Name:  "SERVER_NAME",
											Value: "$(POD_NAME)",
										},
									},
									Image: dd.NatsImage,
									Lifecycle: &corev1.Lifecycle{
										PreStop: &corev1.LifecycleHandler{
											Exec: &corev1.ExecAction{
												Command: []string{
													"nats-server",
													"-sl=ldm=/var/run/nats/nats.pid",
												},
											},
										},
									},
									LivenessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Path: "/healthz?js-enabled-only=true",
												Port: intstr.FromString("monitor"),
											},
										},
										InitialDelaySeconds: 10,
										TimeoutSeconds:      5,
										PeriodSeconds:       30,
										SuccessThreshold:    1,
										FailureThreshold:    3,
									},
									Name: "nats",
									Ports: []corev1.ContainerPort{
										{
											Name:          "nats",
											ContainerPort: 4222,
										},
										{
											Name:          "monitor",
											ContainerPort: 8222,
										},
									},
									ReadinessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Path: "/healthz?js-server-only=true",
												Port: intstr.FromString("monitor"),
											},
										},
										InitialDelaySeconds: 10,
										TimeoutSeconds:      5,
										PeriodSeconds:       10,
										SuccessThreshold:    1,
										FailureThreshold:    3,
									},
									StartupProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Path: "/healthz",
												Port: intstr.FromString("monitor"),
											},
										},
										InitialDelaySeconds: 10,
										TimeoutSeconds:      5,
										PeriodSeconds:       10,
										SuccessThreshold:    1,
										FailureThreshold:    90,
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											MountPath: "/etc/nats-config",
											Name:      "config",
										},
										{
											MountPath: "/var/run/nats",
											Name:      "pid",
										},
									},
								},
								{
									Args: []string{
										"-pid",
										"/var/run/nats/nats.pid",
										"-config",
										"/etc/nats-config/nats.conf",
									},
									Image: dd.ReloaderImage,
									Name:  "reloader",
									VolumeMounts: []corev1.VolumeMount{
										{
											MountPath: "/var/run/nats",
											Name:      "pid",
										},
										{
											MountPath: "/etc/nats-config",
											Name:      "config",
										},
									},
								},
							},
							EnableServiceLinks:    &falseBool,
							ShareProcessNamespace: &trueBool,
							Volumes: []corev1.Volume{
								{
									Name: "config",
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "nats-config",
											},
										},
									},
								},
								{
									Name: "pid",
									VolumeSource: corev1.VolumeSource{
										EmptyDir: &corev1.EmptyDirVolumeSource{},
									},
								},
							},
						},
					},
					ServiceName:         fullName + "-headless",
					PodManagementPolicy: "Parallel",
				},
			},
		},
		ExtraConfigMap: Resource[corev1.ConfigMap]{
			ID:       dr.ExtraConfigMap.ID,
			HasValue: false,
			Value: corev1.ConfigMap{
				TypeMeta: v1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:   fullName + "-extra",
					Labels: natsLabels(),
				},
			},
		},
		ExtraService: Resource[corev1.Service]{
			ID:       dr.ExtraService.ID,
			HasValue: false,
			Value: corev1.Service{
				TypeMeta: v1.TypeMeta{
					Kind:       "Service",
					APIVersion: "v1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:   fullName + "-extra",
					Labels: natsLabels(),
				},
				Spec: corev1.ServiceSpec{
					Selector: natsSelectorLabels(),
				},
			},
		},
	}
}

func TestDefaultValues(t *testing.T) {
	t.Parallel()
	test := DefaultTest()
	expected := DefaultResources(t, test)
	RenderAndCheck(t, test, expected)
}
