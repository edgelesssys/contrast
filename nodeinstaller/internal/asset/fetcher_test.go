// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package asset

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/edgelesssys/contrast/nodeinstaller/internal/fileop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestFetch(t *testing.T) {
	sourceDir := t.TempDir()
	fooSource := filepath.Join(sourceDir, "foo")
	require.NoError(t, os.WriteFile(fooSource, []byte("foo"), 0o644))
	defer os.RemoveAll(sourceDir)

	testCases := map[string]struct {
		dstContent   []byte
		sourceURIs   []string
		sri          string
		wantModified bool
		wantErr      bool
	}{
		"identical": {
			dstContent: []byte("foo"),
			sourceURIs: []string{"file://" + fooSource, "http://example.com/foo"},
			sri:        "sha256-LCa0a2j/xo/5m0U8HTBBNBNCLXBkg7+g+YpeiGJm564=",
		},
		"different": {
			dstContent:   []byte("bar"),
			sourceURIs:   []string{"file://" + fooSource, "http://example.com/foo"},
			sri:          "sha256-LCa0a2j/xo/5m0U8HTBBNBNCLXBkg7+g+YpeiGJm564=",
			wantModified: true,
		},
		"sri mismatch": {
			dstContent: []byte("foo"),
			sourceURIs: []string{"file://" + fooSource, "http://example.com/foo"},
			sri:        "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
			wantErr:    true,
		},
		"unchecked": {
			dstContent:   []byte("bar"),
			sourceURIs:   []string{"file://" + fooSource, "http://example.com/foo"},
			wantModified: true,
		},
		"src not found": {
			dstContent: []byte("foo"),
			sourceURIs: []string{"file://this/file/is/nonexistent", "http://example.com//this/file/is/nonexistent"},
			wantErr:    true,
		},
		"dst not found": {
			sourceURIs:   []string{"file://" + fooSource, "http://example.com/foo"},
			wantModified: true,
		},
	}
	for name, tc := range testCases {
		for _, sourceURI := range tc.sourceURIs {
			t.Run(name+"_"+sourceURI, func(t *testing.T) {
				assert := assert.New(t)
				require := require.New(t)

				emptyDir := t.TempDir()
				defer os.RemoveAll(emptyDir)
				dst := filepath.Join(emptyDir, "dst")
				if tc.dstContent != nil {
					require.NoError(os.WriteFile(dst, tc.dstContent, 0o644))
					defer os.Remove(dst)
				}
				httpResponses := map[string][]byte{
					"/foo": []byte("foo"),
				}
				fetcher := NewFetcher(map[string]handler{
					"file": NewFileFetcher(),
					"http": newFakeHTTPFetcher(httpResponses),
				})
				var modified bool
				var fetchErr error
				if tc.sri != "" {
					modified, fetchErr = fetcher.Fetch(context.Background(), sourceURI, dst, tc.sri)
				} else {
					modified, fetchErr = fetcher.FetchUnchecked(context.Background(), sourceURI, dst)
				}
				if tc.wantErr {
					require.Error(fetchErr)
					return
				}
				require.NoError(fetchErr)
				assert.Equal(tc.wantModified, modified)
				got, err := os.ReadFile(dst)
				require.NoError(err)
				assert.Equal([]byte("foo"), got)
			})
		}
	}
}

func newFakeHTTPFetcher(responses map[string][]byte) *HTTPFetcher {
	hClient := http.Client{
		Transport: &stubRoundTripper{store: responses},
	}

	return &HTTPFetcher{
		client: &hClient,
		mover:  fileop.NewDefault(),
	}
}

type stubRoundTripper struct {
	store map[string][]byte
}

func (f *stubRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	path := req.URL.Path
	body, ok := f.store[path]
	if !ok {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(bytes.NewReader([]byte("not found"))),
		}, nil
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}, nil
}
