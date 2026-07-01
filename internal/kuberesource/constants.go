// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kuberesource

const (
	annotationPrefix     = "contrast.edgeless.systems/"
	kataAnnotationPrefix = "io.katacontainers.config.hypervisor."
)

const (
	// ContrastRoleAnnotationKey assigns a role to the annotated pod.
	//
	// This is used to determine whether a given pod may act as Coordinator.
	ContrastRoleAnnotationKey = annotationPrefix + "pod-role"

	// ExposeServiceAnnotationKey is the annotation key used to specify whether a Service should be exposed via a LoadBalancer.
	ExposeServiceAnnotationKey = annotationPrefix + "expose-service"

	// ImageStoreSizeAnnotationKey is the annotation key used to configure the size of the image store volume.
	ImageStoreSizeAnnotationKey = annotationPrefix + "image-store-size"

	// SecurePVAnnotationKey is the annotation key used to specify a secure persistent volume for a pod.
	SecurePVAnnotationKey = annotationPrefix + "secure-pv"

	// SkipInitializerAnnotationKey is the annotation key used to specify whether a pod should skip the Contrast initializer injection.
	SkipInitializerAnnotationKey = annotationPrefix + "skip-initializer"

	// SmAdminInterfaceAnnotationKey is the annotation key used to specify the port of the service mesh admin interface.
	SmAdminInterfaceAnnotationKey = annotationPrefix + "servicemesh-admin-interface-port"

	// SmEgressConfigAnnotationKey is the annotation key used to specify the egress configuration of the service mesh.
	SmEgressConfigAnnotationKey = annotationPrefix + "servicemesh-egress"

	// SmIngressConfigAnnotationKey is the annotation key used to specify the ingress configuration of the service mesh.
	SmIngressConfigAnnotationKey = annotationPrefix + "servicemesh-ingress"

	// WorkloadSecretIDAnnotationKey is the annotation key used to specify the workload secret ID for a pod.
	WorkloadSecretIDAnnotationKey = annotationPrefix + "workload-secret-id"
)

const (
	// GuestPolicyAnnotationKey is the annotation key used to specify the guest policy for a pod.
	GuestPolicyAnnotationKey = kataAnnotationPrefix + "snp_guest_policy_"

	// IDAuthAnnotationKey is the annotation key used to specify the ID Authentication for a pod.
	IDAuthAnnotationKey = kataAnnotationPrefix + "snp_id_auth_"

	// IDBlockAnnotationKey is the annotation key used to specify the ID Block for a pod.
	IDBlockAnnotationKey = kataAnnotationPrefix + "snp_id_block_"

	// InitdataAnnotationKey as specified in: https://github.com/kata-containers/kata-containers/blob/f6ff9cf7176989d414bf3f45a5b0c0b9fdb1bf3a/src/libs/kata-types/src/annotations/mod.rs#L276
	InitdataAnnotationKey = kataAnnotationPrefix + "cc_init_data"
)

const (
	// TDXEnabledNodeLabel is the node-feature-discovery label that marks a node as TDX-capable.
	TDXEnabledNodeLabel = "feature.node.kubernetes.io/tdx.enabled"

	// MainRunnerNodeLabel restricts pods to the bare-metal nodes of our CI runner.
	MainRunnerNodeLabel = "ci.contrast.edgeless.systems/main-runner"
)
