package cmd

import (
	"errors"
	"os"

	configapi "github.com/fi-ts/gardener-extension-monitoring-fits/pkg/apis/config"
	"github.com/fi-ts/gardener-extension-monitoring-fits/pkg/apis/config/v1alpha1"
	healthcheckconfig "github.com/gardener/gardener/extensions/pkg/apis/config/v1alpha1"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

var (
	scheme  *runtime.Scheme
	decoder runtime.Decoder
)

func init() {
	scheme = runtime.NewScheme()
	utilruntime.Must(configapi.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))

	decoder = serializer.NewCodecFactory(scheme).UniversalDecoder()
}

// MonitoringOptions holds options related to the monitoring service.
type MonitoringOptions struct {
	ConfigLocation string
	config         *MonitoringServiceConfig
}

// AddFlags implements Flagger.AddFlags.
func (o *MonitoringOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.ConfigLocation, "config", "", "Path to service configuration")
}

// Complete implements Completer.Complete.
func (o *MonitoringOptions) Complete() error {
	if o.ConfigLocation == "" {
		return errors.New("config location is not set")
	}
	data, err := os.ReadFile(o.ConfigLocation)
	if err != nil {
		return err
	}

	config := configapi.ControllerConfiguration{}
	_, _, err = decoder.Decode(data, nil, &config)
	if err != nil {
		return err
	}

	// if errs := validation.ValidateConfiguration(&config); len(errs) > 0 {
	// 	return errs.ToAggregate()
	// }

	o.config = &MonitoringServiceConfig{
		config: config,
	}

	return nil
}

// Completed returns the decoded MonitoringServiceConfiguration instance. Only call this if `Complete` was successful.
func (o *MonitoringOptions) Completed() *MonitoringServiceConfig {
	return o.config
}

// MonitoringServiceConfig contains configuration information about the monitoring service.
type MonitoringServiceConfig struct {
	config configapi.ControllerConfiguration
}

// Apply applies the MonitoringOptions to the passed ControllerOptions instance.
func (c *MonitoringServiceConfig) Apply(config *configapi.ControllerConfiguration) {
	*config = c.config
}

// ApplyHealthCheckConfig applies the HealthCheckConfig.
func (c *MonitoringServiceConfig) ApplyHealthCheckConfig(config *healthcheckconfig.HealthCheckConfig) {
	if c.config.HealthCheckConfig != nil {
		*config = *c.config.HealthCheckConfig
	}
}
