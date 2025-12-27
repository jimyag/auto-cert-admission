package cabundle

import (
	"context"
	"testing"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestSyncer_syncCABundle_NotFound(t *testing.T) {
	client := fake.NewSimpleClientset()
	syncer := NewSyncer(client, "test-ns", "ca-bundle", nil)

	ctx := context.Background()
	err := syncer.syncCABundle(ctx)
	// Should not return error if configmap not found
	if err != nil {
		t.Errorf("syncCABundle should not return error for not found: %v", err)
	}
}

func TestSyncer_syncCABundle_Found(t *testing.T) {
	// Create a configmap with CA bundle
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ca-bundle",
			Namespace: "test-ns",
		},
		Data: map[string]string{
			"ca-bundle.crt": "test-ca-bundle-data",
		},
	}

	// Create a mutating webhook configuration
	webhookConfig := &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-webhook",
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name: "test.webhook.svc",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					CABundle: []byte("old-ca-bundle"),
				},
			},
		},
	}

	client := fake.NewSimpleClientset(cm, webhookConfig)

	refs := []WebhookRef{
		{Name: "test-webhook", Type: MutatingWebhook},
	}

	syncer := NewSyncer(client, "test-ns", "ca-bundle", refs)

	ctx := context.Background()
	err := syncer.syncCABundle(ctx)
	if err != nil {
		t.Fatalf("syncCABundle failed: %v", err)
	}

	// Verify webhook was patched
	updated, err := client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(ctx, "test-webhook", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get updated webhook: %v", err)
	}

	if string(updated.Webhooks[0].ClientConfig.CABundle) != "test-ca-bundle-data" {
		t.Errorf("CABundle not updated: got %q, want %q",
			string(updated.Webhooks[0].ClientConfig.CABundle), "test-ca-bundle-data")
	}
}

func TestSyncer_patchValidatingWebhook(t *testing.T) {
	webhookConfig := &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-validating-webhook",
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			{
				Name: "test.webhook.svc",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					CABundle: []byte("old-ca"),
				},
			},
			{
				Name: "test2.webhook.svc",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					CABundle: []byte("old-ca"),
				},
			},
		},
	}

	client := fake.NewSimpleClientset(webhookConfig)
	syncer := NewSyncer(client, "test-ns", "ca-bundle", nil)

	ctx := context.Background()
	err := syncer.patchValidatingWebhook(ctx, "test-validating-webhook", []byte("new-ca-bundle"))
	if err != nil {
		t.Fatalf("patchValidatingWebhook failed: %v", err)
	}

	// Verify all webhooks were patched
	updated, err := client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(ctx, "test-validating-webhook", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get updated webhook: %v", err)
	}

	for i, webhook := range updated.Webhooks {
		if string(webhook.ClientConfig.CABundle) != "new-ca-bundle" {
			t.Errorf("Webhook %d CABundle not updated: got %q, want %q",
				i, string(webhook.ClientConfig.CABundle), "new-ca-bundle")
		}
	}
}

func TestSyncer_patchMutatingWebhook(t *testing.T) {
	webhookConfig := &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-mutating-webhook",
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name: "test.webhook.svc",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					CABundle: []byte("old-ca"),
				},
			},
		},
	}

	client := fake.NewSimpleClientset(webhookConfig)
	syncer := NewSyncer(client, "test-ns", "ca-bundle", nil)

	ctx := context.Background()
	err := syncer.patchMutatingWebhook(ctx, "test-mutating-webhook", []byte("new-ca-bundle"))
	if err != nil {
		t.Fatalf("patchMutatingWebhook failed: %v", err)
	}

	updated, err := client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(ctx, "test-mutating-webhook", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get updated webhook: %v", err)
	}

	if string(updated.Webhooks[0].ClientConfig.CABundle) != "new-ca-bundle" {
		t.Errorf("CABundle not updated: got %q, want %q",
			string(updated.Webhooks[0].ClientConfig.CABundle), "new-ca-bundle")
	}
}

func TestSyncer_patchWebhook_NotFound(t *testing.T) {
	client := fake.NewSimpleClientset()
	syncer := NewSyncer(client, "test-ns", "ca-bundle", nil)

	ctx := context.Background()

	// Should not return error if webhook not found
	err := syncer.patchValidatingWebhook(ctx, "non-existent", []byte("ca"))
	if err != nil {
		t.Errorf("patchValidatingWebhook should not return error for not found: %v", err)
	}

	err = syncer.patchMutatingWebhook(ctx, "non-existent", []byte("ca"))
	if err != nil {
		t.Errorf("patchMutatingWebhook should not return error for not found: %v", err)
	}
}

func TestSyncer_patchWebhook_UnknownType(t *testing.T) {
	client := fake.NewSimpleClientset()
	syncer := NewSyncer(client, "test-ns", "ca-bundle", nil)

	ctx := context.Background()
	err := syncer.patchWebhook(ctx, WebhookRef{Name: "test", Type: "unknown"}, []byte("ca"))

	if err == nil {
		t.Error("patchWebhook should return error for unknown type")
	}
}

func TestSyncer_onConfigMapUpdate_NoCABundle(t *testing.T) {
	client := fake.NewSimpleClientset()
	syncer := NewSyncer(client, "test-ns", "ca-bundle", nil)

	// ConfigMap without ca-bundle.crt
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ca-bundle",
			Namespace: "test-ns",
		},
		Data: map[string]string{},
	}

	ctx := context.Background()
	syncer.onConfigMapUpdate(ctx, cm)
	// Should not panic or error
}

func TestSyncer_syncMultipleWebhooks(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ca-bundle",
			Namespace: "test-ns",
		},
		Data: map[string]string{
			"ca-bundle.crt": "test-ca-data",
		},
	}

	mutatingWebhook := &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mutating-webhook",
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name: "test.webhook.svc",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					CABundle: []byte("old-ca"),
				},
			},
		},
	}

	validatingWebhook := &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "validating-webhook",
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			{
				Name: "validate.webhook.svc",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					CABundle: []byte("old-ca"),
				},
			},
		},
	}

	client := fake.NewSimpleClientset(cm, mutatingWebhook, validatingWebhook)

	refs := []WebhookRef{
		{Name: "mutating-webhook", Type: MutatingWebhook},
		{Name: "validating-webhook", Type: ValidatingWebhook},
	}

	syncer := NewSyncer(client, "test-ns", "ca-bundle", refs)

	ctx := context.Background()
	err := syncer.syncCABundle(ctx)
	if err != nil {
		t.Fatalf("syncCABundle failed: %v", err)
	}

	// Verify mutating webhook was updated
	updatedMutating, err := client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(ctx, "mutating-webhook", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get updated mutating webhook: %v", err)
	}
	if string(updatedMutating.Webhooks[0].ClientConfig.CABundle) != "test-ca-data" {
		t.Errorf("Mutating CABundle not updated: got %q, want %q",
			string(updatedMutating.Webhooks[0].ClientConfig.CABundle), "test-ca-data")
	}

	// Verify validating webhook was updated
	updatedValidating, err := client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(ctx, "validating-webhook", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get updated validating webhook: %v", err)
	}
	if string(updatedValidating.Webhooks[0].ClientConfig.CABundle) != "test-ca-data" {
		t.Errorf("Validating CABundle not updated: got %q, want %q",
			string(updatedValidating.Webhooks[0].ClientConfig.CABundle), "test-ca-data")
	}
}

func TestSyncer_EmptyWebhookRefs(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ca-bundle",
			Namespace: "test-ns",
		},
		Data: map[string]string{
			"ca-bundle.crt": "test-ca-data",
		},
	}

	client := fake.NewSimpleClientset(cm)

	// No webhook refs
	syncer := NewSyncer(client, "test-ns", "ca-bundle", nil)

	ctx := context.Background()
	err := syncer.syncCABundle(ctx)
	if err != nil {
		t.Errorf("syncCABundle with empty refs should not error: %v", err)
	}
}

func TestSyncer_WebhookWithMultipleHooks(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ca-bundle",
			Namespace: "test-ns",
		},
		Data: map[string]string{
			"ca-bundle.crt": "multi-ca-data",
		},
	}

	// Webhook with multiple hooks
	webhookConfig := &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "multi-hook-webhook",
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			{
				Name: "hook1.webhook.svc",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					CABundle: []byte("old-ca-1"),
				},
			},
			{
				Name: "hook2.webhook.svc",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					CABundle: []byte("old-ca-2"),
				},
			},
			{
				Name: "hook3.webhook.svc",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					CABundle: []byte("old-ca-3"),
				},
			},
		},
	}

	client := fake.NewSimpleClientset(cm, webhookConfig)

	refs := []WebhookRef{
		{Name: "multi-hook-webhook", Type: ValidatingWebhook},
	}

	syncer := NewSyncer(client, "test-ns", "ca-bundle", refs)

	ctx := context.Background()
	err := syncer.syncCABundle(ctx)
	if err != nil {
		t.Fatalf("syncCABundle failed: %v", err)
	}

	// Verify all hooks were updated
	updated, err := client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(ctx, "multi-hook-webhook", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get updated webhook: %v", err)
	}

	for i, webhook := range updated.Webhooks {
		if string(webhook.ClientConfig.CABundle) != "multi-ca-data" {
			t.Errorf("Webhook %d CABundle not updated: got %q, want %q",
				i, string(webhook.ClientConfig.CABundle), "multi-ca-data")
		}
	}
}

func TestSyncer_EmptyCABundleData(t *testing.T) {
	// ConfigMap with empty ca-bundle.crt
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ca-bundle",
			Namespace: "test-ns",
		},
		Data: map[string]string{
			"ca-bundle.crt": "",
		},
	}

	webhookConfig := &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-webhook",
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name: "test.webhook.svc",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					CABundle: []byte("original-ca"),
				},
			},
		},
	}

	client := fake.NewSimpleClientset(cm, webhookConfig)

	refs := []WebhookRef{
		{Name: "test-webhook", Type: MutatingWebhook},
	}

	syncer := NewSyncer(client, "test-ns", "ca-bundle", refs)

	ctx := context.Background()
	err := syncer.syncCABundle(ctx)
	if err != nil {
		t.Fatalf("syncCABundle failed: %v", err)
	}

	// Webhook should NOT be updated because ca-bundle.crt is empty
	updated, err := client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(ctx, "test-webhook", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get webhook: %v", err)
	}
	if string(updated.Webhooks[0].ClientConfig.CABundle) != "original-ca" {
		t.Errorf("Webhook should not be updated with empty CA bundle: got %q, want %q",
			string(updated.Webhooks[0].ClientConfig.CABundle), "original-ca")
	}
}

func TestSyncer_MissingCABundleKey(t *testing.T) {
	// ConfigMap without ca-bundle.crt key
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ca-bundle",
			Namespace: "test-ns",
		},
		Data: map[string]string{
			"other-key": "some-data",
		},
	}

	webhookConfig := &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-webhook",
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name: "test.webhook.svc",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					CABundle: []byte("original-ca"),
				},
			},
		},
	}

	client := fake.NewSimpleClientset(cm, webhookConfig)

	refs := []WebhookRef{
		{Name: "test-webhook", Type: MutatingWebhook},
	}

	syncer := NewSyncer(client, "test-ns", "ca-bundle", refs)

	ctx := context.Background()
	err := syncer.syncCABundle(ctx)
	if err != nil {
		t.Fatalf("syncCABundle failed: %v", err)
	}

	// Webhook should NOT be updated because ca-bundle.crt key is missing
	updated, err := client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(ctx, "test-webhook", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get webhook: %v", err)
	}
	if string(updated.Webhooks[0].ClientConfig.CABundle) != "original-ca" {
		t.Errorf("Webhook should not be updated with missing CA bundle key: got %q, want %q",
			string(updated.Webhooks[0].ClientConfig.CABundle), "original-ca")
	}
}
