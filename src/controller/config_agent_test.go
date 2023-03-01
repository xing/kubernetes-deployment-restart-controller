package controller

import (
	"errors"
	"testing"
	"time"

	"github.com/xing/kubernetes-deployment-restart-controller/src/controller/test"
	"github.com/xing/kubernetes-deployment-restart-controller/src/util"
)

func TestStopLoop(t *testing.T) {
	a := agent()

	a.Start(nil)
	a.Stop()
}

func TestControllerStopChReceivesStopOnError(t *testing.T) {
	a := agent()

	controllerStopCh := make(chan struct{})
	a.Start(controllerStopCh)
	go a.stopWithError(errors.New("some error"))
	<-controllerStopCh
}

func TestResourceUpdatedTracksNewConfigs(t *testing.T) {
	a := agent()
	c := configA()

	a.Start(nil)
	a.ResourceUpdated(c)
	a.Stop()

	config := a.configs[c.FullName()]
	equals(t, config != nil, true)
	equals(t, config.Checksum(), c.Checksum())
}

func TestResourceUpdatedUpdatesAlreadyTrackedConfigs(t *testing.T) {
	a := agent()
	c1 := configA()
	c2 := configAUpdated()

	a.Start(nil)
	a.ResourceUpdated(c1)
	a.ResourceUpdated(c2)
	a.Stop()

	equals(t, len(a.configs), 1)
	config := a.configs[c1.FullName()]
	equals(t, config != nil, true)
	equals(t, config.Checksum(), c2.Checksum())
}

func TestResourceUpdatedDoesNotUpdateConfigWhenVersionIsNotChanged(t *testing.T) {
	a := agent()
	c1 := configA()
	c2 := configAUpdated()
	c2.VersionValue = c1.VersionValue

	a.Start(nil)
	a.ResourceUpdated(c1)
	a.ResourceUpdated(c2)
	a.Stop()

	config := a.configs[c1.FullName()]
	equals(t, config != nil, true)
	equals(t, config.Checksum(), c1.Checksum())
}

func TestResourceUpdatedCreatesAChangeWhenNewConfigIsTracked(t *testing.T) {
	a := agent()
	c := configA()

	a.Start(nil)
	a.ResourceUpdated(c)
	a.Stop()

	change := a.changes[c.FullName()]
	equals(t, change != nil, true)
	equals(t, change.Observations, 1)
}

func TestResourceUpdatedDoesNotCreateAChangeWhenSameConfigVersionIsObserved(t *testing.T) {
	a := agent()
	c := configA()

	a.Start(nil)
	a.ResourceUpdated(c)
	a.ResourceUpdated(c)
	a.Stop()

	equals(t, len(a.changes), 1)
	change := a.changes[c.FullName()]
	equals(t, change != nil, true)
	equals(t, change.Observations, 1)
}

func TestResourceUpdatedDoesNotCreateAChangeWhenConfigWithSameChecksumIsObserved(t *testing.T) {
	a := agent()
	c1 := configA()
	c2 := configAUpdated()
	c2.ChecksumValue = c1.ChecksumValue

	a.Start(nil)
	a.ResourceUpdated(c1)
	a.ResourceUpdated(c2)
	a.Stop()

	equals(t, len(a.changes), 1)
	equals(t, a.changes[c1.FullName()].Observations, 1)

}

func TestResourceUpdatedDoesNotTrackIrrelevantDeployments(t *testing.T) {
	a := agent()
	d := deploymentA()
	d.NeedsRestartOnConfigChangeValue = false

	a.Start(nil)
	a.ResourceUpdated(d)
	a.Stop()

	equals(t, len(a.deployments), 0)
}

func TestResourceUpdatedCreatesPendingConfigsForThoseReferencedInDeployment(t *testing.T) {
	a := agent()
	d := deploymentA()

	a.Start(nil)
	a.ResourceUpdated(d)
	a.Stop()

	equals(t, len(a.configs), 2)
	deployment := a.deployments[d.FullName()]

	for _, v := range d.ReferencedConfigs() {
		config := a.configs[v]
		equals(t, config != nil, true)
		equals(t, config.Pending(), true)
		equals(t, config.Deployments[d.FullName()], deployment)

		equals(t, deployment.Configs[v], config)
	}
}

func TestResourceUpdatedUnlinksOrphanedDeploymentConfigs(t *testing.T) {
	a := agent()
	d1 := deploymentA()
	d2 := deploymentB()
	d3 := deploymentAUpdated()
	d3.ReferencedConfigsValue = d1.ReferencedConfigsValue[:1]

	a.Start(nil)
	a.ResourceUpdated(d1)
	a.ResourceUpdated(d2)
	a.ResourceUpdated(d3)
	a.Stop()

	deployment := a.deployments[d1.FullName()]
	equals(t, len(a.configs), 2)
	equals(t, len(deployment.Configs), 1)
}

func TestResourceUpdatedCleaunsUpOrphanedAndUnusedConfigs(t *testing.T) {
	a := agent()
	d1 := deploymentA()
	d2 := deploymentAUpdated()
	d2.ReferencedConfigsValue = d1.ReferencedConfigsValue[:1]

	a.Start(nil)
	a.ResourceUpdated(d1)
	a.ResourceUpdated(d2)
	a.Stop()

	equals(t, len(a.configs), 1)
}

func TestResourceUpdatedAddingDeploymentIntroducesAChange(t *testing.T) {
	a := agent()
	d := deploymentA()

	a.Start(nil)
	a.ResourceUpdated(d)
	a.Stop()

	equals(t, len(a.changes), 1)
}

func TestResourceUpdatedUpdatingADeploymentReferencingSameConfigsDoesNotIntroduceAChange(t *testing.T) {
	a := agent()
	d := deploymentA()

	a.Start(nil)
	a.ResourceUpdated(d)
	d.VersionValue = deploymentAUpdated().Version()
	a.ResourceUpdated(d)
	a.Stop()

	equals(t, len(a.changes), 1)
	equals(t, a.changes[d.FullName()].Observations, 1)
}

func TestResourceDeletedCleansUpConfigs(t *testing.T) {
	a := agent()
	c := configA()

	a.Start(nil)
	a.ResourceUpdated(c)
	a.ResourceDeleted(c)
	a.Stop()

	equals(t, len(a.configs), 0)
	equals(t, len(a.changes), 0)
}

func TestResourceDeletedCleansUpDeploymentsAndUnusedConfigs(t *testing.T) {
	a := agent()
	d := deploymentA()

	a.Start(nil)
	a.ResourceUpdated(d)
	a.ResourceDeleted(d)
	a.Stop()

	equals(t, len(a.deployments), 0)
	equals(t, len(a.configs), 0)
	equals(t, len(a.changes), 0)
}

func TestResourceDeletedCleansUpDeploymentConfigs(t *testing.T) {
	a := agent()
	c := configA()
	d := deploymentA()

	a.Start(nil)
	a.ResourceUpdated(c)
	a.ResourceUpdated(d)
	a.ResourceDeleted(c)
	a.Stop()

	equals(t, len(a.configs), 1)
	equals(t, len(a.changes), 1)
	deployment := a.deployments[d.FullName()]
	equals(t, deployment != nil, true)
	equals(t, len(deployment.Configs), 1)
}

func TestConfigChangesGetProcessedInGracePeriod(t *testing.T) {
	a := agent()
	c := configA()
	d := deploymentA()

	a.Start(nil)
	a.ResourceUpdated(c)
	a.ResourceUpdated(d)
	time.Sleep(150 * time.Millisecond)
	a.Stop()

	equals(t, len(a.changes), 0)
	equals(t, d.UpdatedChecksums, map[string]string(nil))
}

func TestConfigChangesOrphanedResourceChangesGetCleanedUp(t *testing.T) {
	a := agent()

	a.changes["non-existent"] = NewChange()
	a.Start(nil)
	time.Sleep(150 * time.Millisecond)
	a.Stop()

	equals(t, len(a.changes), 0)
}

func TestConfigChangesNewDeploymentsGetChecksumUpdatesWithoutRestarts(t *testing.T) {
	a := agent()
	c := configAUpdated()
	d := deploymentA()
	delete(d.AppliedChecksumsValue, c.FullName())

	a.Start(nil)
	a.ResourceUpdated(c)
	a.ResourceUpdated(d)
	time.Sleep(150 * time.Millisecond)
	a.Stop()

	expectedChecksums := map[string]string{
		c.FullName():         c.Checksum(),
		configB().FullName(): configB().Checksum(),
	}

	equals(t, len(a.changes), 0)
	equals(t, d.UpdatedChecksums, expectedChecksums)
	equals(t, d.UpdatedRestart, false)
}

func TestConfigChangesConfigChangeGetDeploymentChecksumsUpdatedWithRestart(t *testing.T) {
	a := agent()
	c := configAUpdated()
	d := deploymentA()

	a.Start(nil)
	a.ResourceUpdated(c)
	a.ResourceUpdated(d)
	time.Sleep(150 * time.Millisecond)
	a.Stop()

	expectedChecksums := map[string]string{
		c.FullName():         c.Checksum(),
		configB().FullName(): configB().Checksum(),
	}

	equals(t, len(a.changes), 0)
	equals(t, d.UpdatedChecksums, expectedChecksums)
	equals(t, d.UpdatedRestart, true)
}

func TestConfigChangesConfigChangeCleansUpDeploymentChange(t *testing.T) {
	a := agent()
	c := configAUpdated()
	d1 := deploymentA()
	d2 := deploymentAUpdated()
	d2.AppliedChecksumsValue = map[string]string{}

	a.Start(nil)
	a.ResourceUpdated(c)
	a.ResourceUpdated(d1)
	a.ResourceUpdated(d2)
	time.Sleep(150 * time.Millisecond)
	a.Stop()

	expectedChecksums := map[string]string{
		c.FullName(): c.Checksum(),
	}

	equals(t, len(a.changes), 0)
	equals(t, d2.UpdatedChecksums, expectedChecksums)
	equals(t, d2.UpdatedRestart, false)
}

func TestConfigChangesMissingDeploymentAnnotaionsGetRestored(t *testing.T) {
	a := agent()
	c := configA()
	d1 := deploymentA()
	d2 := deploymentAUpdated()
	delete(d2.AppliedChecksumsValue, c.FullName())

	a.Start(nil)
	a.ResourceUpdated(c)
	a.ResourceUpdated(d1)
	a.ResourceUpdated(d2)
	time.Sleep(150 * time.Millisecond)
	a.Stop()

	expectedChecksums := map[string]string{
		c.FullName():         c.Checksum(),
		configB().FullName(): configB().Checksum(),
	}

	equals(t, len(a.changes), 0)
	equals(t, d2.UpdatedChecksums, expectedChecksums)
	equals(t, d2.UpdatedRestart, false)
}

func TestConfigChangesUnsavedImportantChangesGetAppliedWhenStopped(t *testing.T) {
	a := agent()
	d := deploymentA()
	c1 := configA()
	c2 := configAUpdated()
	delete(d.AppliedChecksumsValue, c1.FullName())

	a.Start(nil)
	a.ResourceUpdated(d)
	a.ResourceUpdated(c1)
	a.ResourceUpdated(c2)
	a.Stop()

	time.Sleep(150 * time.Millisecond)

	expectedChecksums := map[string]string{
		c2.FullName():        c2.Checksum(),
		configB().FullName(): configB().Checksum(),
	}

	equals(t, len(a.changes), 0)
	equals(t, d.UpdatedChecksums, expectedChecksums)
	equals(t, d.UpdatedRestart, true)
}

func TestConfigChangesProcessStopsOnError(t *testing.T) {
	a := agent()
	c := configAUpdated()
	d := deploymentA()
	d.UpdateError = errors.New("Dummy error")

	controllerStopCh := make(chan struct{})
	a.Start(controllerStopCh)
	a.ResourceUpdated(c)
	a.ResourceUpdated(d)
	time.Sleep(150 * time.Millisecond)
	<-controllerStopCh

	expectedChecksums := map[string]string{
		c.FullName():         c.Checksum(),
		configB().FullName(): configB().Checksum(),
	}

	equals(t, d.UpdatedChecksums, expectedChecksums)
	equals(t, d.UpdatedRestart, true)
}

func TestConfigChangesProcessContinuesOnIgnoredErrors(t *testing.T) {
	a := agent()
	c := configAUpdated()
	d := deploymentA()
	d.UpdateError = errors.New("ignore-me")

	a.Start(nil)
	a.ResourceUpdated(c)
	a.ResourceUpdated(d)
	time.Sleep(150 * time.Millisecond)
	a.Stop()

	equals(t, len(a.changes), 0)
	equals(t, d.UpdatedRestart, true)
}

func configA() *test.DummyMetaConfig {
	return test.NewMetaConfigWithParams("configmap/test/test", "12345", "abc")
}

func configAUpdated() *test.DummyMetaConfig {
	return test.NewMetaConfigWithParams("configmap/test/test", "23456", "bcd")
}

func configB() *test.DummyMetaConfig {
	return test.NewMetaConfigWithParams("secret/test/test", "67890", "def")
}

func deploymentA() *test.DummyMetaDeployment {
	d := test.NewDummyMetaDeployment()
	d.FullNameValue = "deployment/test/test-deployment"
	d.VersionValue = "12345"
	d.ReferencedConfigsValue = []string{configA().FullName(), configB().FullName()}
	d.AppliedChecksumsValue = map[string]string{
		configA().FullName(): configA().Checksum(),
		configB().FullName(): configB().Checksum(),
	}
	d.NeedsRestartOnConfigChangeValue = true
	return d
}

func deploymentAUpdated() *test.DummyMetaDeployment {
	d := deploymentA()
	d.VersionValue = "23456"
	return d
}

func deploymentB() *test.DummyMetaDeployment {
	d := test.NewDummyMetaDeployment()
	d.FullNameValue = "statefulset/test/test-statefulset"
	d.VersionValue = "12345"
	d.ReferencedConfigsValue = []string{configA().FullName(), configB().FullName()}
	d.NeedsRestartOnConfigChangeValue = true
	return d
}

func agent() *RealConfigAgent {
	// flag.Set("logtostderr", "true")
	// flag.Set("v", "3")
	// flag.CommandLine.Parse([]string{})
	agent := NewConfigAgent(util.Clientset(), 20*time.Millisecond, 100*time.Millisecond, []string{"ignore-me"}).(*RealConfigAgent)
	agent.k8sClient = test.NewDummyK8sClient()
	return agent
}
