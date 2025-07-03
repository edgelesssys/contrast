# Coordinator High-Availability

This guide shows how to scale the Contrast Coordinator to more than one instance and make it highly available.

## Workflow

Deploy the Coordinator according to the [basic workflow](../deployment.md).
By default, there is only one Coordinator instance.
Verify that this Coordinator is in state `Ready`:

```sh
kubectl get pods -l app.kubernetes.io/name=coordinator
```

```raw
NAME            READY   STATUS    RESTARTS   AGE
coordinator-0   1/1     Running   0          11m
```

Next, we increase the number of instances to 3.
This will create additional Coordinator instances, one after another, as described in the [Kubernetes documentation for `StatefulSet`](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#deployment-and-scaling-guarantees).
After some time, the additional Coordinators should enter the `Ready` state, too.

```sh
kubectl scale statefulset/coordinator --replicas 3
kubectl wait statefulset/coordinator --for=jsonpath='{.status.readyReplicas}=3' --timeout=2m
kubectl get pods -l app.kubernetes.io/name=coordinator
```

```raw
statefulset.apps/coordinator scaled
statefulset.apps/coordinator condition met
NAME            READY   STATUS    RESTARTS   AGE
coordinator-0   1/1     Running   0          14m
coordinator-1   1/1     Running   0          107s
coordinator-2   1/1     Running   0          73s
```

You can now try deleting individual Coordinator pods, draining a node in a multi-node cluster, or scheduling a full rollout of the Coordinator.
Availability of the Coordinator endpoint shouldn't be affected, and all Coordinator pods should automatically recover.

```sh
kubectl rollout restart statefulset/coordinator
kubectl rollout status statefulset/coordinator -w
kubectl get pods -l app.kubernetes.io/name=coordinator

```

```raw
statefulset.apps/coordinator restarted
Waiting for 1 pods to be ready...
Waiting for partitioned roll out to finish: 1 out of 3 new pods have been updated...
Waiting for 1 pods to be ready...
Waiting for 1 pods to be ready...
Waiting for 1 pods to be ready...
Waiting for partitioned roll out to finish: 2 out of 3 new pods have been updated...
Waiting for 1 pods to be ready...
Waiting for 1 pods to be ready...
Waiting for 1 pods to be ready...
partitioned roll out complete: 3 new pods have been updated...
NAME            READY   STATUS    RESTARTS   AGE
coordinator-0   1/1     Running   0          41s
coordinator-1   1/1     Running   0          71s
coordinator-2   1/1     Running   0          99s
```

## How it works

<!-- TODO(burgerdev): link to Coordinator page after https://github.com/edgelesssys/contrast/pull/1436 landed. -->

Newly started (or restarted) Coordinator instances try to recover from other Coordinator instances in the cluster.
As long as a single Coordinator is initialized, the other instances eventually recover from it.
`StatefulSet` semantics guarantee that Coordinator pods are started predictably, and only after all existing Coordinators are recovered.
