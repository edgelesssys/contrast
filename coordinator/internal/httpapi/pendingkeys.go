// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"crypto/hpke"
	"sync"
	"time"

	apitypesv1 "github.com/edgelesssys/contrast/apitypes/apiv1"
)

const (
	// pendingKeyTTL is how long the Coordinator keeps an HPKE private key for a follow-up request.
	pendingKeyTTL = 2 * time.Minute
	// pendingKeysMax bounds the number of HPKE private keys kept at the same time.
	pendingKeysMax = 1024
)

// PendingKeys holds the ephemeral HPKE private keys the Coordinator handed out the public key of,
// until the follow-up request arrives that is sealed to it.
//
// Keys are identified by the nonce of the exchange they were generated for.
// The cache is bounded in both lifetime and size, and it isn't shared between Coordinator instances.
// Losing a key is therefore expected and benign: the follow-up request fails, and the client repeats the exchange.
type PendingKeys struct {
	ttl time.Duration
	max int
	now func() time.Time

	mu   sync.Mutex
	keys map[[apitypesv1.NonceSize]byte]pendingKey
}

type pendingKey struct {
	key     hpke.PrivateKey
	expires time.Time
}

// NewPendingKeys returns an empty [PendingKeys].
func NewPendingKeys() *PendingKeys {
	return &PendingKeys{
		ttl:  pendingKeyTTL,
		max:  pendingKeysMax,
		now:  time.Now,
		keys: make(map[[apitypesv1.NonceSize]byte]pendingKey),
	}
}

// Put stores the key generated for the exchange with the given nonce.
//
// A key already stored for that nonce is replaced. If the cache is full, the key expiring first is dropped.
func (p *PendingKeys) Put(nonce [apitypesv1.NonceSize]byte, key hpke.PrivateKey) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// The cache is small, so walking it is cheap next to issuing the attestation report each key comes with.
	now := p.now()
	delete(p.keys, nonce)
	var oldest [apitypesv1.NonceSize]byte
	var oldestExpires time.Time
	for candidate, pending := range p.keys {
		if !pending.expires.After(now) {
			delete(p.keys, candidate)
			continue
		}
		if oldestExpires.IsZero() || pending.expires.Before(oldestExpires) {
			oldest, oldestExpires = candidate, pending.expires
		}
	}
	// At most one key is added per call, so dropping one keeps the cache within its bound.
	if len(p.keys) >= p.max {
		delete(p.keys, oldest)
	}
	p.keys[nonce] = pendingKey{key: key, expires: now.Add(p.ttl)}
}

// Take returns the key generated for the exchange with the given nonce, and removes it from the cache.
//
// A key can thus be used for a single follow-up request only.
// It returns false if there is no such key, because it expired, was dropped, was used already, or never existed.
func (p *PendingKeys) Take(nonce [apitypesv1.NonceSize]byte) (hpke.PrivateKey, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	pending, ok := p.keys[nonce]
	if !ok {
		return nil, false
	}
	delete(p.keys, nonce)
	if !pending.expires.After(p.now()) {
		return nil, false
	}
	return pending.key, true
}
