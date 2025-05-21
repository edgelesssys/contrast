// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package snp

import (
	"crypto/x509/pkix"
	"fmt"

	"github.com/edgelesssys/contrast/internal/attestation/extension"
	"github.com/edgelesssys/contrast/internal/oid"
	"github.com/google/go-sev-guest/abi"
	"github.com/google/go-sev-guest/kds"
	"github.com/google/go-sev-guest/proto/sevsnp"
)

var (
	// We use the raw SNP OID as root range for our parsed SNP report extensions.
	// This OID NOT be used for any parsed extension directly.
	rootOID = oid.RawSNPReport

	versionOID  = append(rootOID, 1)
	guestSVNOID = append(rootOID, 2)

	policyABIMajorOID     = append(rootOID, 3)
	policyABIMinorOID     = append(rootOID, 4)
	policyDebugOID        = append(rootOID, 5)
	policyMigrateMAOID    = append(rootOID, 6)
	policySMTOID          = append(rootOID, 7)
	policySingleSocketOID = append(rootOID, 8)

	familyIDOID = append(rootOID, 9)
	imageIDOID  = append(rootOID, 10)
	vmplOID     = append(rootOID, 11)

	currentTCBPartsBlSplOID    = append(rootOID, 12)
	currentTCBPartsSnpSplOID   = append(rootOID, 13)
	currentTCBPartsTeeSplOID   = append(rootOID, 14)
	currentTCBPartsUcodeSplOID = append(rootOID, 15)

	platformInfoSMTEnabledOID  = append(rootOID, 16)
	platformInfoTSMEEnabledOID = append(rootOID, 17)

	singerInfoAuthorKeyEnOID = append(rootOID, 18)
	singerInfoMaskChipKeyOID = append(rootOID, 19)
	singerInfoSigningKeyOID  = append(rootOID, 20)

	reportDataOID      = append(rootOID, 21)
	measurementOID     = append(rootOID, 22)
	hostDataOID        = append(rootOID, 23)
	idKeyDigestOID     = append(rootOID, 24)
	authorKeyDigestOID = append(rootOID, 25)
	reportIDOID        = append(rootOID, 26)
	reportIDMAOID      = append(rootOID, 27)

	reportedTCBPartsBlSplOID    = append(rootOID, 28)
	reportedTCBPartsSnpSplOID   = append(rootOID, 29)
	reportedTCBPartsTeeSplOID   = append(rootOID, 30)
	reportedTCBPartsUcodeSplOID = append(rootOID, 31)

	chipIDOID = append(rootOID, 32)

	committedTCBPartsBlSplOID    = append(rootOID, 33)
	committedTCBPartsSnpSplOID   = append(rootOID, 34)
	committedTCBPartsTeeSplOID   = append(rootOID, 35)
	committedTCBPartsUcodeSplOID = append(rootOID, 36)

	currentBuildOID   = append(rootOID, 37)
	currentMinorOID   = append(rootOID, 38)
	currentMajorOID   = append(rootOID, 39)
	committedBuildOID = append(rootOID, 40)
	committedMinorOID = append(rootOID, 41)
	committedMajorOID = append(rootOID, 42)

	launchTCBPartsBlSplOID    = append(rootOID, 43)
	launchTCBPartsSnpSplOID   = append(rootOID, 44)
	launchTCBPartsTeeSplOID   = append(rootOID, 45)
	launchTCBPartsUcodeSplOID = append(rootOID, 46)
)

// claimsToCertExtension constructs certificate extensions from a SNP report.
func claimsToCertExtension(report *sevsnp.Report) ([]pkix.Extension, error) {
	var extensions []extension.Extension

	extensions = append(extensions, extension.NewBigIntExtension(versionOID, report.Version))
	extensions = append(extensions, extension.NewBigIntExtension(guestSVNOID, report.GuestSvn))

	parsedPolicy, err := abi.ParseSnpPolicy(report.Policy)
	if err != nil {
		return nil, fmt.Errorf("parsing policy: %w", err)
	}

	extensions = append(extensions, extension.NewBigIntExtension(policyABIMajorOID, parsedPolicy.ABIMajor))
	extensions = append(extensions, extension.NewBigIntExtension(policyABIMinorOID, parsedPolicy.ABIMinor))
	extensions = append(extensions, extension.NewBoolExtension(policySMTOID, parsedPolicy.SMT))
	extensions = append(extensions, extension.NewBoolExtension(policyMigrateMAOID, parsedPolicy.MigrateMA))
	extensions = append(extensions, extension.NewBoolExtension(policyDebugOID, parsedPolicy.Debug))
	extensions = append(extensions, extension.NewBoolExtension(policySingleSocketOID, parsedPolicy.SingleSocket))

	extensions = append(extensions, extension.NewBytesExtension(familyIDOID, report.FamilyId))
	extensions = append(extensions, extension.NewBytesExtension(imageIDOID, report.ImageId))
	extensions = append(extensions, extension.NewBigIntExtension(vmplOID, report.Vmpl))

	tcbParts := kds.DecomposeTCBVersion(kds.TCBVersion(report.CurrentTcb))
	extensions = append(extensions, extension.NewBigIntExtension(currentTCBPartsBlSplOID, tcbParts.BlSpl))
	extensions = append(extensions, extension.NewBigIntExtension(currentTCBPartsTeeSplOID, tcbParts.TeeSpl))
	extensions = append(extensions, extension.NewBigIntExtension(currentTCBPartsSnpSplOID, tcbParts.SnpSpl))
	extensions = append(extensions, extension.NewBigIntExtension(currentTCBPartsUcodeSplOID, tcbParts.UcodeSpl))

	parsedPlatformInfo, err := abi.ParseSnpPlatformInfo(report.PlatformInfo)
	if err != nil {
		return nil, fmt.Errorf("parsing platform info: %w", err)
	}
	extensions = append(extensions, extension.NewBoolExtension(platformInfoSMTEnabledOID, parsedPlatformInfo.SMTEnabled))
	extensions = append(extensions, extension.NewBoolExtension(platformInfoTSMEEnabledOID, parsedPlatformInfo.TSMEEnabled))

	parsedSingerInfo, err := abi.ParseSignerInfo(report.SignerInfo)
	if err != nil {
		return nil, fmt.Errorf("parsing singer info: %w", err)
	}
	extensions = append(extensions, extension.NewBigIntExtension(singerInfoSigningKeyOID, parsedSingerInfo.SigningKey))
	extensions = append(extensions, extension.NewBoolExtension(singerInfoAuthorKeyEnOID, parsedSingerInfo.AuthorKeyEn))
	extensions = append(extensions, extension.NewBoolExtension(singerInfoMaskChipKeyOID, parsedSingerInfo.MaskChipKey))

	extensions = append(extensions, extension.NewBytesExtension(reportDataOID, report.ReportData))
	extensions = append(extensions, extension.NewBytesExtension(measurementOID, report.Measurement))
	extensions = append(extensions, extension.NewBytesExtension(hostDataOID, report.HostData))
	extensions = append(extensions, extension.NewBytesExtension(idKeyDigestOID, report.IdKeyDigest))
	extensions = append(extensions, extension.NewBytesExtension(authorKeyDigestOID, report.AuthorKeyDigest))
	extensions = append(extensions, extension.NewBytesExtension(reportIDOID, report.ReportId))
	extensions = append(extensions, extension.NewBytesExtension(reportIDMAOID, report.ReportIdMa))

	tcbParts = kds.DecomposeTCBVersion(kds.TCBVersion(report.ReportedTcb))
	extensions = append(extensions, extension.NewBigIntExtension(reportedTCBPartsBlSplOID, tcbParts.BlSpl))
	extensions = append(extensions, extension.NewBigIntExtension(reportedTCBPartsTeeSplOID, tcbParts.TeeSpl))
	extensions = append(extensions, extension.NewBigIntExtension(reportedTCBPartsSnpSplOID, tcbParts.SnpSpl))
	extensions = append(extensions, extension.NewBigIntExtension(reportedTCBPartsUcodeSplOID, tcbParts.UcodeSpl))

	extensions = append(extensions, extension.NewBytesExtension(chipIDOID, report.ChipId))

	tcbParts = kds.DecomposeTCBVersion(kds.TCBVersion(report.CommittedTcb))
	extensions = append(extensions, extension.NewBigIntExtension(committedTCBPartsBlSplOID, tcbParts.BlSpl))
	extensions = append(extensions, extension.NewBigIntExtension(committedTCBPartsTeeSplOID, tcbParts.TeeSpl))
	extensions = append(extensions, extension.NewBigIntExtension(committedTCBPartsSnpSplOID, tcbParts.SnpSpl))
	extensions = append(extensions, extension.NewBigIntExtension(committedTCBPartsUcodeSplOID, tcbParts.UcodeSpl))

	extensions = append(extensions, extension.NewBigIntExtension(currentBuildOID, report.CurrentBuild))
	extensions = append(extensions, extension.NewBigIntExtension(currentMinorOID, report.CurrentMinor))
	extensions = append(extensions, extension.NewBigIntExtension(currentMajorOID, report.CurrentMajor))
	extensions = append(extensions, extension.NewBigIntExtension(committedBuildOID, report.CommittedBuild))
	extensions = append(extensions, extension.NewBigIntExtension(committedMinorOID, report.CommittedMinor))
	extensions = append(extensions, extension.NewBigIntExtension(committedMajorOID, report.CommittedMajor))

	tcbParts = kds.DecomposeTCBVersion(kds.TCBVersion(report.LaunchTcb))
	extensions = append(extensions, extension.NewBigIntExtension(launchTCBPartsBlSplOID, tcbParts.BlSpl))
	extensions = append(extensions, extension.NewBigIntExtension(launchTCBPartsTeeSplOID, tcbParts.TeeSpl))
	extensions = append(extensions, extension.NewBigIntExtension(launchTCBPartsSnpSplOID, tcbParts.SnpSpl))
	extensions = append(extensions, extension.NewBigIntExtension(launchTCBPartsUcodeSplOID, tcbParts.UcodeSpl))

	return extension.ConvertExtensions(extensions)
}
