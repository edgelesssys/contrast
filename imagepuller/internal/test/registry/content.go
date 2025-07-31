// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package registry

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"
)

// BlobDigest is the digest of the blob.
func BlobDigest() string {
	return blob().digest()
}

// WrongBlobDigest is a digest for which a blob with a different digest will be served.
func WrongBlobDigest() string {
	return digested("BAD BLOB").digest()
}

// ManifestDigest is the digest of the well-formed manifest.
func ManifestDigest() string {
	return manifest().digest()
}

// WrongManifestDigest is a digest for which an image manifest with a different digest will be served.
func WrongManifestDigest() string {
	return digested("BAD MANIFEST").digest()
}

// WrongIndexDigest is a digest for which an image index with a different digest will be served.
func WrongIndexDigest() string {
	return digested("BAD INDEX").digest()
}

// ManifestForWrongBlobDigest refers to an image that's referring to WrongBlobDigest.
func ManifestForWrongBlobDigest() string {
	return manifestForWrongBlob().digest()
}

// IndexForWrongManifestDigest refers to an image index that's referring to WrongManifestDigest.
func IndexForWrongManifestDigest() string {
	return indexForWrongManifest().digest()
}

// IndexForManifestForWrongBlobDigest refers to an image index that's referring to ManifestForWrongBlobDigest.
func IndexForManifestForWrongBlobDigest() string {
	return indexForWrongBlob().digest()
}

type digested []byte

func (d digested) digest() string {
	digest := sha256.Sum256(d)
	return fmt.Sprintf("sha256:%x", digest)
}

func blob() digested {
	var buf bytes.Buffer

	gzipWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzipWriter)

	header := &tar.Header{
		Name:    "hello.txt",
		Mode:    0o644,
		Size:    0,
		ModTime: time.Unix(0, 0),
	}
	err := errors.Join(
		tarWriter.WriteHeader(header),
		tarWriter.Close(),
		gzipWriter.Close(),
	)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func manifest() digested {
	blob := blob()
	config := digested(config)
	return fmt.Appendf(nil, manifestTemplate, config.digest(), len(config), blob.digest(), len(blob))
}

func index() digested {
	manifest := manifest()
	return fmt.Appendf(nil, indexTemplate, manifest.digest(), len(manifest))
}

func manifestForWrongBlob() digested {
	config := digested(config)
	return fmt.Appendf(nil, manifestTemplate, config.digest(), len(config), WrongBlobDigest(), 0)
}

func indexForWrongManifest() digested {
	manifest := manifest()
	return fmt.Appendf(nil, indexTemplate, WrongManifestDigest(), len(manifest))
}

func indexForWrongBlob() digested {
	manifest := manifestForWrongBlob()
	return fmt.Appendf(nil, indexTemplate, manifest.digest(), len(manifest))
}

const indexTemplate = `
{
  "manifests": [
    {
      "digest": "%s",
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "platform": {
        "architecture": "amd64",
        "os": "linux"
      },
      "size": %d
    }
  ],
  "mediaType": "application/vnd.oci.image.index.v1+json",
  "schemaVersion": 2
}
`

const manifestTemplate = `
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "config": {
    "mediaType": "application/vnd.oci.image.config.v1+json",
    "digest": "%s",
    "size": %d
  },
  "layers": [
    {
      "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip",
      "digest": "%s",
      "size": %d
    }
  ]
}
`

const config = `
{
  "config": {
    "Cmd": [
      "sh"
    ],
    "Env": [
      "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
    ]
  },
  "created": "2023-05-18T22:34:17Z",
  "rootfs": {
    "type": "layers",
    "diff_ids": [
      "sha256:0000000000000000000000000000000000000000000000000000000000000000"
    ]
  },
  "architecture": "amd64",
  "os": "linux"
}
`
