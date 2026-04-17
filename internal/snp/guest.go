// Translated from the Python sev-snp-measure tool:
// Copyright 2022- IBM Inc. All rights reserved
// SPDX-License-Identifier: Apache-2.0
// Source: https://github.com/virtee/sev-snp-measure/blob/46664d0347fb07c5ac2cb8ab5bf5aebc09fc67ab/sevsnpmeasure/guest.py

// Package snp implements AMD SEV-SNP launch measurement calculation.
// Only the SNP mode required by the Kata calculateSnpLaunchDigest nix package
// is implemented (--mode snp, VMM type QEMU).
package snp

import "fmt"

// CalcSNPLaunchDigest calculates the SEV-SNP launch measurement digest.
//
//   - ovmfPath: path to the OVMF firmware binary
//   - vcpus:    number of guest vCPUs
//   - vcpuSig:  CPUID signature (e.g. from CPUSigs["EPYC-Milan"])
//   - kernel:   path to kernel bzImage (empty string to skip)
//   - initrd:   path to initrd (empty string to skip)
//   - append:   kernel command line string (empty string for none)
//   - guestFeatures: guest feature flags (default 0x1)
func CalcSNPLaunchDigest(ovmfPath string, vcpus int, vcpuSig uint32,
	kernel, initrd, appendStr string, guestFeatures uint64,
) ([LaunchDigestSize]byte, error) {
	ovmf, err := NewOVMF(ovmfPath)
	if err != nil {
		return [LaunchDigestSize]byte{}, fmt.Errorf("parsing OVMF: %w", err)
	}

	gctx := NewGCTX()

	// Measure the OVMF firmware pages.
	data := ovmf.Data()
	if len(data)%pageSize != 0 {
		return [LaunchDigestSize]byte{}, fmt.Errorf("OVMF size %d is not page-aligned", len(data))
	}
	gctx.UpdateNormalPages(ovmf.GPA(), data)

	// Measure metadata sections (zero pages, secrets, CPUID, kernel hashes).
	var sevHashes *SevHashes
	if kernel != "" {
		sevHashes, err = NewSevHashes(kernel, initrd, appendStr)
		if err != nil {
			return [LaunchDigestSize]byte{}, fmt.Errorf("computing kernel hashes: %w", err)
		}
	}
	if err := updateMetadataPages(gctx, ovmf, sevHashes); err != nil {
		return [LaunchDigestSize]byte{}, err
	}

	// Measure VMSA pages (one per vCPU).
	apEIP, err := ovmf.SEVESResetEIP()
	if err != nil {
		return [LaunchDigestSize]byte{}, fmt.Errorf("reading OVMF reset EIP: %w", err)
	}
	for _, page := range VMSAPages(vcpus, apEIP, guestFeatures, vcpuSig) {
		gctx.UpdateVMSAPage(page)
	}

	return gctx.LaunchDigest(), nil
}

// updateMetadataPages processes each OVMF SEV metadata section in order,
// matching the logic in guest.py snp_update_metadata_pages for VMM type QEMU.
func updateMetadataPages(gctx *GCTX, ovmf *OVMF, sevHashes *SevHashes) error {
	for _, desc := range ovmf.MetadataItems() {
		if err := updateSection(gctx, ovmf, desc, sevHashes); err != nil {
			return err
		}
	}

	if sevHashes != nil && !ovmf.HasMetadataSection(SectionTypeSNPKernelHash) {
		return fmt.Errorf("kernel specified but OVMF metadata doesn't include SNP_KERNEL_HASHES section")
	}
	return nil
}

// updateSection handles a single metadata section entry (QEMU VMM type).
func updateSection(gctx *GCTX, ovmf *OVMF, desc MetadataSection, sevHashes *SevHashes) error {
	gpa := uint64(desc.GPA)
	size := int(desc.Size)

	switch desc.Type {
	case SectionTypeSNPSecMem:
		// QEMU uses zero pages (GCE uses unmeasured, but we only support QEMU).
		gctx.UpdateZeroPages(gpa, size)

	case SectionTypeSNPSecrets:
		gctx.UpdateSecretsPage(gpa)

	case SectionTypeCPUID:
		gctx.UpdateCPUIDPage(gpa)

	case SectionTypeSNPKernelHash:
		if sevHashes != nil {
			hashGPA, err := ovmf.SEVHashesTableGPA()
			if err != nil {
				return fmt.Errorf("getting SEV hashes table GPA: %w", err)
			}
			offsetInPage := hashGPA & 0xfff
			page := sevHashes.ConstructPage(offsetInPage)
			if size != 4096 {
				return fmt.Errorf("SNP_KERNEL_HASHES section size %d != 4096", size)
			}
			gctx.UpdateNormalPages(gpa, page[:])
		} else {
			gctx.UpdateZeroPages(gpa, size)
		}

	case SectionTypeSVSMCAA:
		gctx.UpdateZeroPages(gpa, size)

	default:
		return fmt.Errorf("unknown OVMF metadata section type 0x%x", uint32(desc.Type))
	}
	return nil
}
