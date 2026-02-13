// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package quote

import (
	"fmt"
	"testing"

	"github.com/google/go-tdx-guest/proto/tdx"
	"github.com/stretchr/testify/assert"
)

func TestGetExtensions(t *testing.T) {
	for name, tc := range map[string]struct {
		quote   *tdx.QuoteV4
		wantErr error
	}{
		"no extra bytes": {
			quote:   &tdx.QuoteV4{},
			wantErr: errNoExtensions,
		},
		"padding in extra bytes": {
			quote:   &tdx.QuoteV4{ExtraBytes: []byte{0, 0, 0, 0}},
			wantErr: errNoExtensions,
		},
		"JSON encoding": {
			// This tests backwards-compatibility of the extension serialization.
			// Don't change without a good reason!
			quote: &tdx.QuoteV4{ExtraBytes: []byte(`{"Collateral": {"TcbInfoBody": ""}}`)},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			ext, err := GetExtensions(tc.quote)

			if tc.wantErr != nil {
				assert.ErrorIs(err, tc.wantErr)
				assert.Nil(ext)
				return
			}
			assert.NoError(err)
			assert.NotNil(ext)
			assert.NotNil(ext.Collateral)
		})
	}
}

func TestGetPCKCertificate(t *testing.T) {
	for name, tc := range map[string]struct {
		quote   *tdx.QuoteV4
		wantErr error
	}{
		"empty quote": {
			quote:   &tdx.QuoteV4{},
			wantErr: errNoPCKCertificate,
		},
		"incomplete chain": {
			quote:   quoteWithCertChain(fmt.Appendf(nil, "%s\n%s", rootCertPEM, intermediateCertPEM)),
			wantErr: errNoPCKCertificate,
		},
		"root to leaf": {
			quote: quoteWithCertChain(fmt.Appendf(nil, "%s\n%s\n%s", rootCertPEM, intermediateCertPEM, pckCertPEM)),
		},
		"leaf to root": {
			quote: quoteWithCertChain(fmt.Appendf(nil, "%s\n%s\n%s", pckCertPEM, intermediateCertPEM, rootCertPEM)),
		},
		"leaf in the middle": {
			quote: quoteWithCertChain(fmt.Appendf(nil, "%s\n%s\n%s", intermediateCertPEM, pckCertPEM, rootCertPEM)),
		},
		"garbage in between PEM": {
			quote: quoteWithCertChain(fmt.Appendf(nil, "foo\n%s\nbar\nbar\n%s\n%s\nbaz\n", rootCertPEM, intermediateCertPEM, pckCertPEM)),
		},
		"not a certificate": {
			quote:   quoteWithCertChain([]byte("-----BEGIN VOID-----\n\n-----END VOID-----")),
			wantErr: errParseCertificate,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			pck, err := GetPCKCertificate(tc.quote)
			if tc.wantErr != nil {
				assert.ErrorIs(err, tc.wantErr)
				return
			}
			assert.NoError(err)
			assert.NotNil(pck)
		})
	}
}

func quoteWithCertChain(certChain []byte) *tdx.QuoteV4 {
	return &tdx.QuoteV4{
		SignedData: &tdx.Ecdsa256BitQuoteV4AuthData{
			CertificationData: &tdx.CertificationData{
				QeReportCertificationData: &tdx.QEReportCertificationData{
					PckCertificateChainData: &tdx.PCKCertificateChainData{
						PckCertChain: certChain,
					},
				},
			},
		},
	}
}

const (
	pckCertPEM = `-----BEGIN CERTIFICATE-----
MIIE8DCCBJagAwIBAgIUG0LpXJ4aM7BzY6g/fuuK2vE+ncwwCgYIKoZIzj0EAwIw
cDEiMCAGA1UEAwwZSW50ZWwgU0dYIFBDSyBQbGF0Zm9ybSBDQTEaMBgGA1UECgwR
SW50ZWwgQ29ycG9yYXRpb24xFDASBgNVBAcMC1NhbnRhIENsYXJhMQswCQYDVQQI
DAJDQTELMAkGA1UEBhMCVVMwHhcNMjUxMjA5MTQ0MDM4WhcNMzIxMjA5MTQ0MDM4
WjBwMSIwIAYDVQQDDBlJbnRlbCBTR1ggUENLIENlcnRpZmljYXRlMRowGAYDVQQK
DBFJbnRlbCBDb3Jwb3JhdGlvbjEUMBIGA1UEBwwLU2FudGEgQ2xhcmExCzAJBgNV
BAgMAkNBMQswCQYDVQQGEwJVUzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABHDw
/yDkpgiNQNn9GL7x78W67TnK5VIwNXrBU6eTV5D+r92i6IVfeYNH3NxtDBcEBHdR
vYQoPdRLdq636E4G6T2jggMMMIIDCDAfBgNVHSMEGDAWgBSVb13NvRvh6UBJydT0
M84BVwveVDBrBgNVHR8EZDBiMGCgXqBchlpodHRwczovL2FwaS50cnVzdGVkc2Vy
dmljZXMuaW50ZWwuY29tL3NneC9jZXJ0aWZpY2F0aW9uL3Y0L3Bja2NybD9jYT1w
bGF0Zm9ybSZlbmNvZGluZz1kZXIwHQYDVR0OBBYEFM3/SqfuQu2NXbVabyk+TKYK
r6D6MA4GA1UdDwEB/wQEAwIGwDAMBgNVHRMBAf8EAjAAMIICOQYJKoZIhvhNAQ0B
BIICKjCCAiYwHgYKKoZIhvhNAQ0BAQQQmOivJhyBqPm/MIPK3n/r5DCCAWMGCiqG
SIb4TQENAQIwggFTMBAGCyqGSIb4TQENAQIBAgEDMBAGCyqGSIb4TQENAQICAgED
MBAGCyqGSIb4TQENAQIDAgECMBAGCyqGSIb4TQENAQIEAgECMBAGCyqGSIb4TQEN
AQIFAgEEMBAGCyqGSIb4TQENAQIGAgEBMBAGCyqGSIb4TQENAQIHAgEAMBAGCyqG
SIb4TQENAQIIAgEFMBAGCyqGSIb4TQENAQIJAgEAMBAGCyqGSIb4TQENAQIKAgEA
MBAGCyqGSIb4TQENAQILAgEAMBAGCyqGSIb4TQENAQIMAgEAMBAGCyqGSIb4TQEN
AQINAgEAMBAGCyqGSIb4TQENAQIOAgEAMBAGCyqGSIb4TQENAQIPAgEAMBAGCyqG
SIb4TQENAQIQAgEAMBAGCyqGSIb4TQENAQIRAgENMB8GCyqGSIb4TQENAQISBBAD
AwICBAEABQAAAAAAAAAAMBAGCiqGSIb4TQENAQMEAgAAMBQGCiqGSIb4TQENAQQE
BpDAbwAAADAPBgoqhkiG+E0BDQEFCgEBMB4GCiqGSIb4TQENAQYEEOkCEHAqLMWt
l2TyndyP3owwRAYKKoZIhvhNAQ0BBzA2MBAGCyqGSIb4TQENAQcBAQH/MBAGCyqG
SIb4TQENAQcCAQH/MBAGCyqGSIb4TQENAQcDAQH/MAoGCCqGSM49BAMCA0gAMEUC
IQDzyYPVyE/QaMPJNUj4uJMmVY6Hfd6phc7PeFN/p400+QIgUSa1nApG1uAleSYW
uCTrkjLzdIy5EqAE/+WFeMwNDxw=
-----END CERTIFICATE-----`
	intermediateCertPEM = `-----BEGIN CERTIFICATE-----
MIICljCCAj2gAwIBAgIVAJVvXc29G+HpQEnJ1PQzzgFXC95UMAoGCCqGSM49BAMC
MGgxGjAYBgNVBAMMEUludGVsIFNHWCBSb290IENBMRowGAYDVQQKDBFJbnRlbCBD
b3Jwb3JhdGlvbjEUMBIGA1UEBwwLU2FudGEgQ2xhcmExCzAJBgNVBAgMAkNBMQsw
CQYDVQQGEwJVUzAeFw0xODA1MjExMDUwMTBaFw0zMzA1MjExMDUwMTBaMHAxIjAg
BgNVBAMMGUludGVsIFNHWCBQQ0sgUGxhdGZvcm0gQ0ExGjAYBgNVBAoMEUludGVs
IENvcnBvcmF0aW9uMRQwEgYDVQQHDAtTYW50YSBDbGFyYTELMAkGA1UECAwCQ0Ex
CzAJBgNVBAYTAlVTMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAENSB/7t21lXSO
2Cuzpxw74eJB72EyDGgW5rXCtx2tVTLq6hKk6z+UiRZCnqR7psOvgqFeSxlmTlJl
eTmi2WYz3qOBuzCBuDAfBgNVHSMEGDAWgBQiZQzWWp00ifODtJVSv1AbOScGrDBS
BgNVHR8ESzBJMEegRaBDhkFodHRwczovL2NlcnRpZmljYXRlcy50cnVzdGVkc2Vy
dmljZXMuaW50ZWwuY29tL0ludGVsU0dYUm9vdENBLmRlcjAdBgNVHQ4EFgQUlW9d
zb0b4elAScnU9DPOAVcL3lQwDgYDVR0PAQH/BAQDAgEGMBIGA1UdEwEB/wQIMAYB
Af8CAQAwCgYIKoZIzj0EAwIDRwAwRAIgXsVki0w+i6VYGW3UF/22uaXe0YJDj1Ue
nA+TjD1ai5cCICYb1SAmD5xkfTVpvo4UoyiSYxrDWLmUR4CI9NKyfPN+
-----END CERTIFICATE-----`
	rootCertPEM = `
-----BEGIN CERTIFICATE-----
MIICjzCCAjSgAwIBAgIUImUM1lqdNInzg7SVUr9QGzknBqwwCgYIKoZIzj0EAwIw
aDEaMBgGA1UEAwwRSW50ZWwgU0dYIFJvb3QgQ0ExGjAYBgNVBAoMEUludGVsIENv
cnBvcmF0aW9uMRQwEgYDVQQHDAtTYW50YSBDbGFyYTELMAkGA1UECAwCQ0ExCzAJ
BgNVBAYTAlVTMB4XDTE4MDUyMTEwNDUxMFoXDTQ5MTIzMTIzNTk1OVowaDEaMBgG
A1UEAwwRSW50ZWwgU0dYIFJvb3QgQ0ExGjAYBgNVBAoMEUludGVsIENvcnBvcmF0
aW9uMRQwEgYDVQQHDAtTYW50YSBDbGFyYTELMAkGA1UECAwCQ0ExCzAJBgNVBAYT
AlVTMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEC6nEwMDIYZOj/iPWsCzaEKi7
1OiOSLRFhWGjbnBVJfVnkY4u3IjkDYYL0MxO4mqsyYjlBalTVYxFP2sJBK5zlKOB
uzCBuDAfBgNVHSMEGDAWgBQiZQzWWp00ifODtJVSv1AbOScGrDBSBgNVHR8ESzBJ
MEegRaBDhkFodHRwczovL2NlcnRpZmljYXRlcy50cnVzdGVkc2VydmljZXMuaW50
ZWwuY29tL0ludGVsU0dYUm9vdENBLmRlcjAdBgNVHQ4EFgQUImUM1lqdNInzg7SV
Ur9QGzknBqwwDgYDVR0PAQH/BAQDAgEGMBIGA1UdEwEB/wQIMAYBAf8CAQEwCgYI
KoZIzj0EAwIDSQAwRgIhAOW/5QkR+S9CiSDcNoowLuPRLsWGf/Yi7GSX94BgwTwg
AiEA4J0lrHoMs+Xo5o/sX6O9QWxHRAvZUGOdRQ7cvqRXaqI=
-----END CERTIFICATE-----`
)
