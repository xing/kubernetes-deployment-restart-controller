package interfaces

// MetaResource is a Kubernetes object that has meta data and is identifiable by some name
type MetaResource interface {
	FullName() string
	Version() string
}

// MetaDeployment unifies "deployment" object types, i.e. Deployment and StatefulSet
type MetaDeployment interface {
	MetaResource
	NeedsRestartOnConfigChange() bool
	ReferencedConfigs() []string
	AppliedChecksums() map[string]string
	UpdateConfigChecksums(k8sClient K8sClient, checksums map[string]string, restart bool) error
}

// MetaConfig unifies "config" object types, i.e. ConfigMap and Secret
type MetaConfig interface {
	MetaResource
	Checksum() string
}
