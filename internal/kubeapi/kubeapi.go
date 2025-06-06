// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kubeapi

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
)

type (
	// Pod is a Kubernetes Pod.
	Pod = corev1.Pod
	// Deployment is a Kubernetes Deployment.
	Deployment = appsv1.Deployment
	// StatefulSet is a Kubernetes StatefulSet.
	StatefulSet = appsv1.StatefulSet
	// ReplicaSet is a Kubernetes ReplicaSet.
	ReplicaSet = appsv1.ReplicaSet
	// DaemonSet is a Kubernetes DaemonSet.
	DaemonSet = appsv1.DaemonSet
	// Job is a Kubernetes Job.
	Job = batchv1.Job
	// CronJob is a Kubernetes CronJob.
	CronJob = batchv1.CronJob
	// ReplicationController is a Kubernetes ReplicationController.
	ReplicationController = corev1.ReplicationController
)

// UnmarshalK8SResources unmarshals a Kubernetes resource into a list of objects that can be
// type casted to a Kubernetes resource.
func UnmarshalK8SResources(data []byte) ([]any, error) {
	objs, err := UnmarshalUnstructuredK8SResource(data)
	if err != nil {
		return nil, err
	}
	var result []any
	for _, obj := range objs {
		switch obj.GetKind() {
		case "Pod":
			var pod corev1.Pod
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &pod)
			if err != nil {
				return nil, err
			}
			result = append(result, &pod)
		case "Deployment":
			var deployment appsv1.Deployment
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &deployment)
			if err != nil {
				return nil, err
			}
			result = append(result, &deployment)
		case "StatefulSet":
			var statefulSet appsv1.StatefulSet
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &statefulSet)
			if err != nil {
				return nil, err
			}
			result = append(result, &statefulSet)
		case "ReplicaSet":
			var replicaSet appsv1.ReplicaSet
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &replicaSet)
			if err != nil {
				return nil, err
			}
			result = append(result, &replicaSet)
		case "DaemonSet":
			var daemonSet appsv1.DaemonSet
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &daemonSet)
			if err != nil {
				return nil, err
			}
			result = append(result, &daemonSet)
		case "Job":
			var job batchv1.Job
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &job)
			if err != nil {
				return nil, err
			}
			result = append(result, &job)
		case "CronJob":
			var cronJob batchv1.CronJob
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &cronJob)
			if err != nil {
				return nil, err
			}
			result = append(result, &cronJob)
		case "ReplicationController":
			var replicationController corev1.ReplicationController
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &replicationController)
			if err != nil {
				return nil, err
			}
			result = append(result, &replicationController)
		}
	}
	return result, nil
}

// UnmarshalUnstructuredK8SResource parses the input YAML into unstructured Kubernetes resources.
func UnmarshalUnstructuredK8SResource(data []byte) ([]*unstructured.Unstructured, error) {
	documentsData, err := splitYAML(data)
	if err != nil {
		return nil, fmt.Errorf("splitting YAML into multiple documents: %w", err)
	}

	var objects []*unstructured.Unstructured
	for _, documentData := range documentsData {
		obj := &unstructured.Unstructured{}
		decoder := serializer.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
		_, _, err := decoder.Decode(documentData, nil, obj)
		if err != nil {
			return nil, err
		}
		objects = append(objects, obj)
	}

	return objects, nil
}

// splitYAML splits a YAML multidoc into a slice of multiple YAML docs.
func splitYAML(resources []byte) ([][]byte, error) {
	dec := yaml.NewDecoder(bytes.NewReader(resources))
	var res [][]byte
	for {
		var value any
		err := dec.Decode(&value)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		valueBytes, err := yaml.Marshal(value)
		if err != nil {
			return nil, err
		}
		res = append(res, valueBytes)
	}
	return res, nil
}
