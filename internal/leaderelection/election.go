package leaderelection

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2"
)

const defaultRetryPeriod = 5 * time.Second

// Config holds leader election configuration.
type Config struct {
	// Namespace is the namespace for the lease resource.
	Namespace string

	// Name is the name of the lease resource.
	Name string

	// LeaseDuration is the duration of the lease.
	LeaseDuration time.Duration

	// RenewDeadline is the deadline for renewing the lease.
	RenewDeadline time.Duration

	// RetryPeriod is the period between retries.
	RetryPeriod time.Duration
}

// Callbacks defines the callbacks for leader election events.
type Callbacks struct {
	// OnStartedLeading is called when this instance becomes the leader.
	OnStartedLeading func(ctx context.Context)

	// OnStoppedLeading is called when this instance loses leadership.
	OnStoppedLeading func()

	// OnNewLeader is called when a new leader is elected.
	OnNewLeader func(identity string)
}

var runLeaderElectionAttempt = func(
	ctx context.Context,
	client kubernetes.Interface,
	config Config,
	callbacks Callbacks,
) error {
	identity := getIdentity()

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
		},
		Client: client.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: identity,
		},
	}

	leaderElector, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:            lock,
		LeaseDuration:   config.LeaseDuration,
		RenewDeadline:   config.RenewDeadline,
		RetryPeriod:     config.RetryPeriod,
		ReleaseOnCancel: true,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				klog.Infof("Started leading as %s", identity)
				if callbacks.OnStartedLeading != nil {
					callbacks.OnStartedLeading(ctx)
				}
			},
			OnStoppedLeading: func() {
				klog.Infof("Stopped leading as %s", identity)
				if callbacks.OnStoppedLeading != nil {
					callbacks.OnStoppedLeading()
				}
			},
			OnNewLeader: func(currentLeader string) {
				if currentLeader == identity {
					return
				}
				klog.Infof("New leader elected: %s", currentLeader)
				if callbacks.OnNewLeader != nil {
					callbacks.OnNewLeader(currentLeader)
				}
			},
		},
	})
	if err != nil {
		klog.Errorf("Failed to create leader elector: %v", err)
		return err
	}

	klog.Infof("Starting leader election with identity %s", identity)
	leaderElector.Run(ctx)
	return nil
}

// Run runs the leader election with the given callbacks.
func Run(ctx context.Context, client kubernetes.Interface, config Config, callbacks Callbacks) error {
	if config.RetryPeriod <= 0 {
		config.RetryPeriod = defaultRetryPeriod
	}

	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		if err := runLeaderElectionAttempt(ctx, client, config, callbacks); err != nil {
			return err
		}

		if err := ctx.Err(); err != nil {
			return nil
		}

		klog.Warning("Leader election stopped without context cancellation, retrying")

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(config.RetryPeriod):
		}
	}
}

// getIdentity returns the identity for this instance.
func getIdentity() string {
	// Try to get pod name from environment
	identity := os.Getenv("POD_NAME")
	if identity != "" {
		return identity
	}

	// Fall back to hostname
	hostname, err := os.Hostname()
	if err != nil {
		klog.Errorf("Failed to get hostname: %v, generating random identity", err)
		hostname = "unknown-" + randomSuffix()
	}
	return hostname
}

// randomSuffix generates a random 8-character hex string.
func randomSuffix() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// Fallback to a fixed value if random fails (very unlikely)
		return "00000000"
	}
	return hex.EncodeToString(b)
}
