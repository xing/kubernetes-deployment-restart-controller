package controller

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/xing/kubernetes-deployment-restart-controller/src/controller/test"
	"k8s.io/client-go/tools/cache"
)

func init() {
	if os.Getenv("V") == "" {
		flag.Set("stderrthreshold", "5")
	}
}

func TestStopControllerLoop(t *testing.T) {
	c := controller()

	go func() {
		c.Stop <- struct{}{}
	}()

	c.Run()
}

func TestSendingConfigMaps(t *testing.T) {
	c := controller()
	r := newConfigMap("test", "one", "", nil)

	go func() {
		c.updateResource(r, r)
		c.Stop <- struct{}{}
	}()

	c.Run()

	equals(t, getDummyAgent(c).UpdatedResources[0].FullName(), "configmap/test/one")
}

func TestSendingSecrets(t *testing.T) {
	c := controller()
	r := newSecret("test", "one", "", nil)

	go func() {
		c.addResource(r)
		c.Stop <- struct{}{}
	}()

	c.Run()

	equals(t, getDummyAgent(c).UpdatedResources[0].FullName(), "secret/test/one")
}

func TestSendingDeployments(t *testing.T) {
	c := controller()
	r := newDeploymentFromYAML(`
---
metadata:
  name: one
  namespace: test
`)

	go func() {
		c.addResource(r)
		c.Stop <- struct{}{}
	}()

	c.Run()

	equals(t, getDummyAgent(c).UpdatedResources[0].FullName(), "deployment/test/one")
}

func TestSendingStatefulSets(t *testing.T) {
	c := controller()
	r := newStatefulSetFromYAML(`
---
metadata:
  name: one
  namespace: test
`)

	go func() {
		c.addResource(r)
		c.Stop <- struct{}{}
	}()

	c.Run()

	equals(t, getDummyAgent(c).UpdatedResources[0].FullName(), "statefulset/test/one")
}

func TestSendingMissedDeletionEvents(t *testing.T) {
	c := controller()
	r := newConfigMap("test", "one", "", nil)

	go func() {
		c.deleteResource(cache.DeletedFinalStateUnknown{Obj: r})
		c.Stop <- struct{}{}
	}()

	c.Run()

	equals(t, getDummyAgent(c).DeletedResources[0].FullName(), "configmap/test/one")
}

func TestAddingUnknownResourcesDoesNothing(t *testing.T) {
	c := controller()
	r := struct{}{}

	go func() {
		c.addResource(r)
		c.Stop <- struct{}{}
	}()

	c.Run()

	equals(t, len(getDummyAgent(c).UpdatedResources), 0)
}

func TestUpdatingUnknownResourceDoesNothing(t *testing.T) {
	c := controller()
	r := struct{}{}

	go func() {
		c.updateResource(r, r)
		c.Stop <- struct{}{}
	}()

	c.Run()

	equals(t, len(getDummyAgent(c).UpdatedResources), 0)
}

func TestDeletingUnknownResourceDoesNothing(t *testing.T) {
	c := controller()
	r := struct{}{}

	go func() {
		c.deleteResource(r)
		c.Stop <- struct{}{}
	}()

	c.Run()

	equals(t, len(getDummyAgent(c).DeletedResources), 0)
}

func controller() *DeploymentConfigController {
	controller := NewDeploymentConfigController(100*time.Millisecond, 1*time.Second, []string{})
	controller.configAgent = test.NewDummyConfigAgent()
	return controller
}

func getDummyAgent(c *DeploymentConfigController) *test.DummyConfigAgent {
	return c.configAgent.(*test.DummyConfigAgent)
}
