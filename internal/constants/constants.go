// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package constants

import (
	"time"
)

var (
	// Version is the version of Contrast.
	Version = "0.0.0-dev"
	// KataGenpolicyVersion is the version of Kata's genpolicy tool.
	KataGenpolicyVersion = "0.0.0-dev"
)

const (
	// SecretSeedSize is the size of the secret seed generated in the coordinator.
	SecretSeedSize = 64
	// SecretSeedSaltSize is the size of the secret seed salt generated in the coordinator.
	SecretSeedSaltSize = 32
	// SNPCertChainExtrasCRLKey is the UUID of the cert chain extra that contains the CRL.
	SNPCertChainExtrasCRLKey = "00569ee4-e480-4fba-bbf4-45b629901180"
)

const (
	// ATLSClientConnectTimeout is the maximal amount of time spent by the client to
	// establish a connection and complete the aTLS handshake.
	// In the case of CLI (validator) and Coordinator (issuer), this includes the time spent
	// issuing the attestation document and trying to fetch additional data from endorsement
	// service (like AMD KDS), as well as time needed to validate that attestation document
	// (and fetching endorsements on the validator side, in case they weren't sent by the issuer).
	ATLSClientConnectTimeout = 30 * time.Second

	// ATLSServerHandshakeTimeout is the maximal amount of time a server can spend to finish
	// the aTLS handshake on its side. This can include issuing/validating attestation documents.
	// In the case of CLI (validator) and Coordinator (issuer), it includes the time spent
	// issuing the attestation document and trying to fetch additional data from endorsement
	// service (like AMD KDS), BUT NOT the time needed by the client to validate.
	ATLSServerHandshakeTimeout = ATLSClientConnectTimeout - 5*time.Second

	// ATLSIssuerOptionalEndorsementFetchTimeout is the timeout for fetching optional endorsements
	// (like AMD VCEK, ASK, ARK certificates and CRL) during Issue. This doesn't necessarily need
	// to succeed, as the client can also try to fetch or use cache on their side. Thus, the timeout
	// needs to be lower than the ServerHandshake timeout, as otherwise the ServerHandshake will fail
	// when blocking on the optional fetches.
	//
	// TODO(katexochen): This assumes that the issue is done on the server side, like it is the case for
	// CLI <-> Coordinator communication. Issue can, however, also be used on the client side,
	// for example in the Initializer <-> Coordinator communication. Thus deriving the timeout
	// from the ServerHandshake timeout isn't always correct.
	//
	// TODO(katexochen): Further, we need to also ensure that there is enough time on the validator side to
	// request the endorsement again on that end without hitting the overall ClientConnect timeout.
	// The current difference of 5s might not be sufficient.
	ATLSIssuerOptionalEndorsementFetchTimeout = ATLSServerHandshakeTimeout - 5*time.Second
)

// DisableServiceMeshEnvVar is the environment variable that signals to the initializer
// that the service mesh is disabled and that the initializer should remove
// the default deny iptables rule.
const DisableServiceMeshEnvVar = "CONTRAST_SERVICE_MESH_DISABLED"
