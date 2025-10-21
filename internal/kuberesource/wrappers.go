// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kuberesource

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	applyappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applybatchv1 "k8s.io/client-go/applyconfigurations/batch/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	applymetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
	applynodev1 "k8s.io/client-go/applyconfigurations/node/v1"
	applyrbacv1 "k8s.io/client-go/applyconfigurations/rbac/v1"
)

// PodSpecAccessor is an interface for Kubernetes resources that have a PodSpec with corresponding ObjectMeta.
type PodSpecAccessor interface {
	GetObjectMeta() *applymetav1.ObjectMetaApplyConfiguration
	SetObjectMeta(*applymetav1.ObjectMetaApplyConfiguration)
	GetPodSpec() *applycorev1.PodSpecApplyConfiguration
	SetPodSpec(*applycorev1.PodSpecApplyConfiguration)
}

// PodTemplate wraps applycorev1.PodTemplateSpecApplyConfiguration.
type PodTemplate struct {
	*applycorev1.PodTemplateSpecApplyConfiguration
}

// GetObjectMeta returns the ObjectMeta of the Pod template.
func (t *PodTemplate) GetObjectMeta() *applymetav1.ObjectMetaApplyConfiguration {
	if t.PodTemplateSpecApplyConfiguration != nil {
		return t.ObjectMetaApplyConfiguration
	}
	return nil
}

// SetObjectMeta sets the ObjectMeta of the Pod template.
func (t *PodTemplate) SetObjectMeta(meta *applymetav1.ObjectMetaApplyConfiguration) {
	if t.PodTemplateSpecApplyConfiguration == nil {
		t.PodTemplateSpecApplyConfiguration = &applycorev1.PodTemplateSpecApplyConfiguration{}
	}
	t.ObjectMetaApplyConfiguration = meta
}

// GetPodSpec returns the PodSpec of the Pod template.
func (t *PodTemplate) GetPodSpec() *applycorev1.PodSpecApplyConfiguration {
	if t.PodTemplateSpecApplyConfiguration != nil && t.Spec != nil {
		return t.Spec
	}
	return nil
}

// SetPodSpec sets the PodSpec of the Pod template.
func (t *PodTemplate) SetPodSpec(spec *applycorev1.PodSpecApplyConfiguration) {
	if t.PodTemplateSpecApplyConfiguration == nil {
		t.PodTemplateSpecApplyConfiguration = &applycorev1.PodTemplateSpecApplyConfiguration{}
	}
	t.Spec = spec
}

// DeploymentConfig wraps applyappsv1.DeploymentApplyConfiguration.
type DeploymentConfig struct {
	*applyappsv1.DeploymentApplyConfiguration
}

// Deployment creates a new DeploymentConfig.
func Deployment(name, namespace string) *DeploymentConfig {
	d := applyappsv1.Deployment(name, namespace)
	if namespace == "" && d.ObjectMetaApplyConfiguration != nil {
		d.Namespace = nil
	}
	return &DeploymentConfig{d}
}

// PodSpecAccessor returns the PodTemplate of the Deployment.
func (c *DeploymentConfig) PodSpecAccessor() PodSpecAccessor {
	if c.Spec != nil && c.Spec.Template != nil {
		return &PodTemplate{c.Spec.Template}
	}
	return nil
}

// DeploymentSpecConfig wraps applyappsv1.DeploymentSpecApplyConfiguration.
type DeploymentSpecConfig struct {
	*applyappsv1.DeploymentSpecApplyConfiguration
}

// DeploymentSpec creates a new DeploymentSpecConfig.
func DeploymentSpec() *DeploymentSpecConfig {
	return &DeploymentSpecConfig{applyappsv1.DeploymentSpec()}
}

// DaemonSetConfig wraps applyappsv1.DaemonSetApplyConfiguration.
type DaemonSetConfig struct {
	*applyappsv1.DaemonSetApplyConfiguration
}

// DaemonSet creates a new DaemonSetConfig.
func DaemonSet(name, namespace string) *DaemonSetConfig {
	d := applyappsv1.DaemonSet(name, namespace)
	if namespace == "" && d.ObjectMetaApplyConfiguration != nil {
		d.Namespace = nil
	}
	return &DaemonSetConfig{d}
}

// PodSpecAccessor returns the PodTemplate of the DaemonSet.
func (c *DaemonSetConfig) PodSpecAccessor() PodSpecAccessor {
	if c.Spec != nil && c.Spec.Template != nil {
		return &PodTemplate{c.Spec.Template}
	}
	return nil
}

// DaemonSetSpecConfig wraps applyappsv1.DaemonSetSpecApplyConfiguration.
type DaemonSetSpecConfig struct {
	*applyappsv1.DaemonSetSpecApplyConfiguration
}

// DaemonSetSpec creates a new DaemonSetSpecConfig.
func DaemonSetSpec() *DaemonSetSpecConfig {
	return &DaemonSetSpecConfig{applyappsv1.DaemonSetSpec()}
}

// StatefulSetConfig wraps applyappsv1.StatefulSetApplyConfiguration.
type StatefulSetConfig struct {
	*applyappsv1.StatefulSetApplyConfiguration
}

// StatefulSet creates a new StatefulSetConfig.
func StatefulSet(name, namespace string) *StatefulSetConfig {
	s := applyappsv1.StatefulSet(name, namespace)
	if namespace == "" && s.ObjectMetaApplyConfiguration != nil {
		s.Namespace = nil
	}
	return &StatefulSetConfig{s}
}

// PodSpecAccessor returns the PodTemplate of the StatefulSet.
func (c *StatefulSetConfig) PodSpecAccessor() PodSpecAccessor {
	if c.Spec != nil && c.Spec.Template != nil {
		return &PodTemplate{c.Spec.Template}
	}
	return nil
}

// StatefulSetSpecConfig wraps applyappsv1.StatefulSetSpecApplyConfiguration.
type StatefulSetSpecConfig struct {
	*applyappsv1.StatefulSetSpecApplyConfiguration
}

// StatefulSetSpec creates a new StatefulSetSpecConfig.
func StatefulSetSpec() *StatefulSetSpecConfig {
	return &StatefulSetSpecConfig{applyappsv1.StatefulSetSpec()}
}

// PodConfig wraps applyappsv1.PodApplyConfiguration.
type PodConfig struct {
	*applycorev1.PodApplyConfiguration
}

// Pod creates a new PodConfig.
func Pod(name, namespace string) *PodConfig {
	p := applycorev1.Pod(name, namespace)
	if namespace == "" && p.ObjectMetaApplyConfiguration != nil {
		p.Namespace = nil
	}
	return &PodConfig{p}
}

// GetObjectMeta returns the ObjectMeta of the Pod.
func (c *PodConfig) GetObjectMeta() *applymetav1.ObjectMetaApplyConfiguration {
	if c.PodApplyConfiguration != nil {
		return c.ObjectMetaApplyConfiguration
	}
	return nil
}

// SetObjectMeta sets the ObjectMeta of the Pod.
func (c *PodConfig) SetObjectMeta(meta *applymetav1.ObjectMetaApplyConfiguration) {
	if c.PodApplyConfiguration == nil {
		c.PodApplyConfiguration = &applycorev1.PodApplyConfiguration{}
	}
	c.ObjectMetaApplyConfiguration = meta
}

// GetPodSpec returns the PodSpec of the Pod.
func (c *PodConfig) GetPodSpec() *applycorev1.PodSpecApplyConfiguration {
	if c.PodApplyConfiguration != nil && c.Spec != nil {
		return c.Spec
	}
	return nil
}

// SetPodSpec sets the PodSpec of the Pod.
func (c *PodConfig) SetPodSpec(spec *applycorev1.PodSpecApplyConfiguration) {
	if c.PodApplyConfiguration == nil {
		c.PodApplyConfiguration = &applycorev1.PodApplyConfiguration{}
	}
	c.Spec = spec
}

// CronJobConfig wraps applybatchv1.CronJobApplyConfiguration.
type CronJobConfig struct {
	*applybatchv1.CronJobApplyConfiguration
}

// PodSpecAccessor returns the PodTemplate of the CronJob.
func (c *CronJobConfig) PodSpecAccessor() PodSpecAccessor {
	if c.Spec != nil &&
		c.Spec.JobTemplate != nil &&
		c.Spec.JobTemplate.Spec != nil &&
		c.Spec.JobTemplate.Spec.Template != nil {
		return &PodTemplate{c.Spec.JobTemplate.Spec.Template}
	}
	return nil
}

// JobConfig wraps applybatchv1.CronJobApplyConfiguration.
type JobConfig struct {
	*applybatchv1.JobApplyConfiguration
}

// PodSpecAccessor returns the PodTemplate of the Job.
func (c *JobConfig) PodSpecAccessor() PodSpecAccessor {
	if c.Spec != nil && c.Spec.Template != nil {
		return &PodTemplate{c.Spec.Template}
	}
	return nil
}

// ReplicaSetConfig wraps applyappsv1.ReplicaSetApplyConfiguration.
type ReplicaSetConfig struct {
	*applyappsv1.ReplicaSetApplyConfiguration
}

// PodSpecAccessor returns the PodTemplate of the ReplicaSet.
func (c *ReplicaSetConfig) PodSpecAccessor() PodSpecAccessor {
	if c.Spec != nil && c.Spec.Template != nil {
		return &PodTemplate{c.Spec.Template}
	}
	return nil
}

// ReplicationControllerConfig wraps applycorev1.ReplicationControllerApplyConfiguration.
type ReplicationControllerConfig struct {
	*applycorev1.ReplicationControllerApplyConfiguration
}

// PodSpecAccessor returns the PodTemplate of the ReplicationController.
func (c *ReplicationControllerConfig) PodSpecAccessor() PodSpecAccessor {
	if c.Spec != nil && c.Spec.Template != nil {
		return &PodTemplate{c.Spec.Template}
	}
	return nil
}

// LabelSelectorConfig wraps applymetav1.LabelSelectorApplyConfiguration.
type LabelSelectorConfig struct {
	*applymetav1.LabelSelectorApplyConfiguration
}

// LabelSelector creates a new LabelSelectorConfig.
func LabelSelector() *LabelSelectorConfig {
	return &LabelSelectorConfig{applymetav1.LabelSelector()}
}

// PodTemplateSpecConfig wraps applycorev1.PodTemplateSpecApplyConfiguration.
type PodTemplateSpecConfig struct {
	*applycorev1.PodTemplateSpecApplyConfiguration
}

// PodTemplateSpec creates a new PodTemplateSpecConfig.
func PodTemplateSpec() *PodTemplateSpecConfig {
	return &PodTemplateSpecConfig{applycorev1.PodTemplateSpec()}
}

// PodSpecConfig wraps applycorev1.PodSpecApplyConfiguration.
type PodSpecConfig struct {
	*applycorev1.PodSpecApplyConfiguration
}

// PodSpec creates a new PodSpecConfig.
func PodSpec() *PodSpecConfig {
	return &PodSpecConfig{applycorev1.PodSpec()}
}

// ContainerConfig wraps applycorev1.ContainerApplyConfiguration.
type ContainerConfig struct {
	*applycorev1.ContainerApplyConfiguration
}

// Container creates a new ContainerConfig.
func Container() *ContainerConfig {
	return &ContainerConfig{applycorev1.Container()}
}

// EnvVarConfig wraps applycorev1.EnvVarApplyConfiguration.
type EnvVarConfig struct {
	*applycorev1.EnvVarApplyConfiguration
}

// EnvVar creates a new EnvVarConfig.
func EnvVar() *EnvVarConfig {
	return &EnvVarConfig{applycorev1.EnvVar()}
}

// NewEnvVar creates a new EnvVarApplyConfiguration from name and value.
func NewEnvVar(name, value string) *applycorev1.EnvVarApplyConfiguration {
	return applycorev1.EnvVar().WithName(name).WithValue(value)
}

// VolumeMountConfig wraps applycorev1.VolumeMountApplyConfiguration.
type VolumeMountConfig struct {
	*applycorev1.VolumeMountApplyConfiguration
}

// VolumeMount creates a new VolumeMountConfig.
func VolumeMount() *VolumeMountConfig {
	return &VolumeMountConfig{applycorev1.VolumeMount()}
}

// ResourceRequirementsConfig wraps applycorev1.ResourceRequirementsApplyConfiguration.
type ResourceRequirementsConfig struct {
	*applycorev1.ResourceRequirementsApplyConfiguration
}

// ResourceRequirements creates a new ResourceRequirementsConfig.
func ResourceRequirements() *ResourceRequirementsConfig {
	return &ResourceRequirementsConfig{applycorev1.ResourceRequirements()}
}

// WithMemoryRequest sets the memory request of the ResourceRequirements.
func (r *ResourceRequirementsConfig) WithMemoryRequest(memoryMi int64) *applycorev1.ResourceRequirementsApplyConfiguration {
	return r.
		WithRequests(corev1.ResourceList{
			corev1.ResourceMemory: fromPtr(resource.NewQuantity(memoryMi*1024*1024, resource.BinarySI)),
		})
}

// WithMemoryLimitAndRequest sets the memory limit and request of the ResourceRequirements.
func (r *ResourceRequirementsConfig) WithMemoryLimitAndRequest(memoryMi int64) *applycorev1.ResourceRequirementsApplyConfiguration {
	return r.
		WithRequests(corev1.ResourceList{
			corev1.ResourceMemory: fromPtr(resource.NewQuantity(memoryMi*1024*1024, resource.BinarySI)),
		}).
		WithLimits(corev1.ResourceList{
			corev1.ResourceMemory: fromPtr(resource.NewQuantity(memoryMi*1024*1024, resource.BinarySI)),
		})
}

// WithCPURequest sets the CPU request of the ResourceRequirements.
func (r *ResourceRequirementsConfig) WithCPURequest(cpuM int64) *applycorev1.ResourceRequirementsApplyConfiguration {
	return r.WithRequests(corev1.ResourceList{
		corev1.ResourceCPU: fromPtr(resource.NewMilliQuantity(cpuM, resource.DecimalSI)),
		// Don't set CPU limits, see https://home.robusta.dev/blog/stop-using-cpu-limits
	})
}

// VolumeConfig wraps applycorev1.VolumeApplyConfiguration.
type VolumeConfig struct {
	*applycorev1.VolumeApplyConfiguration
}

// Volume creates a new VolumeConfig.
func Volume() *VolumeConfig {
	return &VolumeConfig{applycorev1.Volume()}
}

// EmptyDirVolumeSourceConfig wraps applycorev1.EmptyDirVolumeSourceApplyConfiguration.
type EmptyDirVolumeSourceConfig struct {
	*applycorev1.EmptyDirVolumeSourceApplyConfiguration
}

// EmptyDirVolumeSource creates a new EmptyDirVolumeSourceConfig.
func EmptyDirVolumeSource() *EmptyDirVolumeSourceConfig {
	return &EmptyDirVolumeSourceConfig{applycorev1.EmptyDirVolumeSource()}
}

// Inner returns the inner applycorev1.EmptyDirVolumeSourceApplyConfiguration.
func (e *EmptyDirVolumeSourceConfig) Inner() *applycorev1.EmptyDirVolumeSourceApplyConfiguration {
	return e.EmptyDirVolumeSourceApplyConfiguration
}

// HostPathVolumeSourceConfig wraps applycorev1.HostPathVolumeSourceApplyConfiguration.
type HostPathVolumeSourceConfig struct {
	*applycorev1.HostPathVolumeSourceApplyConfiguration
}

// HostPathVolumeSource creates a new HostPathVolumeSourceConfig.
func HostPathVolumeSource() *HostPathVolumeSourceConfig {
	return &HostPathVolumeSourceConfig{applycorev1.HostPathVolumeSource()}
}

// Inner returns the inner applycorev1.HostPathVolumeSourceApplyConfiguration.
func (h *HostPathVolumeSourceConfig) Inner() *applycorev1.HostPathVolumeSourceApplyConfiguration {
	return h.HostPathVolumeSourceApplyConfiguration
}

// ConfigMapVolumeSourceConfig wraps applycorev1.ConfigMapVolumeSourceApplyConfiguration.
type ConfigMapVolumeSourceConfig struct {
	*applycorev1.ConfigMapVolumeSourceApplyConfiguration
}

// ConfigMapVolumeSource creates a new ConfigMapVolumeSourceConfig.
func ConfigMapVolumeSource() *ConfigMapVolumeSourceConfig {
	return &ConfigMapVolumeSourceConfig{applycorev1.ConfigMapVolumeSource()}
}

// Inner returns the inner applycorev1.ConfigMapVolumeSourceApplyConfiguration.
func (c *ConfigMapVolumeSourceConfig) Inner() *applycorev1.ConfigMapVolumeSourceApplyConfiguration {
	return c.ConfigMapVolumeSourceApplyConfiguration
}

// ContainerPortConfig wraps applycorev1.ContainerPortApplyConfiguration.
type ContainerPortConfig struct {
	*applycorev1.ContainerPortApplyConfiguration
}

// ContainerPort creates a new ContainerPortConfig.
func ContainerPort() *ContainerPortConfig {
	return &ContainerPortConfig{applycorev1.ContainerPort()}
}

// SecurityContextConfig wraps applycorev1.SecurityContextApplyConfiguration.
type SecurityContextConfig struct {
	*applycorev1.SecurityContextApplyConfiguration
}

// SecurityContext creates a new SecurityContextConfig.
func SecurityContext() *SecurityContextConfig {
	return &SecurityContextConfig{applycorev1.SecurityContext()}
}

// WithPrivileged sets the Privileged field in the declarative configuration to the given value.
func (s *SecurityContextConfig) WithPrivileged(privileged bool) *SecurityContextConfig {
	s.Privileged = &privileged
	return s
}

// AddCapabilities appends the given capabilities to the add list.
func (s *SecurityContextConfig) AddCapabilities(capabilities ...corev1.Capability) *SecurityContextConfig {
	if s.Capabilities == nil {
		s.Capabilities = &applycorev1.CapabilitiesApplyConfiguration{}
	}
	s.Capabilities.Add = append(s.Capabilities.Add, capabilities...)
	return s
}

// ServiceConfig wraps applycorev1.ServiceApplyConfiguration.
type ServiceConfig struct {
	*applycorev1.ServiceApplyConfiguration
}

// Service creates a new ServiceConfig.
func Service(name, namespace string) *ServiceConfig {
	s := applycorev1.Service(name, namespace)
	if namespace == "" && s.ObjectMetaApplyConfiguration != nil {
		s.Namespace = nil
	}
	return &ServiceConfig{s}
}

// ServiceSpecConfig wraps applycorev1.ServiceSpecApplyConfiguration.
type ServiceSpecConfig struct {
	*applycorev1.ServiceSpecApplyConfiguration
}

// ServiceSpec creates a new ServiceSpecConfig.
func ServiceSpec() *ServiceSpecConfig {
	return &ServiceSpecConfig{applycorev1.ServiceSpec()}
}

// ServicePortConfig wraps applycorev1.ServicePortApplyConfiguration.
type ServicePortConfig struct {
	*applycorev1.ServicePortApplyConfiguration
}

// ServicePort creates a new ServicePortConfig.
func ServicePort() *ServicePortConfig {
	return &ServicePortConfig{applycorev1.ServicePort()}
}

// ServiceAccountConfig wraps applycorev1.ServiceAccountApplyConfiguration.
type ServiceAccountConfig struct {
	*applycorev1.ServiceAccountApplyConfiguration
}

// ServiceAccount creates a new ServiceAccountConfig.
func ServiceAccount(name, namespace string) *ServiceAccountConfig {
	s := &ServiceAccountConfig{applycorev1.ServiceAccount(name, namespace)}
	if namespace == "" && s.ObjectMetaApplyConfiguration != nil {
		s.Namespace = nil
	}
	return s
}

// RoleConfig wraps applyrbacv1.RoleApplyConfiguration.
type RoleConfig struct {
	*applyrbacv1.RoleApplyConfiguration
}

// Role creates a new RoleConfig.
func Role(name, namespace string) *RoleConfig {
	r := &RoleConfig{applyrbacv1.Role(name, namespace)}
	if namespace == "" && r.ObjectMetaApplyConfiguration != nil {
		r.Namespace = nil
	}
	return r
}

// RoleBindingConfig wraps applyrbacv1.RoleBindingApplyConfiguration.
type RoleBindingConfig struct {
	*applyrbacv1.RoleBindingApplyConfiguration
}

// RoleBinding creates a new RoleBindingConfig.
func RoleBinding(name, namespace string) *RoleBindingConfig {
	r := &RoleBindingConfig{applyrbacv1.RoleBinding(name, namespace)}
	if namespace == "" && r.ObjectMetaApplyConfiguration != nil {
		r.Namespace = nil
	}
	return r
}

// NamespaceConfig wraps applycorev1.NamespaceApplyConfiguration.
type NamespaceConfig struct {
	*applycorev1.NamespaceApplyConfiguration
}

// Namespace creates a new NamespaceConfig.
func Namespace(name string) *applycorev1.NamespaceApplyConfiguration {
	return applycorev1.Namespace(name)
}

// Probe creates a new ProbeApplyConfiguration.
func Probe() *applycorev1.ProbeApplyConfiguration {
	return applycorev1.Probe()
}

// TCPSocketAction creates a new TCPSocketActionApplyConfiguration.
func TCPSocketAction() *applycorev1.TCPSocketActionApplyConfiguration {
	return applycorev1.TCPSocketAction()
}

// ExecAction creates a new ExecActionApplyConfiguration.
func ExecAction() *applycorev1.ExecActionApplyConfiguration {
	return applycorev1.ExecAction()
}

// RuntimeClassConfig wraps applypodsv1.RuntimeClassApplyConfiguration for a runtime class.
type RuntimeClassConfig struct {
	*applynodev1.RuntimeClassApplyConfiguration
}

// RuntimeClass constructs a new RuntimeClassConfig.
func RuntimeClass(name string) *RuntimeClassConfig {
	return &RuntimeClassConfig{applynodev1.RuntimeClass(name)}
}

// Overhead creates a new OverheadApplyConfiguration.
func Overhead(podFixed corev1.ResourceList) *applynodev1.OverheadApplyConfiguration {
	return applynodev1.Overhead().WithPodFixed(podFixed)
}

// Scheduling creates a new SchedulingApplyConfiguration.
func Scheduling(nodeSelector map[string]string, tolerations ...*applycorev1.TolerationApplyConfiguration) *applynodev1.SchedulingApplyConfiguration {
	return applynodev1.Scheduling().
		WithNodeSelector(nodeSelector).
		WithTolerations(tolerations...)
}

// PersistentVolumeClaimConfig wraps applycorev1.PersistentVolumeClaimApplyConfiguration.
type PersistentVolumeClaimConfig struct {
	*applycorev1.PersistentVolumeClaimApplyConfiguration
}

// PersistentVolumeClaim constructs a new PersistentVolumeClaimConfig.
func PersistentVolumeClaim(name, namespace string) *PersistentVolumeClaimConfig {
	pvc := applycorev1.PersistentVolumeClaim(name, namespace)
	if namespace == "" && pvc.ObjectMetaApplyConfiguration != nil {
		pvc.Namespace = nil
	}
	return &PersistentVolumeClaimConfig{pvc}
}

// ConfigMapConfig wraps applycorev1.ConfigMapApplyConfiguration.
type ConfigMapConfig struct {
	*applycorev1.ConfigMapApplyConfiguration
}

// ConfigMap creates a new ConfigMapConfig.
func ConfigMap(name, namespace string) *ConfigMapConfig {
	s := applycorev1.ConfigMap(name, namespace)
	if namespace == "" && s.ObjectMetaApplyConfiguration != nil {
		s.Namespace = nil
	}
	return &ConfigMapConfig{s}
}

func fromPtr[T any](v *T) T {
	if v != nil {
		return *v
	}
	var zero T
	return zero
}
