package prometheus

import (
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var logger = log.Log.WithName("fits-monitoring-prometheus-webhook")

// New returns a new mutating webhook that ensures that Prometheus objects conform to the fits-monitoring requirements.
func New(mgr manager.Manager) (*extensionswebhook.Webhook, error) {
	logger.Info("Adding Prometheus webhook to manager")

	ensurer := NewEnsurer(mgr, logger)

	types := []extensionswebhook.Type{
		{Obj: &monitoringv1.Prometheus{}},
	}

	handler := admission.HandlerFunc(ensurer.Handle)

	webhook := &extensionswebhook.Webhook{
		Name:     "prometheus",
		Provider: "",
		Types:    types,
		Target:   extensionswebhook.TargetSeed,
		Path:     "prometheus",
		Webhook:  &admission.Webhook{Handler: handler},
	}

	return webhook, nil
}
