// Package admission provides the main entry point for running admission webhooks
// with automatic certificate rotation.
package admission

import (
	"github.com/jimyag/auto-cert-webhook/pkg/webhook"
)

// Re-export types from webhook package for convenience
type (
	// Admission is the main interface that users need to implement.
	Admission = webhook.Admission

	// Config contains the server-level configuration.
	Config = webhook.Config

	// Hook defines a single admission webhook endpoint.
	Hook = webhook.Hook

	// HookType defines the type of admission webhook.
	HookType = webhook.HookType

	// AdmitFunc is the function signature for handling admission requests.
	AdmitFunc = webhook.AdmitFunc
)

// Re-export constants
const (
	Mutating   = webhook.Mutating
	Validating = webhook.Validating
)

// Re-export functions from webhook package
var (
	// Allowed returns an admission response that allows the request.
	Allowed = webhook.Allowed

	// AllowedWithMessage returns an admission response that allows the request with a message.
	AllowedWithMessage = webhook.AllowedWithMessage

	// Denied returns an admission response that denies the request.
	Denied = webhook.Denied

	// DeniedWithReason returns an admission response that denies the request with a specific reason.
	DeniedWithReason = webhook.DeniedWithReason

	// Errored returns an admission response for an error.
	Errored = webhook.Errored

	// ErroredWithCode returns an admission response for an error with a specific code.
	ErroredWithCode = webhook.ErroredWithCode

	// PatchResponse creates a patch response from the original and modified objects.
	PatchResponse = webhook.PatchResponse

	// PatchResponseFromRaw creates a patch response from raw JSON bytes.
	PatchResponseFromRaw = webhook.PatchResponseFromRaw

	// PatchResponseFromPatches creates a patch response from pre-built patches.
	PatchResponseFromPatches = webhook.PatchResponseFromPatches
)
