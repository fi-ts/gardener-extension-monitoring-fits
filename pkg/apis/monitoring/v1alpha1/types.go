package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SeedMonitoringResourceName = "extension-fits-monitoring"
	ShootMonitoringResourceName = "extension-fits-monitoring-shoot"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MonitoringConfig configuration resource
type MonitoringConfig struct {
	metav1.TypeMeta `json:",inline"`
}
