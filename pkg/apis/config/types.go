package config

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	healthcheckconfig "github.com/gardener/gardener/extensions/pkg/apis/config/v1alpha1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ControllerConfiguration defines the configuration for the fi-ts monitoring-fits provider.
type ControllerConfiguration struct {
	metav1.TypeMeta

	// HealthCheckConfig is the config for the health check controller
	HealthCheckConfig *healthcheckconfig.HealthCheckConfig

	// ImagePullSecret provides an opportunity to inject an image pull secret into the resource deployments
	ImagePullSecret *ImagePullSecret

	// Alertmanager is the configuration for external Alertmanager integration
	Alertmanager *AlertmanagerConfig
}

// ImagePullSecret provides an opportunity to inject an image pull secret into the resource deployments
type ImagePullSecret struct {
	// DockerConfigJSON contains the already base64 encoded JSON content for the image pull secret
	DockerConfigJSON string
}

// AlertmanagerConfig contains the configuration for external Alertmanager integration.
type AlertmanagerConfig struct {
	// URL is the Alertmanager URL (host:port)
	URL string
	// Username for basic auth
	Username string
	// Password for basic auth
	Password string
	// PathPrefix for the Alertmanager API
	PathPrefix string
	// Scheme (http/https)
	Scheme string
}
