# Endorsement Caching

## AMD KDS

For successful verification, the client needs to receive both VCEK and CRL.

### Security considerations

VCEK and CRL are both signed by ASK/ARK.
The client is in possession of ASK/ARK and can use it to verify the authenticity and integrity of both VCEK and CRL after receiving it from a potentially untrusted source.
Following, any service, trusted or not, can be used by the client to obtain these values, including the Coordinator the client wants to verify.
The go-sev-guest library checks the signature of both (https://github.com/google/go-sev-guest/blob/65042ded71f9f2cf85a68334288d575977680ba1/verify/verify.go#L526) and CRL (https://github.com/google/go-sev-guest/blob/65042ded71f9f2cf85a68334288d575977680ba1/verify/verify.go#L342).

The CRL has an expiration date included. The go-sev-guest library will check the CRL isn't expired during the verification on client side (https://github.com/google/go-sev-guest/blob/65042ded71f9f2cf85a68334288d575977680ba1/verify/verify.go#L303).

### Request and caching flow

The following gives an overview over the request and caching structure we implement.

![](kds.drawio.svg)

1. The issuer requests the VCEK from THIM, which is a service provided by Azure via the IMDS API.
   It doesn't serve the CRL, which must still be requested from a different source.

2. On the issuer side, the go-sev-guest library will check if the report is an extended report.
   If yes, the VCEK included in the report is used.
   The CRL isn't included in an extended report, it must still be requested from a different source.

3. On the issuer side, go-sev-guest library will request CRL (and VCEK, if not obtained in steps 1-2) from the AMD KDS.
   It uses a cached HTTP getter.
   When the VCEK is stored in cache, it's used without requesting the KDS.
   When the KDS request for CRL fails, a cached, unexpired CRL is used if found.
   The issuer cache is in-memory.

4. On the issuer side, the go-sev-guest library requests CRL (and VCEK, if not obtained in steps 1-3) from the AMD KDS.
   If the VCEK is obtained from KDS, it will be stored in the issuers cache.
   If the CRL is obtained from KDS, it will be stored in the issuers cache.

5. If the issuer successfully obtained VCEK or CRL in 1-4, it will send it in the aTLS handshake.
   The issuer won't fail if it couldn't obtain VCEK or CRL.
   The validator uses VCEK and CRL from the handshake, if present.

6. On the validator side, go-sev-guest library will request the CRL from the AMD KDS, if it wasn't present in the handshake.
   Each request from the go-sev-guest library check the local on-disk cache.
   The validator will use VCEK from the validator cache, if present.
   The validator will use a non-expired CRL from cache, if the KDS request for a fresh CRL fails.

7. On the validator side, go-sev-guest library will request the CRL from the AMD KDS, if it wasn't present in the handshake.
   If the VCEK is obtained from KDS, it will be stored in the validator cache.
   If the CRL is obtained from KDS, it will be stored in the validator cache.
   If the CRL can't be obtained from KDS, the cache is checked for an unexpired CRL.
   The validator cache is on-disk.

### KDS Unavailability

If the KDS is unreachable, the following table shows the validation outcome depending on the cache state.
An empty cell indicates that the value isn't present.
An asterisk (*) indicates that any value is acceptable (present or not present).
CRL + VCEK indicates that *both* CRL and VCEK are present in the cache.

| Issuer can't reach KDS | Validator can't reach KDS | Issuer cache | Validator cache | Validation |
| :--------------------: | :-----------------------: | :----------: | :-------------: | :--------: |
|                        |                           | *            | *               | Success    |
| X                      |                           |              | *               | Success    |
|                        | X                         | *            |                 | Success    |
| X                      | X                         |              |                 | Failure    |
| X                      | X                         | CRL + VCEK   | *               | Success    |
| X                      | X                         | *            | CRL + VCEK      | Success    |

## Intel PCS

For successful verification, the client needs to receive the TCBInfo, QeIdentity, and the Root CRL and PCK CRL.
The certificate chain of the quote can be verified directly using an embedded Intel Root Certificate.

### Security considerations

The quote is signed by the PCK, which is signed through an intermediate certificate by the Intel Root Certificate.
The same goes for the TCBInfo and QeIdentity.
The go-tdx-guest library will [verify the certificate chain included in the quote](https://github.com/google/go-tdx-guest/blob/9efd53b4a100e467dfd00c79fbb3de19f71b1ba4/verify/verify.go#L1328) and [check the signature of the quote](https://github.com/google/go-tdx-guest/blob/9efd53b4a100e467dfd00c79fbb3de19f71b1ba4/verify/verify.go#L1147).
The go-tdx-guest library will also [verify the TD Quote Body against the TCBInfo](https://github.com/google/go-tdx-guest/blob/9efd53b4a100e467dfd00c79fbb3de19f71b1ba4/verify/verify.go#L1160) and
[the QE Report against the QeIdentity](https://github.com/google/go-tdx-guest/blob/9efd53b4a100e467dfd00c79fbb3de19f71b1ba4/verify/verify.go#L1169).
Both the TCBInfo and QeIdentity are signed through an intermediate certificate by the Intel Root Certificate.
This is also [verified by the go-tdx-guest library](https://github.com/google/go-tdx-guest/blob/9efd53b4a100e467dfd00c79fbb3de19f71b1ba4/verify/verify.go#L1339-L1350).
The expiration date of the CRLS as well as the expiration date included in the TCBInfo and QeIdentity is also [checked by the go-tdx-guest library](https://github.com/google/go-tdx-guest/blob/9efd53b4a100e467dfd00c79fbb3de19f71b1ba4/verify/verify.go#L627).

### Request and caching flow

![](pcs.drawio.svg)

1. The issuer generates an attestation document using the Intel Quote Provider Library (QPL).
   The QPL will use the Intel Provisioning Certificate Caching Service (PCCS) to obtain the certificate chain which is included in the quote.
2. The PCCS is a service that runs locally on the host and automatically caches responses from the Intel Provisioning Certificate Service (PCS).
   If the certificate chain isn't present in the PCCS cache, it will be requested from the PCS.
3. The issuer sends the attestation document to the validator.
   The quote contains the Intel Root Certificate, the PCK Platform Certificate, and the PCK Certificate.
   The Intel PCK certificate can be used to verify the quote.
4. On the validator side, the go-tdx-guest library will retrieve the collateral from the PCS which is needed to verify the quote.
   This includes the TCBInfo, the QeIdentity, as well as the Root CRL and the PCK CRL.
5. On the validator side, if the collateral or CRLs can't be retrieved from the PCS, the go-tdx-guest library will use the collateral from the local cache if present.
   If the CRLs can't be retrieved from the PCS, the cache is checked for an unexpired CRL.
   The validator cache is on-disk.

### PCS Unavailability

If the PCS is unreachable, the following table shows the validation outcome depending on the cache state.
An empty cell indicates that the value isn't present.
An asterisk (*) indicates that any value is acceptable (present or not present).
The issuer must be able to obtain the PCK Certificate Chain from PCCS to generate a valid quote.
The validator must be able to obtain the collateral (from PCS or cache) to validate the quote.

| Issuer can't reach PCS | Validator can't reach PCS | Issuer cache           | Validator cache | Validation |
| :--------------------: | :-----------------------: | :--------------------: | :-------------: | :--------: |
|                        |                           | *                      | *               | Success    |
| X                      |                           |                        | *               | Failure    |
| X                      |                           | *PCCS: PCK Cert Chain* | *               | Success    |
|                        | X                         | *                      |                 | Failure    |
|                        | X                         | *                      | Collateral      | Success    |
| X                      | X                         |                        |                 | Failure    |
| X                      | X                         | *PCCS: PCK Cert Chain* |                 | Failure    |
| X                      | X                         |                        | Collateral      | Failure    |
| X                      | X                         | *PCCS: PCK Cert Chain* | Collateral      | Success    |
