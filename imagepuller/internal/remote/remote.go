// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package remote

import (
	"github.com/google/go-containerregistry/pkg/name"
	gcr "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// DefaultRemote implements Remote, passing all function calls to the remote module.
type DefaultRemote struct{}

// Head returns a gcr.Descriptor for the given reference by issuing a HEAD request.
func (DefaultRemote) Head(ref name.Reference, opts ...remote.Option) (*gcr.Descriptor, error) {
	return remote.Head(ref, opts...)
}

// Image provides access to a remote image reference.
func (DefaultRemote) Image(ref name.Reference, opts ...remote.Option) (gcr.Image, error) {
	return remote.Image(ref, opts...)
}

// Index provides access to a remote index reference.
func (DefaultRemote) Index(ref name.Reference, opts ...remote.Option) (gcr.ImageIndex, error) {
	return remote.Index(ref, opts...)
}
