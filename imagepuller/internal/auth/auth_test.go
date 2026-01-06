// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package auth

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/stretchr/testify/assert"
)

var defaultTansport = &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
	Proxy:           http.ProxyFromEnvironment,
}

var exampleAuthConfig = authn.AuthConfig{
	Username:      "user",
	Password:      "pass",
	Auth:          "auth",
	IdentityToken: "id",
	RegistryToken: "reg",
}

func TestAuthTransportFor(t *testing.T) {
	tests := map[string]struct {
		imageRef          string
		config            Config
		wantAuthenticator authn.Authenticator
		wantTransport     *http.Transport
		wantErr           error
	}{
		"missing ref caught": {
			imageRef: "",
			wantErr:  errUnparseableRef,
		},
		"ip/port does not throw error in ref": {
			imageRef: "127.0.0.1:8000/edgelesssys/contrast/coordinator",
		},
		"ip/port does not throw error in config": {
			imageRef: "ghcr.io/edgelesssys/contrast/coordinator",
			config: Config{
				Registries: map[string]Registry{
					"127.0.0.1:8000": {CACerts: dummyCert1},
				},
			},
		},
		"unparseable / incomplete ref caught": {
			imageRef: "",
			wantErr:  errUnparseableRef,
		},
		"anonymous": {
			imageRef: "ghcr.io",
			config:   Config{},
		},
		"full fqdn matches": {
			imageRef: "ghcr.io/edgelesssys/contrast/coordinator",
			config: Config{
				Registries: map[string]Registry{
					"ghcr.io.": {CACerts: dummyCert1},
				},
			},
			wantTransport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: getCerts(t, dummyCert1),
				},
			},
		},
		"partial fqdn matches": {
			imageRef: "ghcr.io/edgelesssys/contrast/coordinator",
			config: Config{
				Registries: map[string]Registry{
					"ghcr.io": {CACerts: dummyCert1},
				},
			},
			wantTransport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: getCerts(t, dummyCert1),
				},
			},
		},
		"subdomain does not match parent zone": {
			imageRef: "ghcr.io/edgelesssys/contrast/coordinator",
			config: Config{
				Registries: map[string]Registry{
					".ghcr.io": {CACerts: dummyCert1},
				},
			},
		},
		"global cert": {
			imageRef: "ghcr.io/edgelesssys/contrast/coordinator",
			config: Config{
				Registries: map[string]Registry{
					".": {CACerts: dummyCert1},
				},
			},
			wantTransport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: getCerts(t, dummyCert1),
				},
			},
		},
		"system certs disabled": {
			imageRef: "ghcr.io/edgelesssys/contrast/coordinator",
			config: Config{
				Registries: map[string]Registry{
					".": {CACerts: "none"},
				},
			},
			wantTransport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: &x509.CertPool{},
				},
			},
		},
		"specific and global cert chooses specific": {
			imageRef: "ghcr.io/edgelesssys/contrast/coordinator",
			config: Config{
				Registries: map[string]Registry{
					"ghcr.io":  {CACerts: dummyCert1},
					".ghcr.io": {CACerts: dummyCert2},
					".":        {CACerts: dummyCert3},
				},
			},
			wantTransport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: getCerts(t, dummyCert1),
				},
			},
		},
		"multiple certs": {
			imageRef: "ghcr.io/edgelesssys/contrast/coordinator",
			config: Config{
				Registries: map[string]Registry{
					"ghcr.io": {CACerts: strings.Join([]string{dummyCert1, dummyCert2}, "\n")},
					".":       {CACerts: dummyCert3},
				},
			},
			wantTransport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: getCerts(t, dummyCert1, dummyCert2),
				},
			},
		},
		"non-matching certificates not applied": {
			imageRef: "ghcr.io/edgelesssys/contrast/coordinator",
			config: Config{
				Registries: map[string]Registry{
					"docker.com": {CACerts: dummyCert1},
				},
			},
		},
		"insecure skip verify": {
			imageRef: "ghcr.io/edgelesssys/contrast/coordinator",
			config: Config{
				Registries: map[string]Registry{
					"ghcr.io": {
						InsecureSkipVerify: true,
					},
				},
			},
			wantTransport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
		"auth config applied": {
			imageRef: "ghcr.io/edgelesssys/contrast/coordinator",
			config: Config{
				Registries: map[string]Registry{
					"ghcr.io": {AuthConfig: exampleAuthConfig},
				},
			},
			wantAuthenticator: authn.FromConfig(exampleAuthConfig),
		},
		"empty auth config ignored": {
			imageRef: "ghcr.io/edgelesssys/contrast/coordinator",
			config: Config{
				Registries: map[string]Registry{
					"ghcr.io": {AuthConfig: authn.AuthConfig{Auth: ""}},
				},
			},
		},
		"non-matching auth config not applied": {
			imageRef: "ghcr.io/edgelesssys/contrast/coordinator",
			config: Config{
				Registries: map[string]Registry{
					"docker.com": {AuthConfig: exampleAuthConfig},
				},
			},
		},
		"combined": {
			imageRef: "ghcr.io/edgelesssys/contrast/coordinator",
			config: Config{
				Registries: map[string]Registry{
					"ghcr.io.":   {CACerts: dummyCert1, AuthConfig: exampleAuthConfig, InsecureSkipVerify: true},
					"docker.com": {CACerts: dummyCert2, AuthConfig: authn.AuthConfig{}},
					".":          {CACerts: dummyCert3},
				},
			},
			wantAuthenticator: authn.FromConfig(exampleAuthConfig),
			wantTransport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:            getCerts(t, dummyCert1),
					InsecureSkipVerify: true,
				},
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			authenticator, transport, err := tc.config.AuthTransportFor(tc.imageRef)
			assert.ErrorIs(err, tc.wantErr)
			if err != nil {
				return
			}

			// We never return an empty authenticator from AuthTransportFor.
			// This assignment saves us from having to specify an authenticator for most test cases.
			if tc.wantAuthenticator == nil {
				tc.wantAuthenticator = authn.Anonymous
			}
			assert.Equal(tc.wantAuthenticator, *authenticator)

			// We never return an empty transport from AuthTransportFor.
			// This assignment saves us from having to specify a transport for most test cases.
			if tc.wantTransport == nil {
				tc.wantTransport = defaultTansport
			}
			assert.True(tc.wantTransport.TLSClientConfig.RootCAs.Equal(transport.TLSClientConfig.RootCAs))
			assert.Equal(tc.wantTransport.TLSClientConfig.InsecureSkipVerify, transport.TLSClientConfig.InsecureSkipVerify)
		})
	}
}

const dummyCert1 = `
Root CA 1
-----BEGIN CERTIFICATE-----
MIIBfDCCASGgAwIBAgIUU5G42y9bIh8+AU38qVOmKocc0CwwCgYIKoZIzj0EAwIw
EzERMA8GA1UEAwwIWW91ck5hbWUwHhcNMjUxMDIxMTUyNzEwWhcNMjYxMDIxMTUy
NzEwWjATMREwDwYDVQQDDAhZb3VyTmFtZTBZMBMGByqGSM49AgEGCCqGSM49AwEH
A0IABOJlyBb/sHBmHRncTqk4lm6hBkBYlZGcScXfl/IuAVVIo4zCGBzCmvc7jYc2
+gyVp+wxuvm7NRza4e1QOfJfrxOjUzBRMB0GA1UdDgQWBBTRE8qju+GIWzr5xCik
MdBJFOd1lzAfBgNVHSMEGDAWgBTRE8qju+GIWzr5xCikMdBJFOd1lzAPBgNVHRMB
Af8EBTADAQH/MAoGCCqGSM49BAMCA0kAMEYCIQCn+fVmAzB8HOakKGLx6oXF0WP0
GJibphhjfHPdNWEDdQIhAN3KFNWIYtE35+/rZb5I+oVKnqKS8igdIU9lXmpOps1j
-----END CERTIFICATE-----
`

const dummyCert2 = `
Root CA 2
-----BEGIN CERTIFICATE-----
MIIBezCCASGgAwIBAgIUUugBbePTzyVApU4DLSMmHnXXjcwwCgYIKoZIzj0EAwIw
EzERMA8GA1UEAwwIWW91ck5hbWUwHhcNMjUxMDIxMTUwNDI0WhcNMjYxMDIxMTUw
NDI0WjATMREwDwYDVQQDDAhZb3VyTmFtZTBZMBMGByqGSM49AgEGCCqGSM49AwEH
A0IABIgsA5IEeiBq6jDpH2ttxrI96beeOqa+EpGqmznQmzpFkPEpLWMUt21Ien71
rxdeFC7ySuuu95VPjSvO7EUM9qyjUzBRMB0GA1UdDgQWBBTVnuI2o36Mrja3RvwE
82lWg2m19zAfBgNVHSMEGDAWgBTVnuI2o36Mrja3RvwE82lWg2m19zAPBgNVHRMB
Af8EBTADAQH/MAoGCCqGSM49BAMCA0gAMEUCIGmEkl8jxjxqyAxs3QoAXeIx++Bz
Zm9dwbeTbrKysrGXAiEA8ce6iyJUCZCZVVJs/HDLcPbOKc2EPZvdcGGjIlGXulo=
-----END CERTIFICATE-----
`

const dummyCert3 = `
Root CA 3
-----BEGIN CERTIFICATE-----
MIIFlTCCA32gAwIBAgIUCijNUEh9MPW2CxDuiP8Gt+RpvP8wDQYJKoZIhvcNAQEL
BQAwczELMAkGA1UEBhMCZGUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDENMAsGA1UEAwwEdGVzdDEdMBsGCSqG
SIb3DQEJARYOdGVzdEB0ZXN0LnRlc3QwHhcNMjUxMDIzMDYzODE1WhcNMjYxMDIz
MDYzODE1WjBzMQswCQYDVQQGEwJkZTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8G
A1UECgwYSW50ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMQ0wCwYDVQQDDAR0ZXN0MR0w
GwYJKoZIhvcNAQkBFg50ZXN0QHRlc3QudGVzdDCCAiIwDQYJKoZIhvcNAQEBBQAD
ggIPADCCAgoCggIBAMWLb9PogQxJuyI8PNu9LnemWX9wpwExvuqhz4oTNDY4KWaV
Nd0JZJxXkNaylJyGcxke6lhkQ8jiDFtudy4K17eRxO8qVsbjwPfK6Th+PJLAqklT
x6u5KGZqk1LTwJN4Hdr959LIICJleSeDuvroG2X1Fr2ElEn3upriR5mFxBAL6LAc
GIoBImwid66wNynHziVMGeQELrX/27shaBgAB2qyDNCtdClohT8BEk7bgRp3EP48
eMxu2g9sEqyYHpE1mSTJtpxZbtLLoh1KBioxQgv+uJIQDDsnht3DOw0RCztJ4Em+
nJhCu7PTORUkb3m5BwODDyyWnNKl/ArJxIWikuxbv9aJ+DxKEG1Ns43HapAAwnVT
8BA+CHej8/SbTlqFXJkk+iqLaRZh/wCRBXxiciSFrOERsn/gyWVC7/HvXZebOVsN
K96WOfqv0LZAyiZEguHhfUMXKXnJadDmx8QU0uV6Vs0Eh68mnQHfyZCl2ijxaXgN
MrYbwIlXthNpT86W2xokNm0+CllPKBqFtZJApDRCztS3IjIaF9VvY39Smn9adrzU
x2LsD6FiRST3W/0y/HVL/cnUUvFJH4O9tcssiQ7w2QD7hReOMHxq4zHKeI71pmFA
NxACngtyIBJH3SjFAaRxZJd2+VKA2uzONOse+/qpo2zsdiqaWP93+S6g8ShrAgMB
AAGjITAfMB0GA1UdDgQWBBQ5qMg/VfIsdC0GU0dJSffQjTOXuDANBgkqhkiG9w0B
AQsFAAOCAgEAue0Zy5XrHWVcXkrLu6P6n0vgz2h8RmYe7v1Fu1r1+QC8/+VyV/ls
xD1LMvDjEogmEh1ckiUm2BpqQmW4HqMXkv3t26x/CzLw1d3DgSjL0LXJmhbHrFm7
ohrDoZjSHkuj9QVejwr3hQyvXbM+eEdIKa0JpaPW5gO4F0c6wfwleptDVpPebYLE
H/oMzTfNYVvUUsLPD29H4D62zHAPhv9vevuku77uTQwpVlkuKtsZofUqzRrZRo7A
eN4pWkKPUmZzpSBrLisoqRdxaFRy70s4sNVnq3Z/HNxp87GQeoJnkHXri5AgfGck
BDzMossinhLPp9W9yWG4Ccu6MP3vNYOGMgj9UMay+InXcUxtAzkVUn+rsPL5bmnN
VeFN/QXiH1+O2vHE7z3oa+3xD5pAMm1Zf0mNaKcev4PXGO36nrFUU8kGwlZuAELK
m+rGIoP9D3F2WaOR82dIt9CU6ZWFwxZ4pfxtVKytyJHAi28BJSml/nnfbDOAcNkK
TFxKd4UvLblFS196B3MCknBqGBiJAWhp57I/SXoCHyxTkFMup0zB9HBaVwOAev/p
5YQKRVpaQlt6tPO3tZCLlHV5HVq5SwHMHyB6hKRNbdRszasn1sqNUzEBjsuXwMQp
rg8wi5a3Un7POqWG1F+imgtRcNgxr9RzZ3SP0/gmZKGUYzsctaLBj2k=
-----END CERTIFICATE-----
`

func getCerts(t *testing.T, certs ...string) *x509.CertPool {
	t.Helper()
	certpool := x509.NewCertPool()
	certpool.AppendCertsFromPEM([]byte(strings.Join(certs, "\n")))
	return certpool
}

func TestRegistryFor(t *testing.T) {
	tests := map[string]string{
		"unknown":                ".",
		"example.org":            ".",
		"example.com":            ".com",
		"some.example.com":       ".example.com",
		"some.example.com.":      ".example.com",
		"other.example.com":      ".example.com",
		"some.other.example.com": ".example.com",
		"other.some.example.com": ".some.example.com",
	}
	for name, fqdn := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			c := Config{Registries: generateRegistries(t, fqdn)}
			registry := c.registryFor(name)

			// Using one of the string fields of authnAuthConfig to communicate the wanted registry name.
			assert.Equal(fqdn, registry.Auth)
		})
	}
}

var exampleFQDNs = []string{
	".",
	".com",
	".example.com",
	".some.example.com",
}

func generateRegistries(t *testing.T, fqdn string) map[string]Registry {
	t.Helper()
	registryMap := make(map[string]Registry)
	for _, registry := range exampleFQDNs {
		registryMap[registry] = Registry{AuthConfig: authn.AuthConfig{Auth: registry}}
	}
	registryMap[fqdn] = Registry{AuthConfig: authn.AuthConfig{Auth: fqdn}}
	return registryMap
}
