package metrics

import (
	"context"
	"fmt"

	coordinationv1 "k8s.io/api/coordination/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

// StartLeaderObserver watches a lease and updates leader metrics from its holder identity.
func StartLeaderObserver(ctx context.Context, client kubernetes.Interface, namespace, leaseName string) error {
	UpdateLeaderMetrics(namespace, leaseName, "")

	if err := syncLeaderLease(ctx, client, namespace, leaseName); err != nil {
		klog.Warningf("Initial leader lease sync failed (will retry via informer): %v", err)
	}

	factory := informers.NewSharedInformerFactoryWithOptions(
		client,
		0,
		informers.WithNamespace(namespace),
	)
	leaseInformer := factory.Coordination().V1().Leases().Informer()

	_, err := leaseInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			lease, ok := obj.(*coordinationv1.Lease)
			if !ok || lease.Name != leaseName {
				return
			}
			updateLeaderMetricsFromLease(namespace, leaseName, lease)
		},
		UpdateFunc: func(_, newObj interface{}) {
			lease, ok := newObj.(*coordinationv1.Lease)
			if !ok || lease.Name != leaseName {
				return
			}
			updateLeaderMetricsFromLease(namespace, leaseName, lease)
		},
		DeleteFunc: func(obj interface{}) {
			lease, ok := obj.(*coordinationv1.Lease)
			if !ok {
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					return
				}
				lease, ok = tombstone.Obj.(*coordinationv1.Lease)
				if !ok {
					return
				}
			}
			if lease.Name == leaseName {
				UpdateLeaderMetrics(namespace, leaseName, "")
			}
		},
	})
	if err != nil {
		return fmt.Errorf("failed to add leader lease event handler: %w", err)
	}

	factory.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), leaseInformer.HasSynced) {
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("failed to sync leader lease informer cache")
	}

	<-ctx.Done()
	return nil
}

func syncLeaderLease(ctx context.Context, client kubernetes.Interface, namespace, leaseName string) error {
	lease, err := client.CoordinationV1().Leases(namespace).Get(ctx, leaseName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			UpdateLeaderMetrics(namespace, leaseName, "")
			return nil
		}
		return err
	}

	updateLeaderMetricsFromLease(namespace, leaseName, lease)
	return nil
}

func updateLeaderMetricsFromLease(namespace, leaseName string, lease *coordinationv1.Lease) {
	holderIdentity := ""
	if lease.Spec.HolderIdentity != nil {
		holderIdentity = *lease.Spec.HolderIdentity
	}
	UpdateLeaderMetrics(namespace, leaseName, holderIdentity)
}
