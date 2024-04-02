package kuberesource

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	applyappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	applymetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

// DeploymentConfig wraps applyappsv1.DeploymentApplyConfiguration.
type DeploymentConfig struct {
	*applyappsv1.DeploymentApplyConfiguration
}

// Deployment creates a new DeploymentConfig.
func Deployment(name, namespace string) *DeploymentConfig {
	return &DeploymentConfig{applyappsv1.Deployment(name, namespace)}
}

// DeploymentSpecConfig wraps applyappsv1.DeploymentSpecApplyConfiguration.
type DeploymentSpecConfig struct {
	*applyappsv1.DeploymentSpecApplyConfiguration
}

// DeploymentSpec creates a new DeploymentSpecConfig.
func DeploymentSpec() *DeploymentSpecConfig {
	return &DeploymentSpecConfig{applyappsv1.DeploymentSpec()}
}

// PodConfig wraps applyappsv1.PodApplyConfiguration.
type PodConfig struct {
	*applycorev1.PodApplyConfiguration
}

// Pod creates a new PodConfig.
func Pod(name, namespace string) *PodConfig {
	return &PodConfig{applycorev1.Pod(name, namespace)}
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

// ContainerPortConfig wraps applycorev1.ContainerPortApplyConfiguration.
type ContainerPortConfig struct {
	*applycorev1.ContainerPortApplyConfiguration
}

// ContainerPort creates a new ContainerPortConfig.
func ContainerPort() *ContainerPortConfig {
	return &ContainerPortConfig{applycorev1.ContainerPort()}
}

// ServiceConfig wraps applycorev1.ServiceApplyConfiguration.
type ServiceConfig struct {
	*applycorev1.ServiceApplyConfiguration
}

// Service creates a new ServiceConfig.
func Service(name, namespace string) *ServiceConfig {
	return &ServiceConfig{applycorev1.Service(name, namespace)}
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
	return &ServiceAccountConfig{applycorev1.ServiceAccount(name, namespace)}
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

func fromPtr[T any](v *T) T {
	if v != nil {
		return *v
	}
	var zero T
	return zero
}
