package metrics

import (
	"crypto/x509"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	namespace = "admission_webhook"
	subsystem = "certificate"
)

var (
	// certExpiryTimestamp is a gauge that tracks the expiry timestamp of certificates.
	certExpiryTimestamp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "expiry_timestamp_seconds",
			Help:      "The expiry timestamp of the certificate in seconds since epoch.",
		},
		[]string{"type"}, // "ca" or "serving"
	)

	// certNotBeforeTimestamp is a gauge that tracks the not-before timestamp of certificates.
	certNotBeforeTimestamp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "not_before_timestamp_seconds",
			Help:      "The not-before timestamp of the certificate in seconds since epoch.",
		},
		[]string{"type"},
	)

	// certValidDurationSeconds is a gauge that tracks the total valid duration of certificates.
	certValidDurationSeconds = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "valid_duration_seconds",
			Help:      "The total valid duration of the certificate in seconds.",
		},
		[]string{"type"},
	)

	leaderInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "leader_info",
			Help:      "Current leader identity for a lease. holder_identity is empty when no leader is held.",
		},
		[]string{"namespace", "lease", "holder_identity"},
	)

	hasLeader = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "has_leader",
			Help:      "Whether a lease currently has a holder identity.",
		},
		[]string{"namespace", "lease"},
	)

	registerOnce  sync.Once
	leaderStateMu sync.Mutex
	leaderStates  = map[string]string{}
)

// Register registers all certificate metrics with the default registry.
func Register() {
	registerOnce.Do(func() {
		prometheus.MustRegister(certExpiryTimestamp)
		prometheus.MustRegister(certNotBeforeTimestamp)
		prometheus.MustRegister(certValidDurationSeconds)
		prometheus.MustRegister(leaderInfo)
		prometheus.MustRegister(hasLeader)
	})
}

// UpdateCertMetrics updates metrics for a certificate.
func UpdateCertMetrics(certType string, cert *x509.Certificate) {
	if cert == nil {
		return
	}

	certExpiryTimestamp.WithLabelValues(certType).Set(float64(cert.NotAfter.Unix()))
	certNotBeforeTimestamp.WithLabelValues(certType).Set(float64(cert.NotBefore.Unix()))
	certValidDurationSeconds.WithLabelValues(certType).Set(cert.NotAfter.Sub(cert.NotBefore).Seconds())
}

// UpdateLeaderMetrics updates leader metrics from the current lease holder state.
func UpdateLeaderMetrics(namespace, lease, holderIdentity string) {
	key := namespace + "/" + lease

	leaderStateMu.Lock()
	defer leaderStateMu.Unlock()

	if previous, ok := leaderStates[key]; ok && previous != holderIdentity {
		leaderInfo.DeleteLabelValues(namespace, lease, previous)
	}

	leaderInfo.WithLabelValues(namespace, lease, holderIdentity).Set(1)
	if holderIdentity == "" {
		hasLeader.WithLabelValues(namespace, lease).Set(0)
	} else {
		hasLeader.WithLabelValues(namespace, lease).Set(1)
	}
	leaderStates[key] = holderIdentity
}

func resetLeaderMetrics() {
	leaderStateMu.Lock()
	defer leaderStateMu.Unlock()

	leaderInfo.Reset()
	hasLeader.Reset()
	leaderStates = map[string]string{}
}

// Handler returns an HTTP handler for the metrics endpoint.
func Handler() http.Handler {
	return promhttp.Handler()
}
