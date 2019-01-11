package interfaces

// K8sClient is a wrapper around update functions for kubernetes to be exchangeable for
// tests
type K8sClient interface {
	PatchDeployment(namespace, name string, data interface{}) error
	PatchStatefulSet(namespace, name string, data interface{}) error
}
