// Translated from the Python sev-snp-measure tool:
// Copyright 2022- IBM Inc. All rights reserved
// SPDX-License-Identifier: Apache-2.0
// Source: https://github.com/virtee/sev-snp-measure/blob/46664d0347fb07c5ac2cb8ab5bf5aebc09fc67ab/sevsnpmeasure/vmsa.py

package snp

import "encoding/binary"

// SevEsSaveArea is the VMSA page layout (AMD APM Vol 2, Table B-4).
// Total size must be exactly 4096 bytes. We represent it as a plain
// byte array and write fields at the correct offsets.
//
// Offset map (all LE, _pack_=1):
//
//	0x000  es      (VmcbSeg: sel u16, attrib u16, limit u32, base u64) = 16 B
//	0x010  cs
//	0x020  ss
//	0x030  ds
//	0x040  fs
//	0x050  gs
//	0x060  gdtr
//	0x070  ldtr
//	0x080  idtr
//	0x090  tr
//	0x0a0  vmpl0_ssp u64
//	0x0a8  vmpl1_ssp u64
//	0x0b0  vmpl2_ssp u64
//	0x0b8  vmpl3_ssp u64
//	0x0c0  u_cet u64
//	0x0c8  reserved (2 B)
//	0x0ca  vmpl u8
//	0x0cb  cpl u8
//	0x0cc  reserved (4 B)
//	0x0d0  efer u64
//	0x0d8  reserved (104 B)
//	0x140  xss u64
//	0x148  cr4 u64
//	0x150  cr3 u64
//	0x158  cr0 u64
//	0x160  dr7 u64
//	0x168  dr6 u64
//	0x170  rflags u64
//	0x178  rip u64
//	0x180  dr0..dr3 (4×8 B)
//	0x1a0  dr0..dr3_addr_mask (4×8 B)
//	0x1c0  reserved (24 B)
//	0x1d8  rsp u64
//	0x1e0  s_cet u64
//	0x1e8  ssp u64
//	0x1f0  isst_addr u64
//	0x1f8  rax u64
//	0x200  star u64
//	0x208  lstar u64
//	0x210  cstar u64
//	0x218  sfmask u64
//	0x220  kernel_gs_base u64
//	0x228  sysenter_cs u64
//	0x230  sysenter_esp u64
//	0x238  sysenter_eip u64
//	0x240  cr2 u64
//	0x248  reserved (32 B)
//	0x268  g_pat u64
//	0x270  dbgctrl u64
//	0x278  br_from u64
//	0x280  br_to u64
//	0x288  last_excp_from u64
//	0x290  last_excp_to u64
//	0x298  reserved (80 B)
//	0x2e8  pkru u32
//	0x2ec  tsc_aux u32
//	0x2f0  reserved (24 B)
//	0x308  rcx u64
//	0x310  rdx u64
//	0x318  rbx u64
//	0x320  reserved u64
//	0x328  rbp u64
//	0x330  rsi u64
//	0x338  rdi u64
//	0x340  r8  u64
//	0x348  r9  u64
//	0x350  r10 u64
//	0x358  r11 u64
//	0x360  r12 u64
//	0x368  r13 u64
//	0x370  r14 u64
//	0x378  r15 u64
//	0x380  reserved (16 B)
//	0x390  guest_exit_info_1 u64
//	0x398  guest_exit_info_2 u64
//	0x3a0  guest_exit_int_info u64
//	0x3a8  guest_nrip u64
//	0x3b0  sev_features u64
//	0x3b8  vintr_ctrl u64
//	0x3c0  guest_exit_code u64
//	0x3c8  virtual_tom u64
//	0x3d0  tlb_id u64
//	0x3d8  pcpu_id u64
//	0x3e0  event_inj u64
//	0x3e8  xcr0 u64
//	0x3f0  reserved (16 B)
//	--- Floating Point Area ---
//	0x400  x87_dp u64
//	0x408  mxcsr u32
//	0x40c  x87_ftw u16
//	0x40e  x87_fsw u16
//	0x410  x87_fcw u16
//	0x412  x87_fop u16
//	0x414  x87_ds u16
//	0x416  x87_cs u16
//	0x418  x87_rip u64
//	0x420  fpreg_x87 (80 B)
//	0x470  fpreg_xmm (256 B)
//	0x570  fpreg_ymm (256 B)
//	0x670  manual_padding (2448 B)
//	0xfff  (end)

// vmcbSeg writes a VmcbSeg at the given offset in buf.
func vmcbSeg(buf []byte, off int, selector, attrib uint16, limit uint32, base uint64) { //nolint:unparam
	binary.LittleEndian.PutUint16(buf[off:], selector)
	binary.LittleEndian.PutUint16(buf[off+2:], attrib)
	binary.LittleEndian.PutUint32(buf[off+4:], limit)
	binary.LittleEndian.PutUint64(buf[off+8:], base)
}

// BuildVMSASaveArea builds a 4096-byte VMSA save-area page for QEMU/SNP.
// eip is the reset EIP (BSP uses 0xfffffff0, APs use the OVMF reset EIP).
// sevFeatures is the guest_features value (default 0x1).
// vcpuSig is the CPUID signature from CPUSigs table.
func BuildVMSASaveArea(eip uint32, sevFeatures uint64, vcpuSig uint32) [4096]byte {
	var page [4096]byte
	b := page[:]

	// QEMU VMM type values
	csFlags := uint16(0x9b)
	ssFlags := uint16(0x93)
	trFlags := uint16(0x8b)
	rdx := uint64(vcpuSig)
	mxcsr := uint32(0x1f80)
	fcw := uint16(0x37f)
	gPat := uint64(0x7040600070406) // PAT MSR, AMD APM Vol 2 Section A.3

	// Segments at offsets 0x00 .. 0x9f
	//  es  = 0x000
	vmcbSeg(b, 0x000, 0, 0x93, 0xffff, 0)
	//  cs  = 0x010  selector=0xf000, base = eip & 0xffff0000, rip = eip & 0xffff
	vmcbSeg(b, 0x010, 0xf000, csFlags, 0xffff, uint64(eip)&0xffff0000)
	//  ss  = 0x020
	vmcbSeg(b, 0x020, 0, ssFlags, 0xffff, 0)
	//  ds  = 0x030
	vmcbSeg(b, 0x030, 0, 0x93, 0xffff, 0)
	//  fs  = 0x040
	vmcbSeg(b, 0x040, 0, 0x93, 0xffff, 0)
	//  gs  = 0x050
	vmcbSeg(b, 0x050, 0, 0x93, 0xffff, 0)
	//  gdtr = 0x060
	vmcbSeg(b, 0x060, 0, 0, 0xffff, 0)
	//  ldtr = 0x070
	vmcbSeg(b, 0x070, 0, 0x82, 0xffff, 0)
	//  idtr = 0x080
	vmcbSeg(b, 0x080, 0, 0, 0xffff, 0)
	//  tr   = 0x090
	vmcbSeg(b, 0x090, 0, trFlags, 0xffff, 0)

	// efer = 0x0d0
	binary.LittleEndian.PutUint64(b[0x0d0:], 0x1000) // KVM enables EFER_SVME

	// cr4 = 0x148
	binary.LittleEndian.PutUint64(b[0x148:], 0x40) // X86_CR4_MCE
	// cr3 = 0x150
	binary.LittleEndian.PutUint64(b[0x150:], 0)
	// cr0 = 0x158
	binary.LittleEndian.PutUint64(b[0x158:], 0x10)
	// dr7 = 0x160
	binary.LittleEndian.PutUint64(b[0x160:], 0x400)
	// dr6 = 0x168
	binary.LittleEndian.PutUint64(b[0x168:], 0xffff0ff0)
	// rflags = 0x170
	binary.LittleEndian.PutUint64(b[0x170:], 0x2)
	// rip = 0x178  (eip & 0xffff)
	binary.LittleEndian.PutUint64(b[0x178:], uint64(eip)&0xffff)

	// g_pat = 0x268
	binary.LittleEndian.PutUint64(b[0x268:], gPat)

	// rdx = 0x310
	binary.LittleEndian.PutUint64(b[0x310:], rdx)

	// sev_features = 0x3b0
	binary.LittleEndian.PutUint64(b[0x3b0:], sevFeatures)

	// xcr0 = 0x3e8
	binary.LittleEndian.PutUint64(b[0x3e8:], 0x1)

	// mxcsr = 0x408
	binary.LittleEndian.PutUint32(b[0x408:], mxcsr)

	// x87_fcw = 0x410
	binary.LittleEndian.PutUint16(b[0x410:], fcw)

	return page
}

const bspEIP = uint32(0xfffffff0)

// VMSAPages returns vcpus VMSA pages for an SNP guest.
// vcpu 0 is the BSP (uses bspEIP = 0xfffffff0).
// vcpus 1..N-1 are APs (use apEIP, i.e. ovmf.SEVESResetEIP()).
func VMSAPages(vcpus int, apEIP uint32, sevFeatures uint64, vcpuSig uint32) [][4096]byte {
	bsp := BuildVMSASaveArea(bspEIP, sevFeatures, vcpuSig)
	ap := BuildVMSASaveArea(apEIP, sevFeatures, vcpuSig)

	pages := make([][4096]byte, vcpus)
	for i := range vcpus {
		if i == 0 {
			pages[i] = bsp
		} else {
			pages[i] = ap
		}
	}
	return pages
}
