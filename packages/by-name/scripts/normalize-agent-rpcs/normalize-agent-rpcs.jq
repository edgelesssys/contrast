[
    "UpdateInterfaceRequest",
    "UpdateRoutesRequest",
    "CreateSandboxRequest",
    "CreateContainerRequest",
    "RemoveContainerRequest",
    "ExecProcessRequest"
] as $kinds

# Filter RPCs by kind
| [
    .[]
    | . as $rpc
    | select($kinds | index($rpc.kind))
]

# Collect sandbox/container IDs
| {
    rpcs: .,
	ids: reduce (
		.[]
		| .request.sandbox_id?,
		  .request.container_id?
		| select(. != null and . != "")
	) as $id
		([];
		 if index($id) == null then . + [$id] else . end
	),
	namespaces: [
		.[]
		| .request.OCI.Annotations["io.kubernetes.cri.sandbox-namespace"]?
		| select(. != null and . != "")
	] | unique
} as $data

# Normalize everything
| $data.rpcs
| walk(
    if type == "string" then

        # Replace tailscale id
        gsub(
            "tail[0-9a-f]{5}";
            "tail00000"
        )

		# Replace IDs with deterministic 64-character values:
        # first ID -> 0000...
        # second ID -> 1111...
        # third ID -> 2222...
        | reduce ($data.ids | to_entries[]) as $entry (.;
            gsub(
                $entry.value;
                (($entry.key | tostring) * 64)
            )
        )

		# Namespaces
		| reduce $data.namespaces[] as $namespace (.;
			gsub($namespace; "default")
		)

        # UUIDs
        | gsub(
            "[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}";
            "00000000-0000-0000-0000-000000000000"
        )

        # UUIDs with '-' replaced by '_'
        | gsub(
            "[0-9a-fA-F]{8}_[0-9a-fA-F]{4}_[0-9a-fA-F]{4}_[0-9a-fA-F]{4}_[0-9a-fA-F]{12}";
            "00000000_0000_0000_0000_000000000000"
        )

        # 16-character hex IDs used in Kata paths
        | gsub(
            "-[0-9a-fA-F]{16}-";
            "-0000000000000000-"
        )

		# IPv4 addresses
		| gsub(
            "([0-9]{1,3}\\.){3}[0-9]{1,3}";
            "10.0.0.0"
        )

		# IPv6 addresses
		| gsub(
            "^([0-9a-f]{4}::?){1,7}[0-9a-f]{4}$";
            "1111::2222:3333:4444:5555"
        )

		# MAC addresses
		| gsub(
            "^([0-9a-f]{2}:){5}[0-9a-f]{2}$";
            "11:22:33:44:55:66"
        )

    else
        .
    end
)

# Add allowed field
| map(. + {allowed: true})

# Sort only OCI Process environment variables
| map(
    if .request.OCI.Process.Env? then
        .request.OCI.Process.Env |= sort
    elif .request.process.Env? then
        .request.process.Env |= sort
    else
        .
    end
)
