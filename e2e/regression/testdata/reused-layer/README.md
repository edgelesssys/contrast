# Reused image layers

This regression test has two images that share a layer, but in different order:

```sh
crane manifest ghcr.io/edgelesssys/reused-layer/b:latest@sha256:58588091715d842894ca69769be2ee0afaa39a94bebad094f90772dd62f13fa0 | jq .layers
```

```json
[
  {
    "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip",
    "size": 45,
    "digest": "sha256:85cea451eec057fa7e734548ca3ba6d779ed5836a3f9de14b8394575ef0d7d8e"
  },
  {
    "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip",
    "size": 742873,
    "digest": "sha256:48b39d63b6ee48560abb8ef6f65ae95ba4b863e3eaa38f65829d6b0d64930223"
  }
]
```

```sh
crane manifest ghcr.io/edgelesssys/reused-layer/a:latest@sha256:a81cb597db43ac54dd1e2e3c95c7198dd9dafda1ceded5b3adbc4e830f47d92e | jq .layers
```

```json
[
  {
    "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip",
    "size": 742873,
    "digest": "sha256:48b39d63b6ee48560abb8ef6f65ae95ba4b863e3eaa38f65829d6b0d64930223"
  },
  {
    "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip",
    "size": 45,
    "digest": "sha256:85cea451eec057fa7e734548ca3ba6d779ed5836a3f9de14b8394575ef0d7d8e"
  }
]
```

If layer caching is applied to the stacked previous layers, instead of just to the layer at hand, the mounted image `a` won't have the busybox binary.
Such a bug was introduced in c3b71a8696926ded472eb473f7413f5786df851f, this test guards against reintroducing it.

These images were generated from <https://github.com/burgerdev/weird-images/commit/2f8300189ee275358ef5f6d7dcdfc023ee776527>.
