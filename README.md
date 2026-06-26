# gardener-extension-monitoring-fits

Gardener extension for integrating external Alertmanager with Prometheus in shoot clusters.

## Overview

This extension automatically configures Prometheus in every shoot cluster to forward alerts to an external Alertmanager. It:

- Creates Alertmanager configuration secrets in each shoot namespace
- Mutates Prometheus objects to include the Alertmanager configuration
- Applies alert relabeling rules for FITS-specific label transformations

## Features

- **Automatic Alertmanager Integration**: Seamlessly integrates external Alertmanager with shoot Prometheus instances
- **Dynamic Configuration**: Alertmanager credentials and URL are configurable via ComponentConfig
- **Custom Prometheus Rules**: Deploy custom PrometheusRules resources to shoot namespaces
- **Label Transformation**: Applies FITS-specific alert relabeling rules
- **Secret Management**: Automatically creates required secrets in shoot namespaces

## Configuration

The extension is configured via the Extension object in the Garden cluster. All configuration is done through the `values` section of the Extension object.

### Configuration Parameters

| Parameter | Description | Example |
| ----------- | ------------- | --------- |
| `config.alertmanager.url` | Alertmanager URL (host:port) | `alerts.example.com:443` |
| `config.alertmanager.username` | Basic auth username | `admin` |
| `config.alertmanager.password` | Basic auth password | `your-password` |
| `config.alertmanager.pathPrefix` | API path prefix | `/` |
| `config.alertmanager.scheme` | HTTP scheme | `https` |
| `config.PrometheusRules.spec` | Custom PrometheusRules spec (YAML) | See example below |
| `image.repository` | Extension image repository | `ghcr.io/fi-ts/gardener-extension-monitoring-fits` |
| `image.tag` | Extension image tag | `v0.1.0` |
| `imageVectorOverwrite` | Image vector configuration for webhooks | See example above |

## Deployment

### 1. Deploy the Extension

Apply the Extension object in the Garden cluster:

```bash
kubectl apply -f example/extension.yaml
```

The Extension object contains all necessary configuration including:

- Alertmanager credentials and URL
- Image repository and tag
- Image vector overwrites
- Auto-enablement for shoot clusters

### 2. Extension Object Configuration

The Extension object uses the new Gardener operator API:

```yaml
apiVersion: operator.gardener.cloud/v1alpha1
kind: Extension
metadata:
  name: fits-monitoring
spec:
  deployment:
    extension:
      helm:
        ociRepository:
          ref: ghcr.io/fi-ts/charts/gardener-extension-monitoring-fits:v0.1.0
      policy: Always
      values:
        config:
          alertmanager:
            url: alerts.example.com:443
            username: admin
            password: your-password-here
            pathPrefix: /
            scheme: https
          PrometheusRules:
            spec: |
              groups:
              - name: coredns-custom.rules
                rules:
                - alert: CoreDNSHighServfailRate
                  expr: |
                    (
                      rate(coredns_dns_responses_total{rcode="SERVFAIL"}[5m])
                      /
                      rate(coredns_dns_responses_total[5m])
                    ) > 0.05
                  for: 2m
                  labels:
                    severity: critical
                    type: shoot
                    service: coredns
                    visibility: all
                  annotations:
                    summary: "CoreDNS SERVFAIL rate above 5% for 2 minutes"
        image:
          pullPolicy: Always
          repository: ghcr.io/fi-ts/gardener-extension-monitoring-fits
          tag: v0.1.0
  resources:
  - autoEnable:
    - shoot
    clusterCompatibility:
    - shoot
    kind: Extension
    primary: true
    type: fits-monitoring
```

## How It Works

### 1. Secret Creation

The extension creates two secrets in the seed cluster's shoot namespace:

- **`fits-am-confg`**: Contains the Alertmanager configuration with credentials
- **`fits-am-relabel-confg`**: Contains static alert relabeling rules

These secrets are created using `managedresources.CreateForSeed()` and are automatically synchronized to the shoot namespace.

### 2. Custom PrometheusRules Deployment

When `config.PrometheusRules.spec` is configured, the extension creates a PrometheusRules resource in the shoot namespace:

- **Name**: `shoot-fits-custom`
- **Namespace**: Shoot namespace
- **Labels**: `prometheus: shoot` (ensures it's picked up by the shoot's Prometheus)
- **Spec**: The custom alert rules provided in the configuration

The PrometheusRules is deployed via managed resources and automatically synchronized to the shoot namespace, where it's picked up by the Prometheus instance.

### 3. Prometheus Mutation

A webhook mutates all Prometheus objects in shoot namespaces to include:

```yaml
spec:
  additionalAlertManagerConfigs:
    key: additional-alertmanager-configs.yaml
    name: fits-am-confg
  additionalAlertRelabelConfigs:
    key: additional-alert-relabel-configs.yaml
    name: fits-am-relabel-confg
```

### 4. Alert Relabeling

The extension applies FITS-specific label transformations:

- Replaces `mc_tool_rule` with `PROM.FITS.NATIVECLUSTER.KUBERNETES.5`
- Replaces `tenant` with `CN`
- Drops `prometheus` and `endpoint` labels
- Sets `severity` to `critical` for `KubeJobFailed` alerts

## Development

### Build

```bash
make build
```

### Generate Code

```bash
make generate
```

### Run Tests

```bash
make test
```

### Build Docker Image

```bash
make docker-image
```
