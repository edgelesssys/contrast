// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package systemdunits

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedExpectedUnits(t *testing.T) {
	for name, golden := range map[string]string{"default": expectedUnits, "gpu": expectedUnitsGPU} {
		t.Run(name, func(t *testing.T) {
			units := parseExpected(golden)
			// Sanity check that both boot stages are covered.
			assert.True(t, units["initrd.target"])
			assert.True(t, units["kata-containers.target"])
			assert.True(t, units["kata-agent.service"])
		})
	}
}

func TestParseExpected(t *testing.T) {
	golden := `# initrd (initrd.target)
#
# generators:
#   systemd-fstab-generator
#   systemd-gpt-auto-generator (masked)
#
initrd.target
emergency.target (on failure of initrd-fs.target)

# system (kata-containers.target)
systemd-coredump@.service (accepted by systemd-coredump.socket)
`
	assert.Equal(t, map[string]bool{
		"initrd.target":             true,
		"emergency.target":          true,
		"systemd-coredump@.service": true,
	}, parseExpected(golden))
}

func TestParseJournal(t *testing.T) {
	out := `{"__CURSOR":"s=1","UNIT":"kata-agent.service","_PID":"1"}
{"__CURSOR":"s=2","UNIT":"sysroot.mount"}

{"__CURSOR":"s=3"}
`
	units, err := parseJournal(out)
	require.NoError(t, err)
	assert.Equal(t, []string{"kata-agent.service", "sysroot.mount"}, units)

	_, err = parseJournal("not json\n")
	assert.Error(t, err)
}

func TestParseShow(t *testing.T) {
	// Excerpt of the output of showCommand in a podvm.
	out := `Id=kata-agent.service
Perpetual=no
FragmentPath=/etc/systemd/system/kata-agent.service
SourcePath=
ActiveEnterTimestampMonotonic=12345
InactiveExitTimestampMonotonic=12000

Id=never-started.service
Perpetual=no
FragmentPath=/etc/systemd/system/never-started.service
SourcePath=
ActiveEnterTimestampMonotonic=0
InactiveExitTimestampMonotonic=0

Id=-.slice
Perpetual=yes
FragmentPath=
SourcePath=
ActiveEnterTimestampMonotonic=1
InactiveExitTimestampMonotonic=1

Id=started-but-failed.service
Perpetual=no
FragmentPath=/etc/systemd/system/started-but-failed.service
SourcePath=
ActiveEnterTimestampMonotonic=0
InactiveExitTimestampMonotonic=500

Id=var.mount
Perpetual=no
FragmentPath=/run/systemd/generator/var.mount
SourcePath=/etc/fstab
ActiveEnterTimestampMonotonic=100
InactiveExitTimestampMonotonic=90

Id=cri-containerd-e8605a91.scope
Perpetual=no
FragmentPath=/run/systemd/transient/cri-containerd-e8605a91.scope
SourcePath=
ActiveEnterTimestampMonotonic=200
InactiveExitTimestampMonotonic=190

Id=initrd-switch-root.service
Perpetual=no
FragmentPath=
SourcePath=
ActiveEnterTimestampMonotonic=50
InactiveExitTimestampMonotonic=40

Id=run-kata\x2dcontainers-image\x2dmemory-graph-overlay.mount
Perpetual=no
FragmentPath=
SourcePath=/proc/self/mountinfo
ActiveEnterTimestampMonotonic=300
InactiveExitTimestampMonotonic=290
`
	started, external := parseShow(out)
	assert.Equal(t, []string{
		"kata-agent.service",
		"started-but-failed.service",
		"var.mount",
		"cri-containerd-e8605a91.scope",
		"initrd-switch-root.service",
	}, started)
	assert.Equal(t, []string{`run-kata\x2dcontainers-image\x2dmemory-graph-overlay.mount`}, external)
}

func TestCheckUnits(t *testing.T) {
	expected := map[string]bool{
		"kata-agent.service":               true,
		"systemd-coredump@.service":        true,
		"systemd-veritysetup@root.service": true,
	}
	observed := []string{
		"kata-agent.service",
		"systemd-coredump@1-2-3.service",
		"systemd-veritysetup@root.service",
		"systemd-veritysetup@other.service",
		"evil.service",
		"evil.service",
		"dev-vda.device",
		"cri-containerd-abc.scope",
		"kubepods.slice",
	}
	unexpected, skipped := checkUnits(observed, expected)
	assert.Equal(t, []string{"evil.service", "systemd-veritysetup@other.service"}, unexpected)
	assert.Equal(t, []string{"cri-containerd-abc.scope", "dev-vda.device", "kubepods.slice"}, skipped)
}
