package lib

import (
	"encoding/json"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"source.xing.com/olympus/kubernetes-deployment-restart-controller/src/controller/interfaces"
)

type k8sClient struct {
	Interface kubernetes.Interface
}

// NewK8sClient returns a implementation of Client with kubernetes
func NewK8sClient(intrfc kubernetes.Interface) interfaces.K8sClient {
	return &k8sClient{Interface: intrfc}
}

func (c *k8sClient) PatchDeployment(namespace, name string, patchData interface{}) (err error) {
	encodedData, err := json.Marshal(patchData)
	if err != nil {
		return
	}
	_, err = c.Interface.AppsV1().Deployments(namespace).Patch(name, types.MergePatchType, encodedData)
	return
}

func (c *k8sClient) PatchStatefulSet(namespace, name string, patchData interface{}) (err error) {
	encodedData, err := json.Marshal(patchData)
	if err != nil {
		return
	}
	_, err = c.Interface.AppsV1beta1().StatefulSets(namespace).Patch(name, types.MergePatchType, encodedData)
	return
}
