// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package qtest

const (
	pciConfigAddressPort = 0xcf8
	pciConfigDataPort    = 0xcfc
)

// PCIConfigWrite32 writes a 32-bit value to a PCI configuration register.
func (c *Client) PCIConfigWrite32(bus, device, function, offset, value uint32) error {
	if err := c.outL(pciConfigAddressPort, pciConfigAddress(bus, device, function, offset)); err != nil {
		return err
	}
	return c.outL(pciConfigDataPort, value)
}

// PCIConfigWrite8 writes an 8-bit value to a PCI configuration register.
func (c *Client) PCIConfigWrite8(bus, device, function, offset uint32, value byte) error {
	if err := c.outL(pciConfigAddressPort, pciConfigAddress(bus, device, function, offset)); err != nil {
		return err
	}
	return c.outB(pciConfigDataPort+uint16(offset&3), value)
}

func pciConfigAddress(bus, device, function, offset uint32) uint32 {
	return 0x80000000 | bus<<16 | device<<11 | function<<8 | offset&0xfc
}
