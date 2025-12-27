package autocertwebhook

import (
	"os"
	"testing"
	"time"

	"github.com/jimyag/auto-cert-webhook/internal/cabundle"
)

func TestApplyEnvConfig_Priority(t *testing.T) {
	// Clean up environment after test
	defer func() {
		for _, key := range []string{
			"ACW_NAME", "ACW_NAMESPACE", "ACW_PORT", "ACW_METRICS_PORT",
			"ACW_CA_VALIDITY", "ACW_METRICS_ENABLED", "ACW_LEADER_ELECTION",
		} {
			os.Unsetenv(key)
		}
	}()

	t.Run("code takes priority over env", func(t *testing.T) {
		os.Setenv("ACW_NAME", "env-name")
		os.Setenv("ACW_PORT", "9999")
		os.Setenv("ACW_CA_VALIDITY", "100h")

		cfg := Config{
			Name:       "code-name",
			Port:       1234,
			CAValidity: 50 * time.Hour,
		}

		if err := applyEnvConfig(&cfg); err != nil {
			t.Fatalf("applyEnvConfig failed: %v", err)
		}

		if cfg.Name != "code-name" {
			t.Errorf("Name: got %q, want %q", cfg.Name, "code-name")
		}
		if cfg.Port != 1234 {
			t.Errorf("Port: got %d, want %d", cfg.Port, 1234)
		}
		if cfg.CAValidity != 50*time.Hour {
			t.Errorf("CAValidity: got %v, want %v", cfg.CAValidity, 50*time.Hour)
		}
	})

	t.Run("env takes priority over default", func(t *testing.T) {
		os.Setenv("ACW_PORT", "9999")
		os.Setenv("ACW_CA_VALIDITY", "100h")
		os.Setenv("ACW_METRICS_PATH", "/custom-metrics")

		cfg := Config{}

		if err := applyEnvConfig(&cfg); err != nil {
			t.Fatalf("applyEnvConfig failed: %v", err)
		}

		if cfg.Port != 9999 {
			t.Errorf("Port: got %d, want %d", cfg.Port, 9999)
		}
		if cfg.CAValidity != 100*time.Hour {
			t.Errorf("CAValidity: got %v, want %v", cfg.CAValidity, 100*time.Hour)
		}
		if cfg.MetricsPath != "/custom-metrics" {
			t.Errorf("MetricsPath: got %q, want %q", cfg.MetricsPath, "/custom-metrics")
		}
	})

	t.Run("default values from struct tags", func(t *testing.T) {
		// Clear all env vars
		for _, key := range []string{
			"ACW_NAME", "ACW_NAMESPACE", "ACW_PORT", "ACW_METRICS_PORT",
			"ACW_CA_VALIDITY", "ACW_METRICS_ENABLED", "ACW_LEADER_ELECTION",
			"ACW_METRICS_PATH",
		} {
			os.Unsetenv(key)
		}

		cfg := Config{}

		if err := applyEnvConfig(&cfg); err != nil {
			t.Fatalf("applyEnvConfig failed: %v", err)
		}

		// Check defaults from struct tags
		if cfg.Port != 8443 {
			t.Errorf("Port: got %d, want %d", cfg.Port, 8443)
		}
		if cfg.MetricsPort != 8080 {
			t.Errorf("MetricsPort: got %d, want %d", cfg.MetricsPort, 8080)
		}
		if cfg.MetricsPath != "/metrics" {
			t.Errorf("MetricsPath: got %q, want %q", cfg.MetricsPath, "/metrics")
		}
		if cfg.HealthzPath != "/healthz" {
			t.Errorf("HealthzPath: got %q, want %q", cfg.HealthzPath, "/healthz")
		}
		if cfg.ReadyzPath != "/readyz" {
			t.Errorf("ReadyzPath: got %q, want %q", cfg.ReadyzPath, "/readyz")
		}
		if cfg.CAValidity != 48*time.Hour {
			t.Errorf("CAValidity: got %v, want %v", cfg.CAValidity, 48*time.Hour)
		}
		if cfg.CARefresh != 24*time.Hour {
			t.Errorf("CARefresh: got %v, want %v", cfg.CARefresh, 24*time.Hour)
		}
		if cfg.CertValidity != 24*time.Hour {
			t.Errorf("CertValidity: got %v, want %v", cfg.CertValidity, 24*time.Hour)
		}
		if cfg.CertRefresh != 12*time.Hour {
			t.Errorf("CertRefresh: got %v, want %v", cfg.CertRefresh, 12*time.Hour)
		}
		if cfg.LeaseDuration != 30*time.Second {
			t.Errorf("LeaseDuration: got %v, want %v", cfg.LeaseDuration, 30*time.Second)
		}
		if cfg.RenewDeadline != 10*time.Second {
			t.Errorf("RenewDeadline: got %v, want %v", cfg.RenewDeadline, 10*time.Second)
		}
		if cfg.RetryPeriod != 5*time.Second {
			t.Errorf("RetryPeriod: got %v, want %v", cfg.RetryPeriod, 5*time.Second)
		}
	})
}

func TestApplyEnvConfig_BoolPointers(t *testing.T) {
	defer func() {
		os.Unsetenv("ACW_METRICS_ENABLED")
		os.Unsetenv("ACW_LEADER_ELECTION")
	}()

	t.Run("bool pointer from env true", func(t *testing.T) {
		os.Setenv("ACW_METRICS_ENABLED", "true")
		os.Setenv("ACW_LEADER_ELECTION", "false")

		cfg := Config{}

		if err := applyEnvConfig(&cfg); err != nil {
			t.Fatalf("applyEnvConfig failed: %v", err)
		}

		if cfg.MetricsEnabled == nil || *cfg.MetricsEnabled != true {
			t.Errorf("MetricsEnabled: got %v, want true", cfg.MetricsEnabled)
		}
		if cfg.LeaderElection == nil || *cfg.LeaderElection != false {
			t.Errorf("LeaderElection: got %v, want false", cfg.LeaderElection)
		}
	})

	t.Run("code bool pointer takes priority", func(t *testing.T) {
		os.Setenv("ACW_METRICS_ENABLED", "true")

		falseVal := false
		cfg := Config{
			MetricsEnabled: &falseVal,
		}

		if err := applyEnvConfig(&cfg); err != nil {
			t.Fatalf("applyEnvConfig failed: %v", err)
		}

		if cfg.MetricsEnabled == nil {
			t.Errorf("MetricsEnabled: got nil, want non-nil")
		} else if *cfg.MetricsEnabled != false {
			t.Errorf("MetricsEnabled: got %v, want false (from code)", *cfg.MetricsEnabled)
		}
	})
}

func TestApplyDefaults_DynamicValues(t *testing.T) {
	t.Run("dynamic defaults based on Name", func(t *testing.T) {
		cfg := Config{
			Name: "my-webhook",
		}

		applyDefaults(&cfg)

		if cfg.ServiceName != "my-webhook" {
			t.Errorf("ServiceName: got %q, want %q", cfg.ServiceName, "my-webhook")
		}
		if cfg.CASecretName != "my-webhook-ca" {
			t.Errorf("CASecretName: got %q, want %q", cfg.CASecretName, "my-webhook-ca")
		}
		if cfg.CertSecretName != "my-webhook-cert" {
			t.Errorf("CertSecretName: got %q, want %q", cfg.CertSecretName, "my-webhook-cert")
		}
		if cfg.CABundleConfigMapName != "my-webhook-ca-bundle" {
			t.Errorf("CABundleConfigMapName: got %q, want %q", cfg.CABundleConfigMapName, "my-webhook-ca-bundle")
		}
		if cfg.LeaderElectionID != "my-webhook-leader" {
			t.Errorf("LeaderElectionID: got %q, want %q", cfg.LeaderElectionID, "my-webhook-leader")
		}
	})

	t.Run("explicit values not overwritten", func(t *testing.T) {
		cfg := Config{
			Name:                  "my-webhook",
			ServiceName:           "custom-service",
			CASecretName:          "custom-ca",
			CertSecretName:        "custom-cert",
			CABundleConfigMapName: "custom-bundle",
			LeaderElectionID:      "custom-leader",
		}

		applyDefaults(&cfg)

		if cfg.ServiceName != "custom-service" {
			t.Errorf("ServiceName: got %q, want %q", cfg.ServiceName, "custom-service")
		}
		if cfg.CASecretName != "custom-ca" {
			t.Errorf("CASecretName: got %q, want %q", cfg.CASecretName, "custom-ca")
		}
		if cfg.CertSecretName != "custom-cert" {
			t.Errorf("CertSecretName: got %q, want %q", cfg.CertSecretName, "custom-cert")
		}
		if cfg.CABundleConfigMapName != "custom-bundle" {
			t.Errorf("CABundleConfigMapName: got %q, want %q", cfg.CABundleConfigMapName, "custom-bundle")
		}
		if cfg.LeaderElectionID != "custom-leader" {
			t.Errorf("LeaderElectionID: got %q, want %q", cfg.LeaderElectionID, "custom-leader")
		}
	})
}

func TestGetNamespace(t *testing.T) {
	defer func() {
		os.Unsetenv("ACW_NAMESPACE")
		os.Unsetenv("POD_NAMESPACE")
	}()

	t.Run("ACW_NAMESPACE takes priority", func(t *testing.T) {
		os.Setenv("ACW_NAMESPACE", "acw-ns")
		os.Setenv("POD_NAMESPACE", "pod-ns")

		ns := getNamespace()
		if ns != "acw-ns" {
			t.Errorf("getNamespace: got %q, want %q", ns, "acw-ns")
		}
	})

	t.Run("POD_NAMESPACE as fallback", func(t *testing.T) {
		os.Unsetenv("ACW_NAMESPACE")
		os.Setenv("POD_NAMESPACE", "pod-ns")

		ns := getNamespace()
		if ns != "pod-ns" {
			t.Errorf("getNamespace: got %q, want %q", ns, "pod-ns")
		}
	})

	t.Run("default namespace when no env", func(t *testing.T) {
		os.Unsetenv("ACW_NAMESPACE")
		os.Unsetenv("POD_NAMESPACE")

		ns := getNamespace()
		// Will be "default" unless running in a pod with ServiceAccount
		if ns != "default" && ns == "" {
			t.Errorf("getNamespace: got empty, want non-empty")
		}
	})
}

func TestApplyEnvConfig_AllFieldTypes(t *testing.T) {
	defer func() {
		for _, key := range []string{
			"ACW_NAME", "ACW_PORT", "ACW_CA_VALIDITY", "ACW_METRICS_ENABLED",
		} {
			os.Unsetenv(key)
		}
	}()

	t.Run("string field", func(t *testing.T) {
		os.Setenv("ACW_NAME", "test-webhook")
		cfg := Config{}
		if err := applyEnvConfig(&cfg); err != nil {
			t.Fatalf("applyEnvConfig failed: %v", err)
		}
		if cfg.Name != "test-webhook" {
			t.Errorf("Name: got %q, want %q", cfg.Name, "test-webhook")
		}
	})

	t.Run("int field", func(t *testing.T) {
		os.Setenv("ACW_PORT", "9443")
		cfg := Config{}
		if err := applyEnvConfig(&cfg); err != nil {
			t.Fatalf("applyEnvConfig failed: %v", err)
		}
		if cfg.Port != 9443 {
			t.Errorf("Port: got %d, want %d", cfg.Port, 9443)
		}
	})

	t.Run("duration field", func(t *testing.T) {
		os.Setenv("ACW_CA_VALIDITY", "720h")
		cfg := Config{}
		if err := applyEnvConfig(&cfg); err != nil {
			t.Fatalf("applyEnvConfig failed: %v", err)
		}
		if cfg.CAValidity != 720*time.Hour {
			t.Errorf("CAValidity: got %v, want %v", cfg.CAValidity, 720*time.Hour)
		}
	})

	t.Run("bool pointer field", func(t *testing.T) {
		os.Setenv("ACW_METRICS_ENABLED", "false")
		cfg := Config{}
		if err := applyEnvConfig(&cfg); err != nil {
			t.Fatalf("applyEnvConfig failed: %v", err)
		}
		if cfg.MetricsEnabled == nil || *cfg.MetricsEnabled != false {
			t.Errorf("MetricsEnabled: got %v, want false", cfg.MetricsEnabled)
		}
	})
}

func TestConfigPriority_Integration(t *testing.T) {
	defer func() {
		os.Unsetenv("ACW_PORT")
		os.Unsetenv("ACW_CA_VALIDITY")
		os.Unsetenv("ACW_METRICS_PATH")
	}()

	t.Run("full priority chain: code > env > default", func(t *testing.T) {
		// Set env vars
		os.Setenv("ACW_PORT", "9999")        // will be overridden by code
		os.Setenv("ACW_CA_VALIDITY", "100h") // will be used (no code value)
		os.Unsetenv("ACW_METRICS_PATH")      // will use default

		cfg := Config{
			Name: "test-webhook",
			Port: 7777, // code value takes priority
		}

		if err := applyEnvConfig(&cfg); err != nil {
			t.Fatalf("applyEnvConfig failed: %v", err)
		}
		applyDefaults(&cfg)

		// Code takes priority
		if cfg.Port != 7777 {
			t.Errorf("Port: got %d, want %d (from code)", cfg.Port, 7777)
		}

		// Env takes priority over default
		if cfg.CAValidity != 100*time.Hour {
			t.Errorf("CAValidity: got %v, want %v (from env)", cfg.CAValidity, 100*time.Hour)
		}

		// Default used when no code or env
		if cfg.MetricsPath != "/metrics" {
			t.Errorf("MetricsPath: got %q, want %q (from default)", cfg.MetricsPath, "/metrics")
		}

		// Dynamic default based on Name
		if cfg.CASecretName != "test-webhook-ca" {
			t.Errorf("CASecretName: got %q, want %q (dynamic default)", cfg.CASecretName, "test-webhook-ca")
		}
	})
}

func TestDeepCopyConfig(t *testing.T) {
	t.Run("copies all fields", func(t *testing.T) {
		trueVal := true
		falseVal := false
		original := &Config{
			Name:           "test",
			Port:           8443,
			CAValidity:     48 * time.Hour,
			MetricsEnabled: &trueVal,
			LeaderElection: &falseVal,
		}

		copied := deepCopyConfig(original)

		// Verify values are copied
		if copied.Name != original.Name {
			t.Errorf("Name: got %q, want %q", copied.Name, original.Name)
		}
		if copied.Port != original.Port {
			t.Errorf("Port: got %d, want %d", copied.Port, original.Port)
		}
		if copied.CAValidity != original.CAValidity {
			t.Errorf("CAValidity: got %v, want %v", copied.CAValidity, original.CAValidity)
		}
	})

	t.Run("pointer fields are deep copied", func(t *testing.T) {
		trueVal := true
		original := &Config{
			MetricsEnabled: &trueVal,
		}

		copied := deepCopyConfig(original)

		// Verify pointer is different address
		if copied.MetricsEnabled == original.MetricsEnabled {
			t.Error("MetricsEnabled should be a different pointer")
		}

		// Verify value is same
		if *copied.MetricsEnabled != *original.MetricsEnabled {
			t.Errorf("MetricsEnabled value: got %v, want %v", *copied.MetricsEnabled, *original.MetricsEnabled)
		}

		// Modify original should not affect copy
		*original.MetricsEnabled = false
		if *copied.MetricsEnabled != true {
			t.Error("Modifying original should not affect copy")
		}
	})

	t.Run("nil pointers stay nil", func(t *testing.T) {
		original := &Config{
			Name:           "test",
			MetricsEnabled: nil,
			LeaderElection: nil,
		}

		copied := deepCopyConfig(original)

		if copied.MetricsEnabled != nil {
			t.Error("MetricsEnabled should be nil")
		}
		if copied.LeaderElection != nil {
			t.Error("LeaderElection should be nil")
		}
	})
}

func TestDetermineWebhookRefs(t *testing.T) {
	t.Run("mutating only", func(t *testing.T) {
		hooks := []Hook{
			{Path: "/mutate", Type: Mutating},
		}

		refs := determineWebhookRefs("my-webhook", hooks)

		if len(refs) != 1 {
			t.Fatalf("Expected 1 ref, got %d", len(refs))
		}
		if refs[0].Name != "my-webhook" {
			t.Errorf("Name: got %q, want %q", refs[0].Name, "my-webhook")
		}
		if refs[0].Type != cabundle.MutatingWebhook {
			t.Errorf("Type: got %v, want %v", refs[0].Type, cabundle.MutatingWebhook)
		}
	})

	t.Run("validating only", func(t *testing.T) {
		hooks := []Hook{
			{Path: "/validate", Type: Validating},
		}

		refs := determineWebhookRefs("my-webhook", hooks)

		if len(refs) != 1 {
			t.Fatalf("Expected 1 ref, got %d", len(refs))
		}
		if refs[0].Type != cabundle.ValidatingWebhook {
			t.Errorf("Type: got %v, want %v", refs[0].Type, cabundle.ValidatingWebhook)
		}
	})

	t.Run("both mutating and validating", func(t *testing.T) {
		hooks := []Hook{
			{Path: "/mutate", Type: Mutating},
			{Path: "/validate", Type: Validating},
		}

		refs := determineWebhookRefs("my-webhook", hooks)

		if len(refs) != 2 {
			t.Fatalf("Expected 2 refs, got %d", len(refs))
		}

		hasMutating := false
		hasValidating := false
		for _, ref := range refs {
			if ref.Type == cabundle.MutatingWebhook {
				hasMutating = true
			}
			if ref.Type == cabundle.ValidatingWebhook {
				hasValidating = true
			}
		}

		if !hasMutating {
			t.Error("Expected mutating webhook ref")
		}
		if !hasValidating {
			t.Error("Expected validating webhook ref")
		}
	})

	t.Run("multiple hooks of same type deduplicated", func(t *testing.T) {
		hooks := []Hook{
			{Path: "/mutate-pods", Type: Mutating},
			{Path: "/mutate-deployments", Type: Mutating},
			{Path: "/mutate-services", Type: Mutating},
		}

		refs := determineWebhookRefs("my-webhook", hooks)

		if len(refs) != 1 {
			t.Errorf("Expected 1 ref (deduplicated), got %d", len(refs))
		}
	})

	t.Run("empty hooks", func(t *testing.T) {
		refs := determineWebhookRefs("my-webhook", nil)

		if len(refs) != 0 {
			t.Errorf("Expected 0 refs, got %d", len(refs))
		}
	})
}
