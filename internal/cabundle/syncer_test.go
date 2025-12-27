package cabundle

import (
	"testing"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateValidatingWebhookConfiguration(t *testing.T) {
	failPolicy := admissionregistrationv1.Fail
	sideEffects := admissionregistrationv1.SideEffectClassNone
	matchPolicy := admissionregistrationv1.Equivalent
	timeout := int32(10)

	rules := []admissionregistrationv1.RuleWithOperations{
		{
			Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create},
			Rule: admissionregistrationv1.Rule{
				APIGroups:   []string{""},
				APIVersions: []string{"v1"},
				Resources:   []string{"pods"},
			},
		},
	}

	nsSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"env": "test"},
	}

	caBundle := []byte("test-ca-bundle")

	config := CreateValidatingWebhookConfiguration(
		"my-webhook",
		"webhook-system",
		"my-service",
		"/validate",
		443,
		caBundle,
		rules,
		&failPolicy,
		&sideEffects,
		&matchPolicy,
		nsSelector,
		nil,
		&timeout,
	)

	if config.Name != "my-webhook" {
		t.Errorf("Name: got %q, want %q", config.Name, "my-webhook")
	}

	if len(config.Webhooks) != 1 {
		t.Fatalf("Expected 1 webhook, got %d", len(config.Webhooks))
	}

	webhook := config.Webhooks[0]

	if webhook.Name != "my-service.webhook-system.svc" {
		t.Errorf("Webhook name: got %q, want %q", webhook.Name, "my-service.webhook-system.svc")
	}

	if webhook.ClientConfig.Service.Name != "my-service" {
		t.Errorf("Service name: got %q, want %q", webhook.ClientConfig.Service.Name, "my-service")
	}

	if webhook.ClientConfig.Service.Namespace != "webhook-system" {
		t.Errorf("Service namespace: got %q, want %q", webhook.ClientConfig.Service.Namespace, "webhook-system")
	}

	if *webhook.ClientConfig.Service.Path != "/validate" {
		t.Errorf("Service path: got %q, want %q", *webhook.ClientConfig.Service.Path, "/validate")
	}

	if *webhook.ClientConfig.Service.Port != 443 {
		t.Errorf("Service port: got %d, want %d", *webhook.ClientConfig.Service.Port, 443)
	}

	if string(webhook.ClientConfig.CABundle) != "test-ca-bundle" {
		t.Errorf("CABundle: got %q, want %q", string(webhook.ClientConfig.CABundle), "test-ca-bundle")
	}

	if *webhook.FailurePolicy != failPolicy {
		t.Errorf("FailurePolicy: got %v, want %v", *webhook.FailurePolicy, failPolicy)
	}

	if *webhook.SideEffects != sideEffects {
		t.Errorf("SideEffects: got %v, want %v", *webhook.SideEffects, sideEffects)
	}

	if *webhook.TimeoutSeconds != timeout {
		t.Errorf("TimeoutSeconds: got %d, want %d", *webhook.TimeoutSeconds, timeout)
	}

	if len(webhook.AdmissionReviewVersions) != 2 {
		t.Errorf("AdmissionReviewVersions: got %d versions, want 2", len(webhook.AdmissionReviewVersions))
	}
}

func TestCreateMutatingWebhookConfiguration(t *testing.T) {
	failPolicy := admissionregistrationv1.Ignore
	sideEffects := admissionregistrationv1.SideEffectClassNoneOnDryRun
	matchPolicy := admissionregistrationv1.Exact
	timeout := int32(5)
	reinvocation := admissionregistrationv1.IfNeededReinvocationPolicy

	rules := []admissionregistrationv1.RuleWithOperations{
		{
			Operations: []admissionregistrationv1.OperationType{
				admissionregistrationv1.Create,
				admissionregistrationv1.Update,
			},
			Rule: admissionregistrationv1.Rule{
				APIGroups:   []string{"apps"},
				APIVersions: []string{"v1"},
				Resources:   []string{"deployments"},
			},
		},
	}

	caBundle := []byte("mutating-ca-bundle")

	config := CreateMutatingWebhookConfiguration(
		"mutating-webhook",
		"default",
		"mutating-service",
		"/mutate",
		8443,
		caBundle,
		rules,
		&failPolicy,
		&sideEffects,
		&matchPolicy,
		nil,
		nil,
		&timeout,
		&reinvocation,
	)

	if config.Name != "mutating-webhook" {
		t.Errorf("Name: got %q, want %q", config.Name, "mutating-webhook")
	}

	if len(config.Webhooks) != 1 {
		t.Fatalf("Expected 1 webhook, got %d", len(config.Webhooks))
	}

	webhook := config.Webhooks[0]

	if webhook.Name != "mutating-service.default.svc" {
		t.Errorf("Webhook name: got %q, want %q", webhook.Name, "mutating-service.default.svc")
	}

	if *webhook.ReinvocationPolicy != reinvocation {
		t.Errorf("ReinvocationPolicy: got %v, want %v", *webhook.ReinvocationPolicy, reinvocation)
	}

	if *webhook.FailurePolicy != failPolicy {
		t.Errorf("FailurePolicy: got %v, want %v", *webhook.FailurePolicy, failPolicy)
	}

	if *webhook.ClientConfig.Service.Port != 8443 {
		t.Errorf("Service port: got %d, want %d", *webhook.ClientConfig.Service.Port, 8443)
	}
}

func TestWebhookRef(t *testing.T) {
	ref := WebhookRef{
		Name: "test-webhook",
		Type: ValidatingWebhook,
	}

	if ref.Name != "test-webhook" {
		t.Errorf("Name: got %q, want %q", ref.Name, "test-webhook")
	}

	if ref.Type != ValidatingWebhook {
		t.Errorf("Type: got %v, want %v", ref.Type, ValidatingWebhook)
	}
}

func TestWebhookType_Constants(t *testing.T) {
	if ValidatingWebhook != "validating" {
		t.Errorf("ValidatingWebhook: got %q, want %q", ValidatingWebhook, "validating")
	}

	if MutatingWebhook != "mutating" {
		t.Errorf("MutatingWebhook: got %q, want %q", MutatingWebhook, "mutating")
	}
}

func TestNewSyncer(t *testing.T) {
	refs := []WebhookRef{
		{Name: "webhook1", Type: ValidatingWebhook},
		{Name: "webhook2", Type: MutatingWebhook},
	}

	syncer := NewSyncer(nil, "test-ns", "ca-bundle-cm", refs)

	if syncer.namespace != "test-ns" {
		t.Errorf("namespace: got %q, want %q", syncer.namespace, "test-ns")
	}

	if syncer.caBundleConfigMapName != "ca-bundle-cm" {
		t.Errorf("caBundleConfigMapName: got %q, want %q", syncer.caBundleConfigMapName, "ca-bundle-cm")
	}

	if len(syncer.webhookRefs) != 2 {
		t.Errorf("webhookRefs: got %d, want 2", len(syncer.webhookRefs))
	}
}
