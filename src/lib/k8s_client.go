package lib

import (
	"encoding/json"
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/xing/kubernetes-deployment-restart-controller/src/controller/interfaces"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
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
	_, err = c.Interface.AppsV1().Deployments(namespace).Patch(context.TODO(), name, types.MergePatchType, encodedData, metav1.PatchOptions{})
	return
}

func (c *k8sClient) PatchStatefulSet(namespace, name string, patchData interface{}) (err error) {
	encodedData, err := json.Marshal(patchData)
	if err != nil {
		return
	}
	_, err = c.Interface.AppsV1().StatefulSets(namespace).Patch(context.TODO(), name, types.MergePatchType, encodedData, metav1.PatchOptions{})
	return
}
