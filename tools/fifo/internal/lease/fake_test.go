// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package lease

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	coordinationv1 "k8s.io/api/coordination/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coordinationtypesv1 "k8s.io/client-go/kubernetes/typed/coordination/v1"
)

type fakeClient struct {
	leases map[string]*coordinationv1.Lease
	mu     sync.RWMutex
	coordinationtypesv1.LeaseInterface
}

func (c *fakeClient) Create(_ context.Context, lease *coordinationv1.Lease, _ metav1.CreateOptions) (*coordinationv1.Lease, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	name := lease.Name
	if lease.Name == "" && lease.GenerateName != "" {
		name = fmt.Sprintf("%s%d", lease.GenerateName, rand.Int64())
	}
	if _, ok := c.leases[name]; ok {
		return nil, errAlreadyExists
	}
	out := lease.DeepCopy()
	out.Name = name
	out.ResourceVersion = time.Now().Format(time.RFC3339Nano)
	if c.leases == nil {
		c.leases = make(map[string]*coordinationv1.Lease)
	}
	c.leases[out.Name] = out
	return out, nil
}

func (c *fakeClient) Update(_ context.Context, lease *coordinationv1.Lease, _ metav1.UpdateOptions) (*coordinationv1.Lease, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	existing, ok := c.leases[lease.Name]
	if !ok {
		return nil, fmt.Errorf("Lease %q not found: %w", lease.Name, errNotFound)
	}
	if existing.ResourceVersion != lease.ResourceVersion {
		return nil, fmt.Errorf("conflict: %w", errConflict)
	}
	out := lease.DeepCopy()
	out.ResourceVersion = time.Now().Format(time.RFC3339Nano)
	c.leases[out.Name] = out
	return out, nil
}

func (c *fakeClient) Delete(_ context.Context, name string, _ metav1.DeleteOptions) error {
	c.mu.Lock()
	delete(c.leases, name)
	c.mu.Unlock()
	return nil
}

func (c *fakeClient) Get(_ context.Context, name string, _ metav1.GetOptions) (*coordinationv1.Lease, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out, ok := c.leases[name]
	if !ok {
		return nil, fmt.Errorf("lease %q not found: %w", name, errNotFound)
	}
	return out, nil
}

func (c *fakeClient) List(_ context.Context, opts metav1.ListOptions) (*coordinationv1.LeaseList, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	list := &coordinationv1.LeaseList{}
	for _, lease := range c.leases {
		// TODO(burgerdev): we should actually be looking at the label selector, but parsing the string is tedious so we fake it here.
		if opts.LabelSelector != "" && len(lease.Labels) == 0 {
			continue
		}
		list.Items = append(list.Items, *lease)
	}
	return list, nil
}

var (
	errAlreadyExists = &k8serrors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonAlreadyExists}}
	errNotFound      = &k8serrors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonNotFound}}
	errConflict      = &k8serrors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonConflict}}
)

func TestErrors(t *testing.T) {
	assert := assert.New(t)
	assert.True(k8serrors.IsAlreadyExists(errAlreadyExists))
	assert.True(k8serrors.IsNotFound(errNotFound))
	assert.True(k8serrors.IsConflict(errConflict))
}
