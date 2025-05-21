// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package tdx

import (
	"crypto/x509/pkix"

	"github.com/edgelesssys/contrast/internal/attestation/extension"
	"github.com/edgelesssys/contrast/internal/oid"
	"github.com/google/go-tdx-guest/proto/tdx"
)

var (
	// We use the raw TDX OID as root range for our parsed TDX report extensions.
	// This OID NOT be used for any parsed extension directly.
	rootOID = oid.RawTDXReport

	// Header.

	versionOID           = append(rootOID, 1)
	attestationKeyTypeID = append(rootOID, 2)
	teeTypeOID           = append(rootOID, 3)
	qeSvnOid             = append(rootOID, 4)
	pceSvnOID            = append(rootOID, 5)
	qeVendorIDOID        = append(rootOID, 6)
	userDataOID          = append(rootOID, 7)

	// TdQuoteBody.

	teeTcbSvnOID      = append(rootOID, 8)
	mrSeamOID         = append(rootOID, 9)
	mrSignerSeamOID   = append(rootOID, 10)
	seamAttributesOID = append(rootOID, 11)
	tdAttributesOID   = append(rootOID, 12)
	xfamOID           = append(rootOID, 13)
	mrTdOID           = append(rootOID, 14)
	mrConfigIDOID     = append(rootOID, 15)
	mrOwnerOID        = append(rootOID, 16)
	mrOwnerConfigOID  = append(rootOID, 17)
	rtmr0OID          = append(rootOID, 18)
	rtmr1OID          = append(rootOID, 19)
	rtmr2OID          = append(rootOID, 20)
	rtmr3OID          = append(rootOID, 21)
	tdReportDataOID   = append(rootOID, 22)

	// End of TdQuoteBody.

	signedDataSizeOID = append(rootOID, 23)

	// SignedData.

	signatureOID           = append(rootOID, 24)
	ecdsaAttestationKeyOID = append(rootOID, 25)

	// CertificationData.

	certificateDataTypeOID = append(rootOID, 26)
	sizeOID                = append(rootOID, 27)

	// QeReportCertificationData.
	// QeReport.

	cpuSvnOID     = append(rootOID, 29)
	miscSelectOID = append(rootOID, 30)
	reserved1OID  = append(rootOID, 31)
	attributesOID = append(rootOID, 32)
	mrEnclaveOID  = append(rootOID, 33)
	reserved2OID  = append(rootOID, 34)
	mrSignerOID   = append(rootOID, 35)
	reserved3OID  = append(rootOID, 36)
	isvProdIDOID  = append(rootOID, 37)
	isvSvnOID     = append(rootOID, 38)
	reserved4OID  = append(rootOID, 39)
	reportDataOID = append(rootOID, 40)

	// End of EnclaveReport.

	qeReportSignatureOID = append(rootOID, 41)

	// QeAuthData.

	parsedDataSizeOID = append(rootOID, 42)
	dataOID           = append(rootOID, 43)

	// End of QeAuthData.
	// PCKCertificateChainData.

	pckCertificateChainDataTypeOID = append(rootOID, 44)
	pckCertificateChainDataSizeOID = append(rootOID, 45)
	pckCertificateChainDataOID     = append(rootOID, 46)

	// End of PCKCertificateChainData.
	// End of QEReportCertificationData.
	// End of CertificationData.
	// End of Ecdsa256BitQuoteV4AuthData.

	extraBytesOID = append(rootOID, 47)
)

// claimsToCertExtension constructs certificate extensions from a SNP report.
func claimsToCertExtension(quote *tdx.QuoteV4) ([]pkix.Extension, error) {
	var extensions []extension.Extension

	extensions = append(extensions, extension.NewBigIntExtension(versionOID, quote.Header.Version))
	extensions = append(extensions, extension.NewBigIntExtension(attestationKeyTypeID, quote.Header.AttestationKeyType))
	extensions = append(extensions, extension.NewBigIntExtension(teeTypeOID, quote.Header.TeeType))
	extensions = append(extensions, extension.NewBytesExtension(qeSvnOid, quote.Header.QeSvn))
	extensions = append(extensions, extension.NewBytesExtension(pceSvnOID, quote.Header.PceSvn))
	extensions = append(extensions, extension.NewBytesExtension(qeVendorIDOID, quote.Header.QeVendorId))
	extensions = append(extensions, extension.NewBytesExtension(userDataOID, quote.Header.UserData))

	extensions = append(extensions, extension.NewBytesExtension(teeTcbSvnOID, quote.TdQuoteBody.TeeTcbSvn))
	extensions = append(extensions, extension.NewBytesExtension(mrSeamOID, quote.TdQuoteBody.MrSeam))
	extensions = append(extensions, extension.NewBytesExtension(mrSignerSeamOID, quote.TdQuoteBody.MrSignerSeam))
	extensions = append(extensions, extension.NewBytesExtension(seamAttributesOID, quote.TdQuoteBody.SeamAttributes))
	extensions = append(extensions, extension.NewBytesExtension(tdAttributesOID, quote.TdQuoteBody.TdAttributes))
	extensions = append(extensions, extension.NewBytesExtension(xfamOID, quote.TdQuoteBody.Xfam))
	extensions = append(extensions, extension.NewBytesExtension(mrTdOID, quote.TdQuoteBody.MrTd))
	extensions = append(extensions, extension.NewBytesExtension(mrConfigIDOID, quote.TdQuoteBody.MrConfigId))
	extensions = append(extensions, extension.NewBytesExtension(mrOwnerOID, quote.TdQuoteBody.MrOwner))
	extensions = append(extensions, extension.NewBytesExtension(mrOwnerConfigOID, quote.TdQuoteBody.MrOwnerConfig))
	extensions = append(extensions, extension.NewBytesExtension(rtmr0OID, quote.TdQuoteBody.Rtmrs[0]))
	extensions = append(extensions, extension.NewBytesExtension(rtmr1OID, quote.TdQuoteBody.Rtmrs[1]))
	extensions = append(extensions, extension.NewBytesExtension(rtmr2OID, quote.TdQuoteBody.Rtmrs[2]))
	extensions = append(extensions, extension.NewBytesExtension(rtmr3OID, quote.TdQuoteBody.Rtmrs[3]))
	extensions = append(extensions, extension.NewBytesExtension(tdReportDataOID, quote.TdQuoteBody.ReportData))

	extensions = append(extensions, extension.NewBigIntExtension(signedDataSizeOID, quote.SignedDataSize))

	extensions = append(extensions, extension.NewBytesExtension(signatureOID, quote.SignedData.Signature))
	extensions = append(extensions, extension.NewBytesExtension(ecdsaAttestationKeyOID, quote.SignedData.EcdsaAttestationKey))

	extensions = append(extensions, extension.NewBigIntExtension(certificateDataTypeOID, quote.SignedData.CertificationData.CertificateDataType))
	extensions = append(extensions, extension.NewBigIntExtension(sizeOID, quote.SignedData.CertificationData.Size))

	extensions = append(extensions, extension.NewBytesExtension(cpuSvnOID, quote.SignedData.CertificationData.QeReportCertificationData.QeReport.CpuSvn))
	extensions = append(extensions, extension.NewBigIntExtension(miscSelectOID, quote.SignedData.CertificationData.QeReportCertificationData.QeReport.MiscSelect))
	extensions = append(extensions, extension.NewBytesExtension(reserved1OID, quote.SignedData.CertificationData.QeReportCertificationData.QeReport.Reserved1))
	extensions = append(extensions, extension.NewBytesExtension(attributesOID, quote.SignedData.CertificationData.QeReportCertificationData.QeReport.Attributes))
	extensions = append(extensions, extension.NewBytesExtension(mrEnclaveOID, quote.SignedData.CertificationData.QeReportCertificationData.QeReport.MrEnclave))
	extensions = append(extensions, extension.NewBytesExtension(reserved2OID, quote.SignedData.CertificationData.QeReportCertificationData.QeReport.Reserved2))
	extensions = append(extensions, extension.NewBytesExtension(mrSignerOID, quote.SignedData.CertificationData.QeReportCertificationData.QeReport.MrSigner))
	extensions = append(extensions, extension.NewBytesExtension(reserved3OID, quote.SignedData.CertificationData.QeReportCertificationData.QeReport.Reserved3))
	extensions = append(extensions, extension.NewBigIntExtension(isvProdIDOID, quote.SignedData.CertificationData.QeReportCertificationData.QeReport.IsvProdId))
	extensions = append(extensions, extension.NewBigIntExtension(isvSvnOID, quote.SignedData.CertificationData.QeReportCertificationData.QeReport.IsvSvn))
	extensions = append(extensions, extension.NewBytesExtension(reserved4OID, quote.SignedData.CertificationData.QeReportCertificationData.QeReport.Reserved4))
	extensions = append(extensions, extension.NewBytesExtension(reportDataOID, quote.SignedData.CertificationData.QeReportCertificationData.QeReport.ReportData))

	extensions = append(extensions, extension.NewBytesExtension(qeReportSignatureOID, quote.SignedData.CertificationData.QeReportCertificationData.QeReportSignature))

	extensions = append(extensions, extension.NewBigIntExtension(parsedDataSizeOID, quote.SignedData.CertificationData.QeReportCertificationData.QeAuthData.ParsedDataSize))
	extensions = append(extensions, extension.NewBytesExtension(dataOID, quote.SignedData.CertificationData.QeReportCertificationData.QeAuthData.Data))

	extensions = append(extensions, extension.NewBigIntExtension(pckCertificateChainDataTypeOID, quote.SignedData.CertificationData.QeReportCertificationData.PckCertificateChainData.CertificateDataType))
	extensions = append(extensions, extension.NewBigIntExtension(pckCertificateChainDataSizeOID, quote.SignedData.CertificationData.QeReportCertificationData.PckCertificateChainData.Size))
	extensions = append(extensions, extension.NewBytesExtension(pckCertificateChainDataOID, quote.SignedData.CertificationData.QeReportCertificationData.PckCertificateChainData.PckCertChain))

	extensions = append(extensions, extension.NewBytesExtension(extraBytesOID, quote.ExtraBytes))

	return extension.ConvertExtensions(extensions)
}
