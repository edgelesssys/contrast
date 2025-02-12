# AMD KDS

For successful verification, the client needs to receive both VCEK and CRL.

## Security considerations

VCEK and CRL are both signed by ASK/ARK.
The client is in possession of ASK/ARK and can use it to verify the authenticity and integrity of both VCEK and CRL after receiving it from a potentially untrusted source.
Following, any service, trusted or not, can be used by the client to obtain these values, including the Coordinator the client wants to verify.
The go-sev-guest library checks the signature of both (https://github.com/google/go-sev-guest/blob/65042ded71f9f2cf85a68334288d575977680ba1/verify/verify.go#L526) and CRL (https://github.com/google/go-sev-guest/blob/65042ded71f9f2cf85a68334288d575977680ba1/verify/verify.go#L342).

The CRL has an expiration date included. The go-sev-guest library will check the CRL isn't expired during the verification on client side (https://github.com/google/go-sev-guest/blob/65042ded71f9f2cf85a68334288d575977680ba1/verify/verify.go#L303).

## Request and caching flow

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
