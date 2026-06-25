package main

import (
	"os"

	"github.com/fi-ts/gardener-extension-monitoring-fits/cmd/gardener-extension-monitoring-fits/app"
	"github.com/gardener/gardener/pkg/logger"

	runtimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

func main() {
	runtimelog.SetLogger(logger.MustNewZapLogger(logger.InfoLevel, logger.FormatJSON))
	cmd := app.NewControllerManagerCommand()

	if err := cmd.Execute(); err != nil {
		runtimelog.Log.Error(err, "error executing the main controller command")
		os.Exit(1)
	}
}
