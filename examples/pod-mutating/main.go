// Package main demonstrates a mutating admission webhook that injects labels into pods.
//
// This example shows:
//   - Basic webhook configuration with Config struct
//   - Using environment variables for configuration (ACW_* prefix)
//   - Mutating webhook implementation with JSON patch response
//   - Proper error handling and logging
//
// Environment variables (all optional, with defaults):
//
//	ACW_NAME           - Webhook name (default: from Configure())
//	ACW_NAMESPACE      - Namespace for secrets/configmaps (default: auto-detected)
//	ACW_PORT           - HTTPS server port (default: 8443)
//	ACW_METRICS_PORT   - Metrics server port (default: 8080)
//	ACW_METRICS_ENABLED - Enable metrics (default: true)
//	ACW_LEADER_ELECTION - Enable leader election (default: true)
//	ACW_CA_VALIDITY    - CA certificate validity (default: 48h)
//	ACW_CERT_VALIDITY  - Server certificate validity (default: 24h)
//
// Usage:
//
//	# Run with defaults
//	go run main.go
//
//	# Run with custom configuration via environment variables
//	ACW_PORT=9443 ACW_METRICS_ENABLED=false go run main.go
//
//	# Run with verbose logging
//	go run main.go -v=2
package main

import (
	"encoding/json"
	"flag"
	"time"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	webhook "github.com/jimyag/auto-cert-webhook"
)

func main() {
	// Initialize klog flags for logging control (-v for verbosity)
	klog.InitFlags(nil)
	flag.Parse()

	// Create and run the webhook server
	// The framework handles:
	//   - TLS certificate generation and rotation
	//   - CA bundle synchronization to webhook configurations
	//   - Leader election for HA deployments
	//   - Metrics exposure
	//   - Health and readiness endpoints
	if err := webhook.Run(&podLabelInjector{}); err != nil {
		klog.Fatalf("Failed to run webhook: %v", err)
	}
}

// podLabelInjector implements the webhook.Webhook interface.
// It injects predefined labels into all pods during creation.
type podLabelInjector struct{}

// Configure returns the webhook server configuration.
// Values set here can be overridden by environment variables (ACW_* prefix).
// Priority: code values > environment variables > defaults
func (p *podLabelInjector) Configure() webhook.Config {
	// Enable metrics
	metricsEnabled := true

	// Enable leader election for HA deployments
	leaderElection := true

	return webhook.Config{
		// Name is used as the base for generated resource names:
		//   - CA Secret: {name}-ca
		//   - Cert Secret: {name}-cert
		//   - CA Bundle ConfigMap: {name}-ca-bundle
		//   - Leader Election Lease: {name}-leader
		Name: "pod-label-injector",

		// Namespace for storing secrets and configmaps.
		// If empty, auto-detected from:
		//   1. ACW_NAMESPACE environment variable
		//   2. POD_NAMESPACE environment variable
		//   3. ServiceAccount namespace file
		//   4. "default" as fallback
		Namespace: "",

		// Server configuration
		Port:        8443,       // HTTPS server port
		MetricsPort: 8080,       // Metrics server port
		HealthzPath: "/healthz", // Liveness probe endpoint
		ReadyzPath:  "/readyz",  // Readiness probe endpoint

		// Certificate configuration
		// Shorter validity periods are more secure but require more frequent rotation
		CAValidity:   48 * time.Hour, // CA certificate validity
		CARefresh:    24 * time.Hour, // Refresh CA when less than this remaining
		CertValidity: 24 * time.Hour, // Server certificate validity
		CertRefresh:  12 * time.Hour, // Refresh cert when less than this remaining

		// Feature toggles (use pointers to distinguish "not set" from "false")
		MetricsEnabled: &metricsEnabled,
		LeaderElection: &leaderElection,

		// Leader election configuration (for HA deployments)
		LeaseDuration: 30 * time.Second, // Duration a leader holds the lease
		RenewDeadline: 10 * time.Second, // Time to renew before lease expires
		RetryPeriod:   5 * time.Second,  // Time between leader election retries

		// Optional: Override auto-generated resource names
		// ServiceName:           "custom-service",
		// CASecretName:          "custom-ca-secret",
		// CertSecretName:        "custom-cert-secret",
		// CABundleConfigMapName: "custom-ca-bundle",
		// LeaderElectionID:      "custom-leader",
	}
}

// Webhooks returns all webhook handlers.
// Each Hook defines a path and admission function.
func (p *podLabelInjector) Webhooks() []webhook.Hook {
	return []webhook.Hook{
		{
			// Path is the URL path where this webhook is served
			// The full URL will be: https://{service}.{namespace}.svc:{port}/mutate-pods
			Path: "/mutate-pods",

			// Type indicates this is a mutating webhook (can modify resources)
			// Use webhook.Validating for validation-only webhooks
			Type: webhook.Mutating,

			// Admit is the handler function for admission requests
			Admit: p.mutatePod,
		},
	}
}

// mutatePod handles mutating admission requests for pods.
// It injects labels into the pod metadata.
func (p *podLabelInjector) mutatePod(ar admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	// Log the incoming request (visible with -v=2 or higher)
	klog.V(2).Infof("Mutating pod %s/%s", ar.Request.Namespace, ar.Request.Name)

	// Parse the pod from the admission request
	pod := &corev1.Pod{}
	if err := json.Unmarshal(ar.Request.Object.Raw, pod); err != nil {
		klog.Errorf("Failed to unmarshal pod: %v", err)
		// Return an error response - the request will be rejected
		return webhook.Errored(err)
	}

	// Make a deep copy for modification
	// This is important to correctly generate the JSON patch
	modifiedPod := pod.DeepCopy()

	// Initialize labels map if nil
	if modifiedPod.Labels == nil {
		modifiedPod.Labels = make(map[string]string)
	}

	// Inject labels
	modifiedPod.Labels["injected-by"] = "pod-label-injector"
	modifiedPod.Labels["injection-time"] = "admission"

	klog.V(2).Infof("Injected labels into pod %s/%s", ar.Request.Namespace, ar.Request.Name)

	// PatchResponse automatically generates the JSON patch from original to modified
	// If no changes are detected, it returns Allowed() without a patch
	return webhook.PatchResponse(pod, modifiedPod)
}
