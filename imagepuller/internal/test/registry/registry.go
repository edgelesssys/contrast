// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package registry

import (
	"fmt"
	"net/http"
)

// Registry implements an OCI image registry that deliberately serves bad content.
// Only instances created with New are populated with the expected content.
type Registry struct {
	http.Handler

	indices   map[string][]byte
	manifests map[string][]byte
	blobs     map[string][]byte
}

// New creates a Registry and some indices, manifests and blobs for testing.
func New() *Registry {
	blob := blob()
	config := digested(config)
	blobs := map[string][]byte{
		blob.digest():     blob,
		WrongBlobDigest(): blob,
		config.digest():   config,
	}

	manifest := manifest()
	wrongBlobManifest := manifestForWrongBlob()
	// TODO(burgerdev): wrongConfigManifest
	manifests := map[string][]byte{
		manifest.digest():          manifest,
		WrongManifestDigest():      manifest,
		wrongBlobManifest.digest(): wrongBlobManifest,
	}

	index := index()
	wrongManifestIndex := indexForWrongManifest()
	wrongBlobIndex := indexForWrongBlob()
	indices := map[string][]byte{
		index.digest():              index,
		wrongManifestIndex.digest(): wrongManifestIndex,
		wrongBlobIndex.digest():     wrongBlobIndex,
		WrongIndexDigest():          index,
	}

	mux := http.NewServeMux()
	r := &Registry{
		Handler: mux,

		blobs:     blobs,
		manifests: manifests,
		indices:   indices,
	}
	mux.HandleFunc("/v2/{repo}/manifests/{digest}", r.handleManifest)
	mux.HandleFunc("/v2/{repo}/blobs/{digest}", r.handleBlob)
	mux.HandleFunc("/v2/", v2Handler)

	return r
}

func (r *Registry) handleManifest(rw http.ResponseWriter, req *http.Request) {
	const (
		contentTypeManifest = "application/vnd.docker.distribution.manifest.v2+json"
		contentTypeIndex    = "application/vnd.oci.image.index.v1+json"
	)

	digest := req.PathValue("digest")
	rw.Header().Set("docker-content-digest", digest)
	if data, ok := r.manifests[digest]; ok {
		rw.Header().Set("Content-Type", contentTypeManifest)
		rw.Header().Set("Content-Length", fmt.Sprint(len(data)))
		_, _ = rw.Write(data)
		return
	} else if data, ok := r.indices[digest]; ok {
		rw.Header().Set("Content-Type", contentTypeIndex)
		rw.Header().Set("Content-Length", fmt.Sprint(len(data)))
		_, _ = rw.Write(data)
		return
	}

	rw.WriteHeader(http.StatusNotFound)
}

func (r *Registry) handleBlob(rw http.ResponseWriter, req *http.Request) {
	digest := req.PathValue("digest")
	blob, ok := r.blobs[digest]
	if !ok {
		rw.WriteHeader(http.StatusNotFound)
		return
	}
	rw.Header().Set("docker-content-digest", digest)
	rw.Header().Set("Content-Length", fmt.Sprint(len(blob)))
	_, _ = rw.Write(blob)
}

func v2Handler(rw http.ResponseWriter, _ *http.Request) {
	rw.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	rw.WriteHeader(http.StatusOK)
}
