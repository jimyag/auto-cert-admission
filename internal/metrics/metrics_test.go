package metrics

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestUpdateCertMetrics(t *testing.T) {
	// Reset metrics for testing
	certExpiryTimestamp.Reset()
	certNotBeforeTimestamp.Reset()
	certValidDurationSeconds.Reset()

	// Create a test certificate
	notBefore := time.Now()
	notAfter := notBefore.Add(24 * time.Hour)
	cert := createTestCert(t, notBefore, notAfter)

	t.Run("updates all metrics", func(t *testing.T) {
		UpdateCertMetrics("serving", cert)

		// Check expiry timestamp
		expiry := getGaugeValue(t, certExpiryTimestamp, "serving")
		if expiry != float64(notAfter.Unix()) {
			t.Errorf("certExpiryTimestamp: got %v, want %v", expiry, float64(notAfter.Unix()))
		}

		// Check not-before timestamp
		notBeforeVal := getGaugeValue(t, certNotBeforeTimestamp, "serving")
		if notBeforeVal != float64(notBefore.Unix()) {
			t.Errorf("certNotBeforeTimestamp: got %v, want %v", notBeforeVal, float64(notBefore.Unix()))
		}

		// Check valid duration
		duration := getGaugeValue(t, certValidDurationSeconds, "serving")
		expectedDuration := notAfter.Sub(notBefore).Seconds()
		if duration != expectedDuration {
			t.Errorf("certValidDurationSeconds: got %v, want %v", duration, expectedDuration)
		}
	})

	t.Run("handles different cert types", func(t *testing.T) {
		certExpiryTimestamp.Reset()

		UpdateCertMetrics("ca", cert)
		UpdateCertMetrics("serving", cert)

		// Both should have metrics
		caExpiry := getGaugeValue(t, certExpiryTimestamp, "ca")
		servingExpiry := getGaugeValue(t, certExpiryTimestamp, "serving")

		if caExpiry != float64(notAfter.Unix()) {
			t.Errorf("CA expiry: got %v, want %v", caExpiry, float64(notAfter.Unix()))
		}
		if servingExpiry != float64(notAfter.Unix()) {
			t.Errorf("Serving expiry: got %v, want %v", servingExpiry, float64(notAfter.Unix()))
		}
	})

	t.Run("nil certificate is handled", func(t *testing.T) {
		// Should not panic
		UpdateCertMetrics("test", nil)
	})
}

func TestRegister(t *testing.T) {
	// Register should be idempotent (can be called multiple times)
	Register()
	Register()
	Register()
	// If it panics, the test fails
}

func TestUpdateLeaderMetrics(t *testing.T) {
	resetLeaderMetrics()

	t.Run("records current leader", func(t *testing.T) {
		UpdateLeaderMetrics("luckin", "luckin-admission-webhook-leader", "pod-a")

		hasLeader := getGaugeValueWithLabels(t, hasLeader, prometheus.Labels{
			"namespace": "luckin",
			"lease":     "luckin-admission-webhook-leader",
		})
		if hasLeader != 1 {
			t.Fatalf("hasLeader: got %v, want 1", hasLeader)
		}

		leaderMetrics := collectGaugeMetrics(t, leaderInfo)
		if len(leaderMetrics) != 1 {
			t.Fatalf("leaderInfo metric count: got %d, want 1", len(leaderMetrics))
		}
		if leaderMetrics[0].labels["holder_identity"] != "pod-a" {
			t.Fatalf("holder_identity: got %q, want %q", leaderMetrics[0].labels["holder_identity"], "pod-a")
		}
		if leaderMetrics[0].value != 1 {
			t.Fatalf("leader_info value: got %v, want 1", leaderMetrics[0].value)
		}
	})

	t.Run("switches to no leader state", func(t *testing.T) {
		UpdateLeaderMetrics("luckin", "luckin-admission-webhook-leader", "pod-a")
		UpdateLeaderMetrics("luckin", "luckin-admission-webhook-leader", "")

		hasLeader := getGaugeValueWithLabels(t, hasLeader, prometheus.Labels{
			"namespace": "luckin",
			"lease":     "luckin-admission-webhook-leader",
		})
		if hasLeader != 0 {
			t.Fatalf("hasLeader: got %v, want 0", hasLeader)
		}

		leaderMetrics := collectGaugeMetrics(t, leaderInfo)
		if len(leaderMetrics) != 1 {
			t.Fatalf("leaderInfo metric count: got %d, want 1", len(leaderMetrics))
		}
		if leaderMetrics[0].labels["holder_identity"] != "" {
			t.Fatalf("holder_identity: got %q, want empty", leaderMetrics[0].labels["holder_identity"])
		}
		if leaderMetrics[0].value != 1 {
			t.Fatalf("leader_info value: got %v, want 1", leaderMetrics[0].value)
		}
	})
}

func TestHandler(t *testing.T) {
	handler := Handler()
	if handler == nil {
		t.Error("Handler() returned nil")
	}
}

// Helper to get gauge value
func getGaugeValue(t *testing.T, gauge *prometheus.GaugeVec, label string) float64 {
	t.Helper()

	metric, err := gauge.GetMetricWithLabelValues(label)
	if err != nil {
		t.Fatalf("Failed to get metric: %v", err)
	}

	var m dto.Metric
	if err := metric.Write(&m); err != nil {
		t.Fatalf("Failed to write metric: %v", err)
	}

	return m.GetGauge().GetValue()
}

func getGaugeValueWithLabels(t *testing.T, gauge *prometheus.GaugeVec, labels prometheus.Labels) float64 {
	t.Helper()

	metric, err := gauge.GetMetricWith(labels)
	if err != nil {
		t.Fatalf("Failed to get metric: %v", err)
	}

	var m dto.Metric
	if err := metric.Write(&m); err != nil {
		t.Fatalf("Failed to write metric: %v", err)
	}

	return m.GetGauge().GetValue()
}

type collectedGaugeMetric struct {
	labels map[string]string
	value  float64
}

func collectGaugeMetrics(t *testing.T, gauge *prometheus.GaugeVec) []collectedGaugeMetric {
	t.Helper()

	ch := make(chan prometheus.Metric, 16)
	gauge.Collect(ch)
	close(ch)

	var metrics []collectedGaugeMetric
	for metric := range ch {
		var m dto.Metric
		if err := metric.Write(&m); err != nil {
			t.Fatalf("Failed to write metric: %v", err)
		}

		labels := make(map[string]string, len(m.Label))
		for _, label := range m.Label {
			labels[label.GetName()] = label.GetValue()
		}
		metrics = append(metrics, collectedGaugeMetric{
			labels: labels,
			value:  m.GetGauge().GetValue(),
		})
	}

	return metrics
}

// Helper to create a test certificate
func createTestCert(t *testing.T, notBefore, notAfter time.Time) *x509.Certificate {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test",
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}

	return cert
}
