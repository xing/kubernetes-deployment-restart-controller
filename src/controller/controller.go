package controller

import (
	"errors"
	"fmt"
	"time"

	"source.xing.com/olympus/kubernetes-deployment-restart-controller/src/controller/interfaces"
	"source.xing.com/olympus/kubernetes-deployment-restart-controller/src/util"

	"github.com/golang/glog"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

// DeploymentConfigController updates an annotation on Deployment-like resources once
// related ConfigMap-like objects change. This causes the Deployment to restart its Pods.
type DeploymentConfigController struct {
	Stop chan struct{}

	configAgent interfaces.ConfigAgent
	factory     informers.SharedInformerFactory
}

// NewDeploymentConfigController creates a new instance of DeploymentConfigController
func NewDeploymentConfigController(restartCheckPeriod time.Duration, restartGracePeriod time.Duration) *DeploymentConfigController {
	k8sClient := util.Clientset()
	factory := informers.NewSharedInformerFactory(k8sClient, 5*time.Minute)

	dcc := &DeploymentConfigController{
		configAgent: NewConfigAgent(k8sClient, restartCheckPeriod, restartGracePeriod),
		factory:     factory,
		Stop:        make(chan struct{}),
	}

	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc:    dcc.addResource,
		UpdateFunc: dcc.updateResource,
		DeleteFunc: dcc.deleteResource,
	}
	factory.Core().V1().ConfigMaps().Informer().AddEventHandler(handlers)
	factory.Core().V1().Secrets().Informer().AddEventHandler(handlers)
	factory.Apps().V1().Deployments().Informer().AddEventHandler(handlers)
	factory.Apps().V1().StatefulSets().Informer().AddEventHandler(handlers)

	return dcc
}

// Run starts the controller loop
func (c *DeploymentConfigController) Run() (err error) {
	defer glog.Flush()

	glog.V(1).Info("Starting")

	factoryStopCh := make(chan struct{})
	c.factory.Start(factoryStopCh)

	configAgentErrorCh := make(chan struct{})
	c.configAgent.Start(configAgentErrorCh)

	select {
	case <-configAgentErrorCh:
		err = errors.New("ConfigAgent encountered a fatal error")
	case <-c.Stop:
		glog.V(1).Info("Stopping")
		c.configAgent.Stop()
	}
	factoryStopCh <- struct{}{}
	return
}

func (c *DeploymentConfigController) addResource(obj interface{}) {
	res, err := convertToMetaResource(obj)
	if err == nil {
		c.configAgent.ResourceUpdated(res)
	} else {
		glog.Error(err)
	}
}

func (c *DeploymentConfigController) updateResource(oldObj, newObj interface{}) {
	res, err := convertToMetaResource(newObj)
	if err == nil {
		c.configAgent.ResourceUpdated(res)
	} else {
		glog.Error(err)
	}
}

func (c *DeploymentConfigController) deleteResource(obj interface{}) {
	res, err := convertToMetaResource(obj)
	if err == nil {
		c.configAgent.ResourceDeleted(res)
	} else {
		glog.Error(err)
	}
}

func convertToMetaResource(obj interface{}) (interfaces.MetaResource, error) {
	switch v := obj.(type) {
	case *core.ConfigMap:
		return MetaConfigFromConfigMap(v), nil
	case *core.Secret:
		return MetaConfigFromSecret(v), nil
	case *apps.Deployment:
		return MetaDeploymentFromDeployment(v), nil
	case *apps.StatefulSet:
		return MetaDeploymentFromStatefulSet(v), nil
	case cache.DeletedFinalStateUnknown: // the deletion event was missed by the watch
		return convertToMetaResource(v.Obj)
	}
	return nil, fmt.Errorf("Unhandled type: %T", obj)
}
