package test

import (
	"fmt"
)

type ResourcePatch struct {
	Path string
	Data interface{}
}

type DummyK8sClient struct {
	Patches []*ResourcePatch
	Error   error
}

// NewDummyK8sClient returns a dummy implementation
func NewDummyK8sClient() *DummyK8sClient {
	return &DummyK8sClient{Patches: []*ResourcePatch{}}
}

func (c *DummyK8sClient) PatchDeployment(namespace, name string, data interface{}) (err error) {
	c.Patches = append(c.Patches, &ResourcePatch{
		Path: fmt.Sprintf("deployment/%s/%s", namespace, name),
		Data: data,
	})
	return c.Error
}

func (c *DummyK8sClient) PatchStatefulSet(namespace, name string, data interface{}) (err error) {
	c.Patches = append(c.Patches, &ResourcePatch{
		Path: fmt.Sprintf("statefulset/%s/%s", namespace, name),
		Data: data,
	})
	return c.Error
}
