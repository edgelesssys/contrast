package agent_policy

test_allow_storage_expected if {
	allow_storage([p_storage], expected_i_storage, bundle_id, sandbox_id)
}

test_allow_storage_different_registry if {
	i_storage := object.union(expected_i_storage, {"source": "registry.k8s.io/pause@sha256:b4b669f27933146227c9180398f99d8b3100637e4a0a1ccf804f8b12f4b9b8df"})
	allow_storage([p_storage], i_storage, bundle_id, sandbox_id)
}

test_allow_storage_bad_source if {
	i_storage_wrong_digest := object.union(expected_i_storage, {"source": "ghcr.io/edgelesssys/kubernetes/pause:3.6@sha256:0000000000000000000000000000000000000000000000000000000000000000"})
	not allow_storage([p_storage], i_storage_wrong_digest, bundle_id, sandbox_id)
	i_storage_no_digest := object.union(expected_i_storage, {"source": "ghcr.io/edgelesssys/kubernetes/pause:3.6"})
	not allow_storage([p_storage], i_storage_no_digest, bundle_id, sandbox_id)
}

test_allow_storage_bad_mount_point_regex if {
	not allow_storage([p_storage], p_storage, bundle_id, sandbox_id)
}

test_allow_storage_bad_mount_point_sensitive_location if {
	i_storage := object.union(expected_i_storage, {"mount_point": "/proc"})
	not allow_storage([p_storage], i_storage, bundle_id, sandbox_id)
}

test_allow_storage_bad_driver if {
	i_storage := object.union(expected_i_storage, {"driver": "scsi"})
	not allow_storage([p_storage], i_storage, bundle_id, sandbox_id)
}

test_allow_storage_bad_fstype if {
	i_storage := object.union(expected_i_storage, {"fstype": "tmpfs"})
	not allow_storage([p_storage], i_storage, bundle_id, sandbox_id)
}

test_allow_storage_bad_options if {
	i_storage := object.union(expected_i_storage, {"options": ["rw"]})
	not allow_storage([p_storage], i_storage, bundle_id, sandbox_id)
}

test_allow_storage_bad_driver_options if {
	i_storage := object.union(expected_i_storage, {"driver_options": ["wtf"]})
	not allow_storage([p_storage], i_storage, bundle_id, sandbox_id)
}

test_allow_storage_bad_fs_group if {
	i_storage := object.union(expected_i_storage, {"fs_group": "TODO"})
	not allow_storage([p_storage], i_storage, bundle_id, sandbox_id)
}

test_allow_storage_bad_shared if {
	i_storage := object.union(expected_i_storage, {"shared": true})
	not allow_storage([p_storage], i_storage, bundle_id, sandbox_id)
}

# TODO(burgerdev): source from genpolicy-settings.json
policy_data := {
	"common": {
		"cpath": "/run/kata-containers/shared/containers(?:/passthrough)?",
		"spath": "/run/kata-containers/sandbox/storage",
		"root_path": "/run/kata-containers/$(bundle-id)/rootfs",
		"sfprefix": "^$(cpath)/(watchable/)?$(bundle-id)-[a-z0-9]{16}-",
		"ip_p": "[0-9]{1,5}",
		"ipv4_a": "(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])",
		"svc_name_downward_env": "[A-Z](?:[A-Z0-9_]{0,61}[A-Z0-9])?",
		"dns_label": "[a-zA-Z0-9_\\.\\-]+",
		"default_caps": [],
		"privileged_caps": [],
	},
	"cluster_config": {},
	"containers": [],
	"devices": {},
	"sandbox": {},
	"request_defaults": {},
}

bundle_id := "foo"
sandbox_id := "bar"

p_storage := {
	"driver": "image_guest_pull",
	"driver_options": [],
	"source": "ghcr.io/edgelesssys/kubernetes/pause:3.6@sha256:b4b669f27933146227c9180398f99d8b3100637e4a0a1ccf804f8b12f4b9b8df",
	"fstype": "overlay",
	"options": [],
	"mount_point": "^$(root_path)$",
	"fs_group": null,
	"shared": false,
}

expected_i_storage := {
	"driver": "image_guest_pull",
	"driver_options": [],
	"source": "ghcr.io/edgelesssys/kubernetes/pause:3.6@sha256:b4b669f27933146227c9180398f99d8b3100637e4a0a1ccf804f8b12f4b9b8df",
	"fstype": "overlay",
	"options": [],
	"mount_point": "/run/kata-containers/foo/rootfs",
	"fs_group": null,
	"shared": false,
}
