package prometheus

import (
	"context"
	"encoding/json"

	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// NewEnsurer creates a new shoot ensurer.
func NewEnsurer(mgr manager.Manager, logger logr.Logger) *ensurer {
	codec := serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder()
	return &ensurer{
		client:  mgr.GetClient(),
		decoder: codec,
		scheme:  mgr.GetScheme(),
		logger:  logger.WithName("fits-monitoring-prometheus-ensurer"),
	}
}

type ensurer struct {
	client  client.Client
	decoder runtime.Decoder
	scheme  *runtime.Scheme
	logger  logr.Logger
}

// Handle handles admission requests for Prometheus objects.
func (e *ensurer) Handle(ctx context.Context, req admission.Request) admission.Response {
	prometheus := &monitoringv1.Prometheus{}

	// Decode the object from the request
	if _, _, err := e.decoder.Decode(req.Object.Raw, nil, prometheus); err != nil {
		return admission.Errored(1, err)
	}

	// Apply mutations
	if err := e.EnsurePrometheus(prometheus, prometheus); err != nil {
		return admission.Errored(1, err)
	}

	// Encode the mutated object back to JSON
	marshaled, err := json.Marshal(prometheus)
	if err != nil {
		return admission.Errored(1, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaled)
}

// EnsurePrometheus ensures that the Prometheus object conforms to the provider requirements.
func (e *ensurer) EnsurePrometheus(new, _ *monitoringv1.Prometheus) error {
	e.logger.Info("ensuring Prometheus object", "name", new.Name, "namespace", new.Namespace)

	// Add additionalAlertManagerConfigs if not already present
	if new.Spec.AdditionalAlertManagerConfigs == nil {
		new.Spec.AdditionalAlertManagerConfigs = &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: "fits-am-confg",
			},
			Key: "additional-alertmanager-configs.yaml",
		}
		e.logger.Info("added additionalAlertManagerConfigs to Prometheus")
	}

	// Add additionalAlertRelabelConfigs if not already present
	if new.Spec.AdditionalAlertRelabelConfigs == nil {
		new.Spec.AdditionalAlertRelabelConfigs = &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: "fits-am-relabel-confg",
			},
			Key: "additional-alert-relabel-configs.yaml",
		}
		e.logger.Info("added additionalAlertRelabelConfigs to Prometheus")
	}

	return nil
}
