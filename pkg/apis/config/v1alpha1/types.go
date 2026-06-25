package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	healthcheckconfigv1alpha1 "github.com/gardener/gardener/extensions/pkg/apis/config/v1alpha1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ControllerConfiguration defines the configuration for the fi-ts monitoring-fits provider.
type ControllerConfiguration struct {
	metav1.TypeMeta `json:",inline"`

	// HealthCheckConfig is the config for the health check controller
	// +optional
	HealthCheckConfig *healthcheckconfigv1alpha1.HealthCheckConfig `json:"healthCheckConfig,omitempty"`

	// ImagePullSecret provides an opportunity to inject an image pull secret into the resource deployments
	ImagePullSecret *ImagePullSecret `json:"imagePullSecret,omitempty"`

	// Alertmanager is the configuration for external Alertmanager integration
	// +optional
	Alertmanager *AlertmanagerConfig `json:"alertmanager,omitempty"`

	// PrometheusRule is the configuration for custom PrometheusRule
	// +optional
	PrometheusRule *PrometheusRuleConfig `json:"prometheusRule,omitempty"`
}

// ImagePullSecret provides an opportunity to inject an image pull secret into the resource deployments
type ImagePullSecret struct {
	// DockerConfigJSON contains the already base64 encoded JSON content for the image pull secret
	DockerConfigJSON string `json:"encodedDockerConfigJSON"`
}

// AlertmanagerConfig contains the configuration for external Alertmanager integration.
type AlertmanagerConfig struct {
	// URL is the Alertmanager URL (host:port)
	URL string `json:"url"`
	// Username for basic auth
	Username string `json:"username"`
	// Password for basic auth
	Password string `json:"password"`
	// PathPrefix for the Alertmanager API
	PathPrefix string `json:"pathPrefix"`
	// Scheme (http/https)
	Scheme string `json:"scheme"`
}

// PrometheusRuleConfig contains the configuration for custom PrometheusRule.
type PrometheusRuleConfig struct {
	// Spec is the raw YAML content of the PrometheusRule spec
	Spec string `json:"spec"`
}
