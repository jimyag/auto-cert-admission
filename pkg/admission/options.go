package admission

import "time"

// Option is a functional option for configuring the webhook server.
type Option func(*ServerConfig)

// WithNamespace sets the namespace where the webhook is deployed.
func WithNamespace(namespace string) Option {
	return func(c *ServerConfig) {
		c.Namespace = namespace
	}
}

// WithServiceName sets the Kubernetes service name for the webhook.
func WithServiceName(name string) Option {
	return func(c *ServerConfig) {
		c.ServiceName = name
	}
}

// WithPort sets the port the webhook server listens on.
func WithPort(port int) Option {
	return func(c *ServerConfig) {
		c.Port = port
	}
}

// WithMetricsEnabled enables or disables the metrics server.
func WithMetricsEnabled(enabled bool) Option {
	return func(c *ServerConfig) {
		c.MetricsEnabled = enabled
	}
}

// WithMetricsPort sets the port for the metrics server.
func WithMetricsPort(port int) Option {
	return func(c *ServerConfig) {
		c.MetricsPort = port
	}
}

// WithHealthzPath sets the path for health check endpoint.
func WithHealthzPath(path string) Option {
	return func(c *ServerConfig) {
		c.HealthzPath = path
	}
}

// WithReadyzPath sets the path for readiness check endpoint.
func WithReadyzPath(path string) Option {
	return func(c *ServerConfig) {
		c.ReadyzPath = path
	}
}

// WithCAValidity sets the validity duration of the CA certificate.
func WithCAValidity(d time.Duration) Option {
	return func(c *ServerConfig) {
		c.CAValidity = d
	}
}

// WithCARefresh sets the refresh interval for the CA certificate.
func WithCARefresh(d time.Duration) Option {
	return func(c *ServerConfig) {
		c.CARefresh = d
	}
}

// WithCertValidity sets the validity duration of the server certificate.
func WithCertValidity(d time.Duration) Option {
	return func(c *ServerConfig) {
		c.CertValidity = d
	}
}

// WithCertRefresh sets the refresh interval for the server certificate.
func WithCertRefresh(d time.Duration) Option {
	return func(c *ServerConfig) {
		c.CertRefresh = d
	}
}

// WithLeaderElection enables or disables leader election.
func WithLeaderElection(enabled bool) Option {
	return func(c *ServerConfig) {
		c.LeaderElection = enabled
	}
}

// WithLeaderElectionID sets the name of the lease resource for leader election.
func WithLeaderElectionID(id string) Option {
	return func(c *ServerConfig) {
		c.LeaderElectionID = id
	}
}

// WithLeaseDuration sets the duration of the leader election lease.
func WithLeaseDuration(d time.Duration) Option {
	return func(c *ServerConfig) {
		c.LeaseDuration = d
	}
}

// WithRenewDeadline sets the deadline for renewing the leader election lease.
func WithRenewDeadline(d time.Duration) Option {
	return func(c *ServerConfig) {
		c.RenewDeadline = d
	}
}

// WithRetryPeriod sets the period between leader election retries.
func WithRetryPeriod(d time.Duration) Option {
	return func(c *ServerConfig) {
		c.RetryPeriod = d
	}
}

// WithCASecretName sets the name of the secret containing the CA certificate.
func WithCASecretName(name string) Option {
	return func(c *ServerConfig) {
		c.CASecretName = name
	}
}

// WithCertSecretName sets the name of the secret containing the server certificate.
func WithCertSecretName(name string) Option {
	return func(c *ServerConfig) {
		c.CertSecretName = name
	}
}

// WithCABundleConfigMapName sets the name of the configmap containing the CA bundle.
func WithCABundleConfigMapName(name string) Option {
	return func(c *ServerConfig) {
		c.CABundleConfigMapName = name
	}
}
