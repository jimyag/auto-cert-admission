package metrics

import (
	"context"
	"testing"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestStartLeaderObserverUpdatesMetrics(t *testing.T) {
	resetLeaderMetrics()

	client := fake.NewClientset()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- StartLeaderObserver(ctx, client, "luckin", "luckin-admission-webhook-leader")
	}()

	assertEventually(t, func() bool {
		return getGaugeValueWithLabels(t, hasLeader, map[string]string{
			"namespace": "luckin",
			"lease":     "luckin-admission-webhook-leader",
		}) == 0
	})

	holderIdentity := "pod-a"
	lease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "luckin-admission-webhook-leader",
			Namespace: "luckin",
		},
		Spec: coordinationv1.LeaseSpec{
			HolderIdentity: &holderIdentity,
		},
	}

	if _, err := client.CoordinationV1().Leases("luckin").Create(ctx, lease, metav1.CreateOptions{}); err != nil {
		t.Fatalf("create lease: %v", err)
	}

	assertEventually(t, func() bool {
		metrics := collectGaugeMetrics(t, leaderInfo)
		return len(metrics) == 1 &&
			metrics[0].labels["holder_identity"] == "pod-a" &&
			getGaugeValueWithLabels(t, hasLeader, map[string]string{
				"namespace": "luckin",
				"lease":     "luckin-admission-webhook-leader",
			}) == 1
	})

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("StartLeaderObserver() error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("observer did not stop in time")
	}
}

func assertEventually(t *testing.T, fn func() bool) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatal("condition not met before timeout")
}
