package admission

import (
	"os"
	"time"

	"github.com/jimyag/auto-cert-webhook/pkg/webhook"
)

// ServerConfig holds the internal server configuration.
type ServerConfig struct {
	// Name is the webhook name.
	Name string

	// Namespace is the namespace where the webhook is deployed.
	Namespace string

	// ServiceName is the name of the Kubernetes service for the webhook.
	ServiceName string

	// Port is the port the webhook server listens on.
	Port int

	// MetricsEnabled enables the metrics server.
	MetricsEnabled bool

	// MetricsPort is the port for the metrics server.
	MetricsPort int

	// HealthzPath is the path for health check endpoint.
	HealthzPath string

	// ReadyzPath is the path for readiness check endpoint.
	ReadyzPath string

	// MetricsPath is the path for metrics endpoint.
	MetricsPath string

	// CASecretName is the name of the secret containing the CA certificate.
	CASecretName string

	// CertSecretName is the name of the secret containing the server certificate.
	CertSecretName string

	// CABundleConfigMapName is the name of the configmap containing the CA bundle.
	CABundleConfigMapName string

	// CAValidity is the validity duration of the CA certificate.
	CAValidity time.Duration

	// CARefresh is the refresh interval for the CA certificate.
	CARefresh time.Duration

	// CertValidity is the validity duration of the server certificate.
	CertValidity time.Duration

	// CertRefresh is the refresh interval for the server certificate.
	CertRefresh time.Duration

	// LeaderElection enables leader election for certificate rotation.
	LeaderElection bool

	// LeaderElectionID is the name of the lease resource for leader election.
	LeaderElectionID string

	// LeaseDuration is the duration of the leader election lease.
	LeaseDuration time.Duration

	// RenewDeadline is the deadline for renewing the leader election lease.
	RenewDeadline time.Duration

	// RetryPeriod is the period between leader election retries.
	RetryPeriod time.Duration
}

const (
	DefaultNamespace             = "default"
	DefaultPort                  = 8443
	DefaultMetricsEnabled        = true
	DefaultMetricsPort           = 8080
	DefaultMetricsPath           = "/metrics"
	DefaultHealthzPath           = "/healthz"
	DefaultReadyzPath            = "/readyz"
	DefaultCAValidity            = 2 * 24 * time.Hour
	DefaultCARefresh             = 1 * 24 * time.Hour
	DefaultCertValidity          = 1 * 24 * time.Hour
	DefaultCertRefresh           = 12 * time.Hour
	DefaultLeaderElection        = true
	DefaultLeaseDuration         = 30 * time.Second
	DefaultRenewDeadline         = 10 * time.Second
	DefaultRetryPeriod           = 5 * time.Second
)

// DefaultServerConfig returns a ServerConfig with default values.
func DefaultServerConfig() *ServerConfig {
	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		namespace = DefaultNamespace
	}

	return &ServerConfig{
		Namespace:      namespace,
		Port:           DefaultPort,
		MetricsEnabled: DefaultMetricsEnabled,
		MetricsPort:    DefaultMetricsPort,
		HealthzPath:    DefaultHealthzPath,
		ReadyzPath:     DefaultReadyzPath,
		MetricsPath:    DefaultMetricsPath,
		CAValidity:     DefaultCAValidity,
		CARefresh:      DefaultCARefresh,
		CertValidity:   DefaultCertValidity,
		CertRefresh:    DefaultCertRefresh,
		LeaderElection: DefaultLeaderElection,
		LeaseDuration:  DefaultLeaseDuration,
		RenewDeadline:  DefaultRenewDeadline,
		RetryPeriod:    DefaultRetryPeriod,
	}
}

// ApplyUserConfig applies user-provided webhook.Config to the server config.
func (c *ServerConfig) ApplyUserConfig(userCfg webhook.Config) {
	if userCfg.Name != "" {
		c.Name = userCfg.Name
		c.ServiceName = userCfg.Name
		c.CASecretName = userCfg.Name + "-ca"
		c.CertSecretName = userCfg.Name + "-cert"
		c.CABundleConfigMapName = userCfg.Name + "-ca-bundle"
		c.LeaderElectionID = userCfg.Name + "-leader"
	}

	if userCfg.Namespace != "" {
		c.Namespace = userCfg.Namespace
	}

	if userCfg.ServiceName != "" {
		c.ServiceName = userCfg.ServiceName
	}

	if userCfg.Port > 0 {
		c.Port = userCfg.Port
	}

	if userCfg.MetricsPort > 0 {
		c.MetricsPort = userCfg.MetricsPort
	}

	if userCfg.MetricsEnabled != nil {
		c.MetricsEnabled = *userCfg.MetricsEnabled
	}

	if userCfg.LeaderElection != nil {
		c.LeaderElection = *userCfg.LeaderElection
	}
}
