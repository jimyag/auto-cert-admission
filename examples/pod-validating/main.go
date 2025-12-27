// Package main demonstrates a validating admission webhook that enforces pod policies.
//
// This example shows:
//   - Validating webhook implementation (reject/allow without modification)
//   - Policy enforcement with custom validation rules
//   - Different response types: Allowed, Denied, DeniedWithReason
//   - Handling multiple validation checks
//
// The webhook enforces the following policies:
//  1. Pods must have at least one label
//  2. Pods must not use the "latest" image tag
//  3. Pods must have resource limits defined
//
// Environment variables (all optional):
//
//	ACW_NAME           - Webhook name (default: from Configure())
//	ACW_NAMESPACE      - Namespace for secrets/configmaps
//	ACW_PORT           - HTTPS server port (default: 8443)
//
// Usage:
//
//	go run main.go
//
//	# Test with kubectl (after deploying)
//	kubectl run test --image=nginx:latest  # Should be denied
//	kubectl run test --image=nginx:1.25    # May pass (depends on other policies)
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	webhook "github.com/jimyag/auto-cert-webhook"
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	if err := webhook.Run(&podValidator{}); err != nil {
		klog.Fatalf("Failed to run webhook: %v", err)
	}
}

// podValidator implements the webhook.Webhook interface.
// It validates pods against a set of security policies.
type podValidator struct{}

// Configure returns a minimal configuration.
// Most settings will use defaults or environment variables.
func (p *podValidator) Configure() webhook.Config {
	return webhook.Config{
		Name: "pod-validator",
		// All other settings use defaults:
		//   Port: 8443, MetricsPort: 8080, etc.
	}
}

// Webhooks returns the validating webhook handler.
func (p *podValidator) Webhooks() []webhook.Hook {
	return []webhook.Hook{
		{
			Path:  "/validate-pods",
			Type:  webhook.Validating, // Validating webhooks can only allow or deny
			Admit: p.validatePod,
		},
	}
}

// validatePod validates pods against security policies.
// Returns Allowed() if all checks pass, or Denied() with explanation if any fail.
func (p *podValidator) validatePod(ar admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	klog.V(2).Infof("Validating pod %s/%s", ar.Request.Namespace, ar.Request.Name)

	// Parse the pod
	pod := &corev1.Pod{}
	if err := json.Unmarshal(ar.Request.Object.Raw, pod); err != nil {
		klog.Errorf("Failed to unmarshal pod: %v", err)
		return webhook.Errored(err)
	}

	// Collect all validation errors
	var violations []string

	// Policy 1: Pods must have at least one label
	if err := validateLabels(pod); err != nil {
		violations = append(violations, err.Error())
	}

	// Policy 2: No "latest" image tags
	if err := validateImageTags(pod); err != nil {
		violations = append(violations, err.Error())
	}

	// Policy 3: Resource limits must be defined
	if err := validateResourceLimits(pod); err != nil {
		violations = append(violations, err.Error())
	}

	// If any violations, deny the request
	if len(violations) > 0 {
		message := fmt.Sprintf("Pod validation failed:\n- %s", strings.Join(violations, "\n- "))
		klog.V(2).Infof("Denied pod %s/%s: %s", ar.Request.Namespace, ar.Request.Name, message)

		// DeniedWithReason allows specifying HTTP status code and reason
		return webhook.DeniedWithReason(
			message,
			metav1.StatusReasonForbidden,
			http.StatusForbidden,
		)
	}

	klog.V(2).Infof("Allowed pod %s/%s", ar.Request.Namespace, ar.Request.Name)

	// AllowedWithMessage can provide informational message (optional)
	return webhook.AllowedWithMessage("All validation checks passed")
}

// validateLabels checks that the pod has at least one label.
func validateLabels(pod *corev1.Pod) error {
	if len(pod.Labels) == 0 {
		return fmt.Errorf("pod must have at least one label")
	}
	return nil
}

// validateImageTags checks that no container uses the "latest" tag.
func validateImageTags(pod *corev1.Pod) error {
	var latestImages []string

	// Check all containers (init + regular)
	allContainers := append(pod.Spec.InitContainers, pod.Spec.Containers...)

	for _, container := range allContainers {
		image := container.Image

		// Check for explicit :latest tag or no tag (defaults to latest)
		if strings.HasSuffix(image, ":latest") {
			latestImages = append(latestImages, container.Name)
		} else if !strings.Contains(image, ":") && !strings.Contains(image, "@") {
			// No tag and no digest means :latest
			latestImages = append(latestImages, container.Name)
		}
	}

	if len(latestImages) > 0 {
		return fmt.Errorf("containers using 'latest' tag: %s", strings.Join(latestImages, ", "))
	}
	return nil
}

// validateResourceLimits checks that all containers have resource limits defined.
func validateResourceLimits(pod *corev1.Pod) error {
	var missing []string

	for _, container := range pod.Spec.Containers {
		limits := container.Resources.Limits
		if limits == nil {
			missing = append(missing, container.Name)
			continue
		}

		// Check for CPU and memory limits
		if _, ok := limits[corev1.ResourceCPU]; !ok {
			missing = append(missing, fmt.Sprintf("%s (no CPU limit)", container.Name))
		}
		if _, ok := limits[corev1.ResourceMemory]; !ok {
			missing = append(missing, fmt.Sprintf("%s (no memory limit)", container.Name))
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("containers missing resource limits: %s", strings.Join(missing, ", "))
	}
	return nil
}
