package admission

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"github.com/jimyag/auto-cert-webhook/pkg/cabundle"
	"github.com/jimyag/auto-cert-webhook/pkg/certmanager"
	"github.com/jimyag/auto-cert-webhook/pkg/certprovider"
	"github.com/jimyag/auto-cert-webhook/pkg/leaderelection"
	"github.com/jimyag/auto-cert-webhook/pkg/metrics"
	"github.com/jimyag/auto-cert-webhook/pkg/server"
	"github.com/jimyag/auto-cert-webhook/pkg/webhook"
)

// Run starts the webhook server with the given Admission implementation.
// This is the main entry point for using this library.
func Run(admission Admission, opts ...Option) error {
	// Setup signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	return RunWithContext(ctx, admission, opts...)
}

// RunWithContext starts the webhook server with the given context.
func RunWithContext(ctx context.Context, admission Admission, opts ...Option) error {
	// Get user configuration
	userConfig := admission.Configure()
	hooks := admission.Webhooks()

	if userConfig.Name == "" {
		return fmt.Errorf("webhook name is required in Configure()")
	}

	if len(hooks) == 0 {
		return fmt.Errorf("at least one webhook hook is required in Webhooks()")
	}

	// Apply default configuration
	config := DefaultServerConfig()
	config.ApplyUserConfig(userConfig)

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	klog.Infof("Starting webhook %s in namespace %s", config.Name, config.Namespace)

	// Create Kubernetes client
	k8sCfg, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	client, err := kubernetes.NewForConfig(k8sCfg)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	errCh := make(chan error, 1)

	// Determine webhook refs for CA bundle syncer
	webhookRefs := determineWebhookRefs(config.Name, hooks)

	// Create certificate provider (runs on all pods)
	certProvider := certprovider.New(client, config.Namespace, config.CertSecretName)

	// Start certificate provider in background
	go func() {
		if err := certProvider.Start(ctx); err != nil {
			klog.Errorf("Certificate provider error: %v", err)
			errCh <- err
		}
	}()

	// Create and start HTTP server (runs on all pods)
	srv := server.New(certProvider, server.Config{
		Port:        config.Port,
		HealthzPath: config.HealthzPath,
		ReadyzPath:  config.ReadyzPath,
	})

	// Register webhook handlers
	for _, hook := range hooks {
		srv.RegisterHook(hook.Path, hook.Type, hook.Admit)
		klog.Infof("Registered %s webhook at path %s", hook.Type, hook.Path)
	}

	// Start HTTP server in background
	go func() {
		if err := srv.Start(ctx); err != nil {
			klog.Errorf("Server error: %v", err)
			errCh <- err
		}
	}()

	// Start metrics server if enabled
	if config.MetricsEnabled {
		metricsSrv := metrics.NewServer(metrics.ServerConfig{
			Port: config.MetricsPort,
			Path: config.MetricsPath,
		})
		go func() {
			if err := metricsSrv.Start(ctx); err != nil {
				klog.Errorf("Metrics server error: %v", err)
				errCh <- err
			}
		}()
	}

	// Create certificate manager and CA bundle syncer (runs on leader only)
	certMgr := certmanager.New(client, certmanager.Config{
		Namespace:             config.Namespace,
		ServiceName:           config.ServiceName,
		CASecretName:          config.CASecretName,
		CertSecretName:        config.CertSecretName,
		CABundleConfigMapName: config.CABundleConfigMapName,
		CAValidity:            config.CAValidity,
		CARefresh:             config.CARefresh,
		CertValidity:          config.CertValidity,
		CertRefresh:           config.CertRefresh,
	})

	caBundleSyncer := cabundle.NewSyncer(client, config.Namespace, config.CABundleConfigMapName, webhookRefs)

	if config.LeaderElection {
		// Run with leader election
		go func() {
			if err := leaderelection.Run(ctx, client, leaderelection.Config{
				Namespace:     config.Namespace,
				Name:          config.LeaderElectionID,
				LeaseDuration: config.LeaseDuration,
				RenewDeadline: config.RenewDeadline,
				RetryPeriod:   config.RetryPeriod,
			}, leaderelection.Callbacks{
				OnStartedLeading: func(leaderCtx context.Context) {
					klog.Info("Became leader, starting certificate management")
					startCertManagement(leaderCtx, certMgr, caBundleSyncer, errCh)
				},
				OnStoppedLeading: func() {
					klog.Info("Lost leadership")
				},
			}); err != nil {
				klog.Errorf("Leader election error: %v", err)
				errCh <- err
			}
		}()
	} else {
		// Run without leader election (single replica mode)
		klog.Info("Running without leader election")
		startCertManagement(ctx, certMgr, caBundleSyncer, errCh)
	}

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		klog.Info("Shutting down")
		return nil
	case err := <-errCh:
		klog.Errorf("Error: %v", err)
		return err
	}
}

func startCertManagement(ctx context.Context, certMgr *certmanager.Manager, caBundleSyncer *cabundle.Syncer, errCh chan error) {
	go func() {
		if err := certMgr.Start(ctx); err != nil {
			klog.Errorf("Certificate manager error: %v", err)
			errCh <- err
		}
	}()

	go func() {
		if err := caBundleSyncer.Start(ctx); err != nil {
			klog.Errorf("CA bundle syncer error: %v", err)
			errCh <- err
		}
	}()
}

// determineWebhookRefs determines webhook references for CA bundle syncing.
func determineWebhookRefs(name string, hooks []webhook.Hook) []cabundle.WebhookRef {
	var refs []cabundle.WebhookRef

	hasMutating := false
	hasValidating := false

	for _, hook := range hooks {
		if hook.Type == webhook.Mutating && !hasMutating {
			refs = append(refs, cabundle.WebhookRef{
				Name: name,
				Type: cabundle.MutatingWebhook,
			})
			hasMutating = true
		}
		if hook.Type == webhook.Validating && !hasValidating {
			refs = append(refs, cabundle.WebhookRef{
				Name: name,
				Type: cabundle.ValidatingWebhook,
			})
			hasValidating = true
		}
	}

	return refs
}
