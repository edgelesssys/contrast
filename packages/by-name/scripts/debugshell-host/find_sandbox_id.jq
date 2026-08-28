first(
  .[]
  | select(
      .status.metadata.name == $podname
      and .status.metadata.namespace == $namespace
      and .status.state == "SANDBOX_READY"
    )
) // error("no SANDBOX_READY pod found for \($podname)/\($namespace)")
| .status.id
