//go:generate sh -c "bash $GARDENER_HACK_DIR/generate-controller-registration.sh fits-monitoring-fits . $(cat ../../VERSION) ../../example/controller-registration.yaml Extension:fits-monitoring-fits"

// Package chart enables go:generate support for generating the correct controller registration.
package chart
