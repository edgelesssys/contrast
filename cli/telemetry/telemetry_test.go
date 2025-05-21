// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package telemetry

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func newTestClient(fn roundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func TestSendTelemetry(t *testing.T) {
	rootTestCmd := &cobra.Command{Version: "0.0.0-dev"}
	goodTestCmd := &cobra.Command{}
	rootTestCmd.AddCommand(goodTestCmd)

	testCases := map[string]struct {
		cmd                *cobra.Command
		cmdErr             error
		serverResponseCode int
		wantError          bool
	}{
		"success no cmdError": {
			cmd:                goodTestCmd,
			cmdErr:             nil,
			serverResponseCode: http.StatusOK,
			wantError:          false,
		},
		"success with cmdError": {
			cmd:                goodTestCmd,
			cmdErr:             fmt.Errorf("test error"),
			serverResponseCode: http.StatusOK,
			wantError:          false,
		},
		"bad http response": {
			cmd:                goodTestCmd,
			cmdErr:             nil,
			serverResponseCode: http.StatusInternalServerError,
			wantError:          true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			client := &Client{
				httpClient: newTestClient(func(_ *http.Request) *http.Response {
					return &http.Response{
						StatusCode: tc.serverResponseCode,
					}
				}),
			}

			err := client.SendTelemetry(t.Context(), tc.cmd, tc.cmdErr)

			if tc.wantError {
				assert.Error(err)
				return
			}
			assert.NoError(err)
		})
	}
}
