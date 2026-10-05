[
    "UpdateInterfaceRequest",
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
    ] | unique,
    hostname: first(.[] | select(.kind == "CreateSandboxRequest") | .request.hostname)
} as $data

# Normalize everything
| $data.rpcs
| map(
    if .kind == "UpdateInterfaceRequest" then
        .request.interface.mtu = 1500
        | .request.interface.hwAddr = "11:22:33:44:55:66"
        | .request.interface.IPAddresses |= map(
            if .family == 1 then
                .address = "1111::2222:3333:4444:5555"
            else .
            end
        )
    elif .kind == "CreateSandboxRequest" then
        .request.dns[0] = "search default.svc.cluster.local svc.cluster.local cluster.local"
    elif .kind == "CreateContainerRequest" then
        # TODO(burgerdev): these fields should probably not be passed to the agent at all
        # and should be removed from the OCI request
        .request.OCI.Annotations |= with_entries(
            if .key == "io.katacontainers.pkg.oci.bundle_path"
                or .key == "io.kubernetes.cri.podsandbox.image-name"
            then .value = ""
            else .
            end
        )
        | .request.OCI.Version = "1.3.0"
    else
        .
    end
)
| walk(
    if type == "string" then

        # Hostnames of pods created by controllers end in generated segments
        # using the alphabet of k8s.io/apimachinery/pkg/util/rand: a 5-character
        # random suffix (<name>-<random>, e.g. DaemonSets, Jobs), preceded by
        # the pod-template-hash for ReplicaSets (<name>-<hash>-<random>).
        # Each generated segment is replaced by "ffff".
        gsub(
            $data.hostname;
            $data.hostname
            # The random suffix is always 5 characters long.
            # https://github.com/kubernetes/apiserver/blob/f6c8a21a437e3f6ed8a8bd0d520aba8578c51ede/pkg/storage/names/generate.go#L45
            | sub("-[bcdfghjklmnpqrstvwxz2456789]{5}$"; "-ffff")
            # The pod-template-hash is derived from the integer representation of a 32-bit hash, encoded as alphanumeric characters.
            # This integer representation is at most 10 characters long, and moer than 6 characters with overwhelming probability.
            # https://github.com/kubernetes/kubernetes/blob/def85dfe8aabea42fa1569ad5c4f60de5b976fdd/pkg/controller/controller_utils.go#L1413
            | sub("-[bcdfghjklmnpqrstvwxz2456789]{6,10}-ffff$"; "-ffff-ffff")
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
