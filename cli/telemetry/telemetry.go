package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"runtime"

	"github.com/spf13/cobra"
)

const (
	apiHost       = "telemetry.confidential.cloud"
	telemetryPath = "api/contrast/v1"
)

// RequestV1 holds the information to be sent to the telemetry server.
type RequestV1 struct {
	Version     string `json:"version"`
	GOOS        string `json:"goos"`
	GOARCH      string `json:"goarch"`
	Cmd         string `json:"cmd"`
	CmdErrClass string `json:"cmderrclass"`
	Test        bool   `json:"test" gorm:"-"`
}

// IsTest checks if the request is used for testing.
func (r *RequestV1) IsTest() bool {
	return r.Test
}

// Client sends the telemetry.
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new Client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{},
	}
}

// SendTelemetry sends telemetry data to the telemetry server.
func (c *Client) SendTelemetry(ctx context.Context, cmd *cobra.Command, cmdErr error) error {
	cmdErrClass := classifyCmdErr(cmdErr)

	if cmd.Root().Version == "" {
		return fmt.Errorf("no cli version found")
	}

	telemetryRequest := RequestV1{
		Version:     cmd.Root().Version,
		GOOS:        runtime.GOOS,
		GOARCH:      runtime.GOARCH,
		Cmd:         cmd.Name(),
		CmdErrClass: cmdErrClass,
		Test:        false,
	}

	reqBody, err := json.Marshal(telemetryRequest)
	if err != nil {
		return fmt.Errorf("marshalling input: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, telemetryURL().String(), bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("doing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http error %d", resp.StatusCode)
	}

	return nil
}

func telemetryURL() *url.URL {
	return &url.URL{
		Scheme: "https",
		Host:   apiHost,
		Path:   telemetryPath,
	}
}
