// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// package lease implements a FIFO queue backed by Kubernetes Lease objects.
//
// There are two types of Lease objects this package is managing:
//
// - A lease representing a shared resource of the same name.
//
//	apiVersion: coordination.k8s.io/v1
//	kind: Lease
//	metadata:
//	  name: gpu-0
//	  labels: {}
//
// - Multiple leases representing processes queueing for the resource mentioned in the labels.
//
//	apiVersion: coordination.k8s.io/v1
//	kind: Lease
//	metadata:
//	  name: gpu-0-q-7jc79
//	  labels:
//	    ci.contrast.edgeless.systems/fifo-lease: gpu-0
//
// Candidates choose their unique name, which will be placed into the HolderIdentity field after
// successful acquisition. The queue entries are ephemeral and deleted when the process acquires
// the lease or gives up.
//
// It's safe to delete any queue entries, the waiting process discovers this situation and fails.
// The main Lease item can be deleted if the current holder does not block the resource anymore.
package lease

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	coordinationtypesv1 "k8s.io/client-go/kubernetes/typed/coordination/v1"
	"k8s.io/utils/clock"
)

const (
	labelKey     = "ci.contrast.edgeless.systems/fifo-lease"
	pollInterval = 10 * time.Second

	// queueEntryTTL is the interval in which a queue entry is considered alive, starting from its renewTime.
	queueEntryTTL = int32(60)
	// queueEntryRefreshInterval is the period with which active queue entries update their renewTime.
	// It's deliberately chosen such that 3 failures to refresh result in queue eviction.
	queueEntryRefreshInterval = 19 * time.Second

	unknownHolder = "<unknown>"
)

// Lease manages access to a shared resource with mutual exclusion.
//
// Instances need to be constructed with New. Every call to Acquire must be matched with a call to
// Release, the lease is not re-entrant and releasing it is not idempotent.
type Lease struct {
	name                 string
	holderIdentity       string
	client               coordinationtypesv1.LeaseInterface
	log                  *slog.Logger
	leaseDurationSeconds int32

	entry        *coordinationv1.Lease
	pollInterval time.Duration
	clock        clock.Clock
}

// New creates a new Lease object.
//
// * leaseName refers to the shared resource you want to lease, for example "gpu-0".
// * holderIdentity refers to yourself, and must be unique among contenders.
// * leaseDuration is the maximum amount of time you're planning to use the shared resource.
func New(leaseName, holderIdentity string, leaseDuration time.Duration, client coordinationtypesv1.LeaseInterface, log *slog.Logger) *Lease {
	return &Lease{
		client:               client,
		log:                  log,
		name:                 leaseName,
		holderIdentity:       holderIdentity,
		leaseDurationSeconds: int32(leaseDuration.Seconds()),
		pollInterval:         pollInterval,
		clock:                clock.RealClock{},
	}
}

// Acquire tries to get exclusive access to the shared resource.
//
// If Acquire returns without an error, access is granted and you MUST call Release eventually.
// If Acquire returns with an error, it's because the context expired before the resource could be
// leased. In this case, you don't have access and MUST NOT call Release.
func (l *Lease) Acquire(ctx context.Context) error {
	if err := l.createQueueEntry(ctx); err != nil {
		return fmt.Errorf("creating queue entry: %w", err)
	}
	l.log.Debug("Joined queue", "name", l.entry.Name)

	// Refresh the queue entry until we acquire the lease or give up.
	ctx, cancel := context.WithCancel(ctx)
	doneCh := make(chan struct{})
	go func() {
		l.refreshQueueEntry(ctx)
		close(doneCh)
	}()
	defer func() {
		cancel()
		<-doneCh
	}()

	if err := wait.PollUntilContextCancel(ctx, l.pollInterval, true /*immediate*/, l.tryAcquire); err != nil {
		return fmt.Errorf("waiting for lease: %w", err)
	}
	l.log.Debug("Acquired lease", "name", l.name, "holder-identity", l.holderIdentity)
	return nil
}

// Release releases the formerly Acquired shared resource.
// You must only call it once, and only after successful Acquisition.
// Errors may or may not be retriable.
func (l *Lease) Release(ctx context.Context) error {
	existing, err := l.client.Get(ctx, l.name, metav1.GetOptions{})
	if err != nil {
		l.log.Debug("Couldn't delete lease", "name", l.name)
		return fmt.Errorf("getting existing lease: %w", err)
	}
	currentHolder := orDefault(existing.Spec.HolderIdentity, unknownHolder)
	if currentHolder != l.holderIdentity {
		return fmt.Errorf("lease is held by %q, expected %q", currentHolder, l.holderIdentity)
	}

	if err := l.client.Delete(ctx, l.name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("deleting lease: %w", err)
	}
	l.log.Debug("Deleted lease", "name", l.name)

	return nil
}

func (l *Lease) tryAcquire(ctx context.Context) (done bool, err error) {
	// Fetch the full queue
	list, err := l.client.List(ctx, metav1.ListOptions{
		LabelSelector: labelKey + "=" + l.name,
	})
	if err != nil {
		// We don't want to stop waiting on transient errors, just log the failure.
		l.log.Warn("Failed listing queue entries", "error", err)
		return false, nil
	}

	// Determine front of the queue.
	var (
		holder   string
		now      = l.clock.Now()
		earliest = now.Add(time.Hour) // arbitrary point in the future
	)
	// Sorting makes queue position deterministic even if the AcquireTimes are identical.
	slices.SortFunc(list.Items, func(a, b coordinationv1.Lease) int {
		return strings.Compare(*a.Spec.HolderIdentity, *b.Spec.HolderIdentity)
	})
	foundSelf := false
	for _, item := range list.Items {
		// Skip expired entries.
		if now.After(item.Spec.RenewTime.Add(time.Duration(*item.Spec.LeaseDurationSeconds) * time.Second)) {
			l.log.Debug("skipping expired entry", "name", item.Name, "holder-identity", orDefault(item.Spec.HolderIdentity, unknownHolder))
			continue
		}
		// Look for our own entry only after checking expiry: if we expired, we will never get in
		// front of the queue.
		if item.Spec.HolderIdentity != nil && *item.Spec.HolderIdentity == l.holderIdentity {
			foundSelf = true
		}
		// Skip entries newer than the current front.
		if item.Spec.AcquireTime.After(earliest) {
			l.log.Debug("skipping older entry", "name", item.Name)
			continue
		}
		earliest = item.Spec.AcquireTime.Time
		holder = *item.Spec.HolderIdentity
	}
	if !foundSelf {
		// The most likely reason for this is that somebody deleted all leases. In that case, we
		// should not continue waiting, but fail.
		return false, fmt.Errorf("our queue entry vanished or expired")
	}

	if holder != l.holderIdentity {
		l.log.Info("waiting for lease", "lease", l.name, "front-of-queue", holder)
		return false, nil
	}
	l.log.Debug("We're in front of the queue", "lease", l.name)

	return l.claimLease(ctx)
}

// claimLease tries to update the holder of the shared lease.
//
// If the function returns (true, nil), the lease was updated successfully.
// - There is no matching Lease object.
// - The Lease object exists and is held by someone else, but it expired.
// This function can fail in both cases due to races, in which case acquiring the lease should be
// retried. This situation is communicated through a (false, nil) return. On k8s errors, it returns
// (false, err).
func (l *Lease) claimLease(ctx context.Context) (bool, error) {
	now := metav1.NewMicroTime(l.clock.Now())
	existing, err := l.client.Get(ctx, l.name, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return false, fmt.Errorf("getting existing lease: %w", err)
	} else if k8serrors.IsNotFound(err) {
		lease := &coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name: l.name,
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity:       &l.holderIdentity,
				AcquireTime:          &now,
				RenewTime:            &now,
				LeaseDurationSeconds: &l.leaseDurationSeconds,
			},
		}
		if _, err := l.client.Create(ctx, lease, metav1.CreateOptions{}); k8serrors.IsAlreadyExists(err) {
			l.log.Debug("Lost race to create the lease")
			return false, nil
		} else if err != nil {
			return false, fmt.Errorf("creating the lease: %w", err)
		}
		l.log.Debug("Claimed lease", "name", l.name, "RenewTime", now, "LeaseDurationSeconds", l.leaseDurationSeconds)
		return true, nil
	}
	expiry := existing.Spec.RenewTime.Add(time.Duration(*existing.Spec.LeaseDurationSeconds) * time.Second)
	if now.Time.Before(expiry) {
		l.log.Info("Lease is currently held", "holder", orDefault(existing.Spec.HolderIdentity, unknownHolder), "expires", expiry)
		return false, nil
	}
	// The lease expired, let's claim it.
	previousHolder := orDefault(existing.Spec.HolderIdentity, unknownHolder)
	l.log.Debug("Found expired lease", "name", existing.Name, "previous-holder", previousHolder, "now", now, "expiry", expiry)

	existing.Spec.LeaseTransitions = toPtr(orDefault(existing.Spec.LeaseTransitions, 0) + 1)
	existing.Spec.AcquireTime = &now
	existing.Spec.RenewTime = &now
	existing.Spec.LeaseDurationSeconds = &l.leaseDurationSeconds
	existing.Spec.HolderIdentity = &l.holderIdentity

	if _, err := l.client.Update(ctx, existing, metav1.UpdateOptions{}); k8serrors.IsConflict(err) {
		l.log.Debug("Lost race to update the lease", "name", existing.Name)
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("updating existing lease: %w", err)
	}
	l.log.Info("Claimed expired lease", "name", existing.Name, "previous-holder", previousHolder)
	return true, nil
}

func (l *Lease) createQueueEntry(ctx context.Context) error {
	if l.entry != nil {
		return fmt.Errorf("queue entry already exists")
	}

	now := metav1.NewMicroTime(l.clock.Now())

	entry := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: l.name + "-q-",
			Labels:       map[string]string{labelKey: l.name},
		},
		Spec: coordinationv1.LeaseSpec{
			HolderIdentity:       &l.holderIdentity,
			AcquireTime:          &now,
			RenewTime:            &now,
			LeaseDurationSeconds: toPtr(queueEntryTTL),
		},
	}

	created, err := l.client.Create(ctx, entry, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	l.entry = created
	l.log.Debug("Created queue entry", "name", created.Name, "holder-identity", l.holderIdentity)
	return nil
}

// refreshQueueEntry periodically updates the lease's RenewTime.
// Once the context expires, the function stops and the lease is deleted.
func (l *Lease) refreshQueueEntry(ctx context.Context) {
	defer func() { //nolint:contextcheck // We only exit when the context expired, so we need to use a fresh one to clean up.
		entry := l.entry
		l.entry = nil
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := l.client.Delete(ctx, entry.Name, metav1.DeleteOptions{}); err != nil {
			l.log.Warn("Failed to clean queue entry", "error", err)
		} else {
			l.log.Debug("Cleaned up queue entry", "name", entry.Name)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(queueEntryRefreshInterval):
		}

		now := metav1.NewMicroTime(l.clock.Now())
		entry := l.entry.DeepCopy()
		entry.Spec.RenewTime = &now

		nextEntry, err := l.client.Update(ctx, entry, metav1.UpdateOptions{})
		if err != nil {
			l.log.Error("Failed to renew lease", "name", l.entry.Name, "namespace", l.entry.Namespace, "error", err)
			continue
		}

		l.log.Debug("Refreshed queue entry", "name", entry.Name)
		l.entry = nextEntry
	}
}

func toPtr[A any](a A) *A {
	return &a
}

func orDefault[A any](a *A, def A) A {
	if a != nil {
		return *a
	}
	return def
}
