// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build enterprise

package history

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

// timeout is the timeout for Kubernetes API calls.
const timeout = 10 * time.Second

var keyRe = regexp.MustCompile(`^[a-zA-Z0-9-]+/[a-zA-Z0-9-]+$`)

// ConfigMapStore is a Store implementation backed by Kubernetes Config Maps.
type ConfigMapStore struct {
	client    kubernetes.Interface
	namespace string
	uid       types.UID
	logger    *slog.Logger
}

// NewConfigMapStore creates a new instance backed by Kubernetes Config Maps.
func NewConfigMapStore(client kubernetes.Interface, namespace string, log *slog.Logger) (*ConfigMapStore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	self, err := client.CoreV1().Pods(namespace).Get(ctx, os.Getenv("HOSTNAME"), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	ownerRefs := self.GetOwnerReferences()
	var statefulset metav1.OwnerReference
	for _, ownerRef := range ownerRefs {
		if ownerRef.Kind == "StatefulSet" && ownerRef.Name == "coordinator" {
			statefulset = ownerRef
			break
		}
	}
	if statefulset.UID == "" {
		return nil, fmt.Errorf("coordinator statefulset not found")
	}
	return &ConfigMapStore{
		client:    client,
		namespace: self.Namespace,
		uid:       statefulset.UID,
		logger:    log,
	}, nil
}

// Get the value for key.
func (s *ConfigMapStore) Get(key string) ([]byte, error) {
	cmName, err := objectName(key)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cm, err := s.client.CoreV1().ConfigMaps(s.namespace).Get(ctx, cmName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return cm.BinaryData[filepath.Base(key)], nil
}

// Has returns true if the key exists.
func (s *ConfigMapStore) Has(key string) (bool, error) {
	cmName, err := objectName(key)
	if err != nil {
		return false, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err = s.client.CoreV1().ConfigMaps(s.namespace).Get(ctx, cmName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return false, nil
	}
	return err == nil, err
}

// Set the value for key.
func (s *ConfigMapStore) Set(key string, value []byte) error {
	cmName, err := objectName(key)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cm, err := s.client.CoreV1().ConfigMaps(s.namespace).Get(ctx, cmName, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if errors.IsNotFound(err) {
		cm = s.newEntry(cmName, key, value)
		_, err := s.client.CoreV1().ConfigMaps(s.namespace).Create(ctx, cm, metav1.CreateOptions{})
		return err
	}
	cm.BinaryData[filepath.Base(key)] = value
	_, err = s.client.CoreV1().ConfigMaps(s.namespace).Update(ctx, cm, metav1.UpdateOptions{})
	return err
}

// CompareAndSwap updates the key to newVal if its current value is oldVal.
func (s *ConfigMapStore) CompareAndSwap(key string, oldVal, newVal []byte) error {
	cmName, err := objectName(key)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cm, err := s.client.CoreV1().ConfigMaps(s.namespace).Get(ctx, cmName, metav1.GetOptions{})
	if err != nil && (!errors.IsNotFound(err) || len(oldVal) != 0) {
		return err
	}
	// Treat non-existing config map as empty to allow initial set.
	if errors.IsNotFound(err) {
		cm = s.newEntry(cmName, key, newVal)
		_, err := s.client.CoreV1().ConfigMaps(s.namespace).Create(ctx, cm, metav1.CreateOptions{})
		return err
	}
	current, ok := cm.BinaryData[filepath.Base(key)]
	if !ok {
		return fmt.Errorf("key %q not found", key)
	}
	if !bytes.Equal(current, oldVal) {
		return fmt.Errorf("object %q has changed since last read", key)
	}
	cm.BinaryData[filepath.Base(key)] = newVal
	_, err = s.client.CoreV1().ConfigMaps(s.namespace).Update(ctx, cm, metav1.UpdateOptions{})
	return err
}

// Watch watches for changes to the value of key.
func (s *ConfigMapStore) Watch(key string) (<-chan []byte, func(), error) {
	cmName, err := objectName(key)
	if err != nil {
		return nil, nil, err
	}
	watcher, err := s.client.CoreV1().ConfigMaps(s.namespace).Watch(context.Background(), metav1.ListOptions{
		LabelSelector: labels.Set(map[string]string{"app.kubernetes.io/managed-by": "contrast.edgeless.systems"}).AsSelector().String(),
		FieldSelector: fields.OneTermEqualSelector("metadata.name", cmName).String(),
	})
	if err != nil {
		return nil, nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan []byte)

	go func() {
		defer watcher.Stop()
		defer close(result)

		s.logger.Debug("watch started", "key", key)
		for {
			select {
			case <-ctx.Done():
				s.logger.Debug("watch canceled", "key", key)
				return
			case event, ok := <-watcher.ResultChan():
				if !ok {
					s.logger.Debug("watch channel closed", "key", key)
					return
				}
				switch event.Type {
				case watch.Error:
					s.logger.Error("watch error", "key", key, "error", event.Object)
					return
				case watch.Added, watch.Modified:
					cm, ok := event.Object.(*corev1.ConfigMap)
					if !ok {
						s.logger.Error("unexpected object type", "key", key, "object", event.Object)
						continue
					}
					result <- cm.BinaryData[filepath.Base(key)]
				}
			}
		}
	}()

	return result, cancel, nil
}

func (s *ConfigMapStore) newEntry(cmName, key string, value []byte) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: s.namespace,
			Labels:    map[string]string{"app.kubernetes.io/managed-by": "contrast.edgeless.systems"},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "StatefulSet",
					Name:       "coordinator",
					UID:        s.uid,
				},
			},
		},
		BinaryData: map[string][]byte{filepath.Base(key): value},
	}
}

func objectName(key string) (string, error) {
	if !keyRe.MatchString(key) {
		return "", fmt.Errorf("invalid key %q", key)
	}
	return fmt.Sprintf("contrast-store-%s", strings.ReplaceAll(key, "/", "-")), nil
}
