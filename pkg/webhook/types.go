package webhook

import (
	admissionv1 "k8s.io/api/admission/v1"
)

// HookType defines the type of admission webhook.
type HookType string

const (
	// Mutating indicates a mutating admission webhook.
	Mutating HookType = "Mutating"
	// Validating indicates a validating admission webhook.
	Validating HookType = "Validating"
)

// AdmitFunc is the function signature for handling admission requests.
type AdmitFunc func(ar admissionv1.AdmissionReview) *admissionv1.AdmissionResponse

// Hook defines a single admission webhook endpoint.
type Hook struct {
	// Path is the URL path for this webhook, e.g., "/mutate-pods".
	Path string

	// Type is the webhook type: Mutating or Validating.
	Type HookType

	// Admit handles the admission request.
	Admit AdmitFunc
}

// Config contains the server-level configuration.
type Config struct {
	// Name is the webhook name, used for generating certificate resources.
	// This will be used as prefix for Secret, ConfigMap, and Lease names.
	Name string

	// Namespace is the namespace where the webhook is deployed.
	// If empty, uses POD_NAMESPACE environment variable or "default".
	Namespace string

	// ServiceName is the Kubernetes service name for the webhook.
	// If empty, defaults to the Name field.
	ServiceName string

	// Port is the port the webhook server listens on.
	// If 0, defaults to 8443.
	Port int

	// MetricsPort is the port for the metrics server.
	// If 0, defaults to 8080.
	MetricsPort int

	// MetricsEnabled enables the metrics server.
	// Defaults to true if not explicitly set.
	MetricsEnabled *bool

	// LeaderElection enables leader election for certificate management.
	// Defaults to true if not explicitly set.
	LeaderElection *bool
}

// Admission is the main interface that users need to implement.
type Admission interface {
	// Configure returns the server-level configuration.
	Configure() Config

	// Webhooks returns all webhook definitions.
	Webhooks() []Hook
}
