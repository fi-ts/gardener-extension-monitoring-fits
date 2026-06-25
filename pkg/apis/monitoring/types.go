package monitoring

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MonitoringConfig configuration resource
type MonitoringConfig struct {
	metav1.TypeMeta
}
