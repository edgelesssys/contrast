// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
)

// activation is a unit that can be started, and the reason why.
type activation struct {
	unit   string
	reason string
}

const (
	// reasonBoot marks the units that are started during boot.
	reasonBoot = ""
	// reasonPerpetual marks the units that are always active and never get a job.
	reasonPerpetual = "perpetual"
)

// kernelRequests are the targets the service manager starts when the kernel signals it, see systemd(1).
// This is independent of whether the hardware to trigger the signal is present.
var kernelRequests = []activation{
	{"ctrl-alt-del.target", "kernel request on Ctrl-Alt-Del"},
	{"kbrequest.target", "kernel request on Alt-ArrowUp"},
}

// analysis searches for the units that can be started in a stage.
//
// An activation starts a unit in a transaction of its own. The transaction is computed by `systemd --test`,
// and every unit that gets a start job in it is recorded as reached.
type analysis struct {
	cfg config
	// transaction returns the dump of `systemd --test` for starting a unit, see stage.transaction.
	transaction func(ctx context.Context, unit string) (*dump, error)
	reviewed    *reviewedCommands
	// units are the units loaded by systemd so far, by primary name.
	units map[string]*unitInfo
	// aliases maps an alias to the primary name of its unit.
	aliases map[string]string

	queue  []activation
	queued map[string]bool
	// reached maps the primary name of every unit that can be started to the reason why.
	reached map[string]string
}

func newAnalysis(cfg config, transaction func(ctx context.Context, unit string) (*dump, error)) *analysis {
	return &analysis{
		cfg:         cfg,
		transaction: transaction,
		reviewed:    newReviewedCommands(cfg.ReviewedCommands),
		units:       map[string]*unitInfo{},
		aliases:     map[string]string{},
		queued:      map[string]bool{},
		reached:     map[string]string{},
	}
}

// run returns every unit that can be started in the stage, mapped to the reason why.
func (a *analysis) run(ctx context.Context) (map[string]string, error) {
	a.enqueue(activation{a.cfg.Target, reasonBoot})
	a.enqueue(kernelRequests...)
	if err := checkNoDeviceDeps(filepath.Join(a.cfg.Etc, "systemd/system")); err != nil {
		return nil, err
	}

	for len(a.queue) > 0 {
		next := a.queue[0]
		a.queue = a.queue[1:]
		if err := a.start(ctx, next); err != nil {
			return nil, err
		}
	}
	if unused := a.reviewed.unused(); len(unused) > 0 {
		return nil, fmt.Errorf("reviewed commands weren't encountered, remove them: %q", unused)
	}
	return a.reached, nil
}

// enqueue adds activations to the queue, unless their unit has been queued before.
func (a *analysis) enqueue(activations ...activation) {
	for _, act := range activations {
		// Device units only reflect the hardware.
		if a.queued[act.unit] || strings.HasSuffix(act.unit, ".device") {
			continue
		}
		a.queued[act.unit] = true
		a.queue = append(a.queue, act)
	}
}

// start computes the transaction for the given activation and records its units as reached.
func (a *analysis) start(ctx context.Context, act activation) error {
	d, err := a.transaction(ctx, act.unit)
	if err != nil {
		return err
	}
	if err := checkInactive(d); err != nil {
		return err
	}
	maps.Copy(a.units, d.units)
	maps.Copy(a.aliases, d.aliases)

	if info := a.units[a.primary(act.unit)]; info == nil || info.loadState != "loaded" {
		// systemd falls back to another target if the unit can't be loaded.
		logf("skipping %s (%s), it can't be loaded", act.unit, act.reason)
		return nil
	}
	if err := a.reach(act.unit, act.reason); err != nil {
		return err
	}

	pulledIn := "pulled in by " + displayName(a.primary(act.unit))
	if act.reason == reasonBoot {
		pulledIn = reasonBoot
	}
	for _, job := range slices.Sorted(slices.Values(d.jobs)) {
		if err := a.reach(job, pulledIn); err != nil {
			return err
		}
	}
	for _, unit := range slices.Sorted(maps.Keys(d.units)) {
		if !d.units[unit].perpetual {
			continue
		}
		if err := a.reach(unit, reasonPerpetual); err != nil {
			return err
		}
	}
	return nil
}

// reach records a loaded unit as reached and queues its activations, unless it has been reached before.
func (a *analysis) reach(unit, reason string) error {
	name := a.primary(unit)
	if _, ok := a.reached[name]; ok || strings.HasSuffix(name, ".device") {
		return nil
	}
	info := a.units[name]
	if info == nil {
		return fmt.Errorf("%s was reached, but systemd didn't load it", name)
	}
	a.reached[name] = reason

	activations, err := unitActivations(name, info, a.reviewed)
	if err != nil {
		return err
	}
	a.enqueue(activations...)

	// The bus and device managers start further units once they are running.
	switch name {
	case "dbus.service", "dbus-broker.service":
		activations, err = dbusActivations(a.cfg.DBusServiceDirs)
	case "systemd-udevd.service":
		activations, err = udevActivations(filepath.Join(a.cfg.Etc, "udev/rules.d"), filepath.Join(a.cfg.Systemd, "lib/udev"), a.reviewed)
	default:
		activations, err = nil, nil
	}
	if err != nil {
		return err
	}
	a.enqueue(activations...)
	return nil
}

// primary returns the primary name of a unit to resolve aliases such as dbus-org.freedesktop.login1.service.
func (a *analysis) primary(unit string) string {
	if primary, ok := a.aliases[unit]; ok {
		return primary
	}
	return unit
}

// checkInactive checks that no unit of the image is active in the analysis environment.
//
// systemd doesn't add a start job for a unit that is already active, so such a unit would be missing from the result.
func checkInactive(d *dump) error {
	for _, name := range slices.Sorted(maps.Keys(d.units)) {
		info := d.units[name]
		if info.activeState != "inactive" && !info.perpetual && info.fragmentPath != "" {
			return fmt.Errorf("%s is %s in the analysis environment, so it's unknown whether it would be started",
				name, info.activeState)
		}
	}
	return nil
}

// instance is the instance name used for templates that are instantiated at runtime, e.g. per connection or device.
// It must not clash with the instance of a unit in an image, as such a unit would be displayed as its template.
const instance = "reachable-units-placeholder"

// instantiate turns foo@.service into foo@reachable-units-placeholder.service, as systemd can only load instances.
func instantiate(template string) string {
	return strings.Replace(template, "@.", "@"+instance+".", 1)
}

// displayName turns foo@reachable-units-placeholder.service back into foo@.service.
func displayName(unit string) string {
	return strings.Replace(unit, "@"+instance+".", "@.", 1)
}
