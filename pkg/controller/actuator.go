package controller

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/fi-ts/gardener-extension-monitoring-fits/pkg/apis/config"
	"github.com/fi-ts/gardener-extension-monitoring-fits/pkg/apis/monitoring/v1alpha1"
	"github.com/fi-ts/gardener-extension-monitoring-fits/pkg/imagevector"
	"github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/extension"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	"github.com/go-logr/logr"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewActuator returns an actuator responsible for Extension resources.
func NewActuator(mgr manager.Manager, config config.ControllerConfiguration) extension.Actuator {
	return &actuator{
		client:  mgr.GetClient(),
		decoder: serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder(),
		config:  config,
	}
}

type actuator struct {
	client  client.Client
	decoder runtime.Decoder
	config  config.ControllerConfiguration
}

// ForceDelete implements extension.Actuator.
func (a *actuator) ForceDelete(context.Context, logr.Logger, *extensionsv1alpha1.Extension) error {
	return nil
}

// Reconcile the Extension resource.
func (a *actuator) Reconcile(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()

	cluster, err := controller.GetCluster(ctx, a.client, namespace)
	if err != nil {
		return err
	}

	if err := a.createResources(ctx, log, cluster, namespace); err != nil {
		return err
	}

	return nil
}

// Delete the Extension resource.
func (a *actuator) Delete(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	return a.deleteResources(ctx, log, ex.GetNamespace())
}

// Restore the Extension resource.
func (a *actuator) Restore(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	return a.Reconcile(ctx, log, ex)
}

// Migrate the Extension resource.
func (a *actuator) Migrate(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	return nil
}

func (a *actuator) createResources(ctx context.Context, log logr.Logger, cluster *controller.Cluster, namespace string) error {
	seedObjects, err := seedObjects(&a.config, cluster, namespace, a.config.Alertmanager)
	if err != nil {
		return err
	}

	seedResources, err := managedresources.NewRegistry(kubernetes.SeedScheme, kubernetes.SeedCodec, kubernetes.SeedSerializer).AddAllAndSerialize(seedObjects...)
	if err != nil {
		return err
	}

	if err := managedresources.CreateForSeed(ctx, a.client, namespace, v1alpha1.SeedMonitoringResourceName, false, seedResources); err != nil {
		return err
	}

	log.Info("managed resource created successfully", "name", v1alpha1.SeedMonitoringResourceName)

	return nil
}

func (a *actuator) deleteResources(ctx context.Context, log logr.Logger, namespace string) error {
	log.Info("deleting managed resource for monitoring")

	if err := managedresources.Delete(ctx, a.client, namespace, v1alpha1.SeedMonitoringResourceName, false); err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	if err := managedresources.WaitUntilDeleted(timeoutCtx, a.client, namespace, v1alpha1.SeedMonitoringResourceName); err != nil {
		return err
	}

	return nil
}

func seedObjects(cc *config.ControllerConfiguration, cluster *controller.Cluster, namespace string, alertmanagerConfig *config.AlertmanagerConfig) ([]client.Object, error) {
	monitoringImage, err := imagevector.ImageVector().FindImage("monitoring-fits-webhook")
	if err != nil {
		return nil, fmt.Errorf("failed to find monitoring-fits-webhook image: %w", err)
	}

	replicas := func(replicas int32) *int32 {
		if controller.IsHibernated(cluster) {
			return pointer.Pointer(int32(0))
		}
		return &replicas
	}

	webhookDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "monitoring-fits-webhook",
			Namespace: namespace,
			Labels: map[string]string{
				"k8s-app": "monitoring-fits-webhook",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: replicas(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"k8s-app": "monitoring-fits-webhook",
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 0,
					},
					MaxSurge: &intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 1,
					},
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"k8s-app": "monitoring-fits-webhook",
						"app":     "monitoring-fits-webhook",
						"networking.gardener.cloud/from-prometheus":      "allowed",
						"networking.gardener.cloud/from-shoot-apiserver": "allowed",
						"networking.gardener.cloud/to-dns":               "allowed",
						"networking.gardener.cloud/to-public-networks":   "allowed",
					},
					Annotations: map[string]string{
						"scheduler.alpha.kubernetes.io/critical-pod": "",
						"prometheus.io/scrape":                       "true",
						"prometheus.io/path":                         "/metrics",
						"prometheus.io/port":                         "2112",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "kubernetes-monitoring-fits-webhook",
							Image:           monitoringImage.String(),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8443,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "monitoring",
									ContainerPort: 2112,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "LISTEN",
									Value: ":8443",
								},
								{
									Name:  "CLUSTER",
									Value: cluster.Shoot.Name,
								},
							},
						},
					},
				},
			},
		},
	}

	objects := []client.Object{
		webhookDeployment,
		&policyv1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{
				Name:      webhookDeployment.Name,
				Namespace: webhookDeployment.Namespace,
			},
			Spec: policyv1.PodDisruptionBudgetSpec{
				MinAvailable: &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 1,
				},
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"k8s-app": "monitoring-fits-webhook",
					},
				},
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "monitoring-fits-webhook",
				Namespace: namespace,
				Labels: map[string]string{
					"app": "monitoring-fits-webhook",
				},
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"app": "monitoring-fits-webhook",
				},
				Ports: []corev1.ServicePort{
					{
						Port:       8443,
						TargetPort: intstr.FromInt(8443),
					},
				},
			},
		},
	}

	if cc.ImagePullSecret != nil && cc.ImagePullSecret.DockerConfigJSON != "" {
		content, err := base64.StdEncoding.DecodeString(cc.ImagePullSecret.DockerConfigJSON)
		if err != nil {
			return nil, fmt.Errorf("unable to decode image pull secret: %w", err)
		}

		objects = append(objects, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "monitoring-fits-webhook-registry-credentials",
				Namespace: namespace,
				Labels: map[string]string{
					"app": "monitoring-fits-webhook-registry-credentials",
				},
			},
			Type: corev1.SecretTypeDockerConfigJson,
			Data: map[string][]byte{
				".dockerconfigjson": content,
			},
		})

		webhookDeployment.Spec.Template.Spec.ImagePullSecrets = append(webhookDeployment.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{
			Name: "monitoring-fits-webhook-registry-credentials",
		})
	}

	// Add alertmanager secrets if configured
	if alertmanagerConfig != nil {
		// Create alertmanager config secret
		alertmanagerConfigYAML := fmt.Sprintf(`basic_auth:
  password: %s
  username: %s
path_prefix: %s
scheme: %s
static_configs:
  - targets:
    - %s`,
			alertmanagerConfig.Password,
			alertmanagerConfig.Username,
			alertmanagerConfig.PathPrefix,
			alertmanagerConfig.Scheme,
			alertmanagerConfig.URL,
		)

		objects = append(objects, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fits-am-confg",
				Namespace: "",
				Labels: map[string]string{
					"release": "prometheus",
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"additional-alertmanager-configs.yaml": []byte(alertmanagerConfigYAML),
			},
		})

		// Create alert relabel config secret (static)
		alertRelabelConfigYAML := `- regex: ()
  replacement: PROMO.FITS.NATIVECLUSTER.KUBERNETES.5
  source_labels:
  - mc_tool_rule
  target_label: mc_tool_rule
- regex: ()
  replacement: CN
  source_labels:
  - tenant
  target_label: tenant
- action: labeldrop
  regex: prometheus
- action: labeldrop
  regex: endpoint
- regex: KubJobFailed
  replacement: critical
  source_labels:
  - alertname
  target_label: severity`

		objects = append(objects, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fits-am-relabel-confg",
				Namespace: "",
				Labels: map[string]string{
					"release": "prometheus",
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"additional-alert-relabel-configs.yaml": []byte(alertRelabelConfigYAML),
			},
		})
	}

	return objects, nil
}
