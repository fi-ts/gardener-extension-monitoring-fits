package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/fi-ts/gardener-extension-monitoring-fits/pkg/apis/config"
	"github.com/fi-ts/gardener-extension-monitoring-fits/pkg/apis/monitoring/v1alpha1"
	"github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/extension"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/yaml"

	corev1 "k8s.io/api/core/v1"
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
	seedObjects, err := seedObjects(&a.config, cluster, namespace, a.config.Alertmanager, a.config.PrometheusRules)
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

func seedObjects(cc *config.ControllerConfiguration, cluster *controller.Cluster, namespace string, alertmanagerConfig *config.AlertmanagerConfig, prometheusRulesConfig *config.PrometheusRulesConfig) ([]client.Object, error) {
	objects := []client.Object{}

	// Add alertmanager secrets if configured
	if alertmanagerConfig != nil {
		// Create alertmanager config secret
		alertmanagerConfigYAML := fmt.Sprintf(`- basic_auth:
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
				Name:      v1alpha1.AlertmanagerConfigSecretName,
				Namespace: namespace,
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
  replacement: PROM.FITS.NATIVECLUSTER.KUBERNETES.5
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
- regex: KubeJobFailed
  replacement: critical
  source_labels:
  - alertname
  target_label: severity`

		objects = append(objects, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      v1alpha1.AlertRelabelConfigSecretName,
				Namespace: namespace,
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

	// Add custom PrometheusRules if configured
	if prometheusRulesConfig != nil {
		// Create PrometheusRule resource
		prometheusRules := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "monitoring.coreos.com/v1",
				"kind":       "PrometheusRule",
				"metadata": map[string]interface{}{
					"name":      "shoot-fits-custom",
					"namespace": namespace,
					"labels": map[string]interface{}{
						"prometheus": "shoot",
					},
				},
				"spec": map[string]interface{}{},
			},
		}

		// Parse the YAML spec and set it in the PrometheusRule
		var spec map[string]interface{}
		if err := yaml.Unmarshal([]byte(prometheusRulesConfig.Spec), &spec); err != nil {
			return nil, fmt.Errorf("failed to parse PrometheusRules spec: %w", err)
		}
		prometheusRules.Object["spec"] = spec

		objects = append(objects, prometheusRules)
	} else {
		// If no PrometheusRules is configured, create an empty PrometheusRule to ensure the resource exists
		prometheusRules := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "monitoring.coreos.com/v1",
				"kind":       "PrometheusRule",
				"metadata": map[string]interface{}{
					"name":      "shoot-fits-custom",
					"namespace": namespace,
					"labels": map[string]interface{}{
						"prometheus": "shoot",
					},
				},
				"spec": map[string]interface{}{
					"groups": []interface{}{},
				},
			},
		}
		objects = append(objects, prometheusRules)
	}

	return objects, nil
}
