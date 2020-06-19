package controller

import (
	"time"

	"github.com/golang/glog"
	"github.com/xing/kubernetes-deployment-restart-controller/src/controller/interfaces"
	"github.com/xing/kubernetes-deployment-restart-controller/src/lib"
	"k8s.io/client-go/kubernetes"
)

// RealConfigAgent implements interfaces.ConfigAgent
type RealConfigAgent struct {
	updateResourceCh chan interfaces.MetaResource
	deleteResourceCh chan interfaces.MetaResource

	restartCheckPeriod time.Duration
	restartGracePeriod time.Duration

	configs     map[string]*Config
	deployments map[string]*Deployment

	versions map[string]string
	changes  map[string]*Change

	k8sClient        interfaces.K8sClient
	processChangesCh chan struct{}
	stopCh           chan struct{}
	stoppedCh        chan struct{}

	stopWithErrorCh chan struct{}
}

// NewConfigAgent creates a new real instance of interfaces.ConfigAgent
func NewConfigAgent(k8sClient kubernetes.Interface, restartCheckPeriod, restartGracePeriod time.Duration) interfaces.ConfigAgent {
	return &RealConfigAgent{
		updateResourceCh: make(chan interfaces.MetaResource),
		deleteResourceCh: make(chan interfaces.MetaResource),

		restartCheckPeriod: restartCheckPeriod,
		restartGracePeriod: restartGracePeriod,

		configs:     make(map[string]*Config),
		deployments: make(map[string]*Deployment),

		versions: make(map[string]string),
		changes:  make(map[string]*Change),

		k8sClient:        lib.NewK8sClient(k8sClient),
		processChangesCh: make(chan struct{}),
		stopCh:           make(chan struct{}),
		stoppedCh:        make(chan struct{}),
	}
}

// ResourceUpdated tracks k8s resource updates and additions
func (c *RealConfigAgent) ResourceUpdated(res interfaces.MetaResource) {
	c.updateResourceCh <- res
}

// ResourceDeleted tracks k8s resource deletions to allow cleanups of internal state
func (c *RealConfigAgent) ResourceDeleted(res interfaces.MetaResource) {
	c.deleteResourceCh <- res
}

// Start the agent as a goroutine
func (c *RealConfigAgent) Start(stopWithErrorCh chan struct{}) {
	c.stopWithErrorCh = stopWithErrorCh
	go c.updateLoop()
	go func() {
		ticker := time.NewTicker(c.restartCheckPeriod)
		for range ticker.C {
			c.processChangesCh <- struct{}{}
		}
	}()
}

// Stop the agent gracefully
func (c *RealConfigAgent) Stop() {
	c.stopCh <- struct{}{}
	<-c.stoppedCh
	glog.V(1).Info("Config agent stopped")
}

func (c *RealConfigAgent) updateLoop() {
	gracefulChange := func(change *Change) bool {
		return change.Age() >= c.restartGracePeriod
	}

	memoryStateSensitiveChange := func(change *Change) bool {
		// Multiple observations of a config change can cause a restart to be missed if
		// the change is discarded. We should process them before terminating the agent.
		// See updateDeployment method for more details.
		return change.Observations > 1

		// TODO: there's another possibility: a new deployment is added that references an
		// existing config. The config then is updated within the grace period of the
		// deployment change, and before the deployment change gets processed controller
		// gets terminated. Would require the name of the affected resource to be stored
		// to check for this situation.
	}

	for {
		select {
		case res := <-c.updateResourceCh:
			if c.knownVersion(res) {
				continue
			}
			switch v := res.(type) {
			case interfaces.MetaConfig:
				c.trackConfig(v)
			case interfaces.MetaDeployment:
				c.trackDeployment(v)
			}
			c.updateResourceGaugeMetrics()
			ResourceVersionsTotal.WithLabelValues().Inc()

		case res := <-c.deleteResourceCh:
			switch v := res.(type) {
			case interfaces.MetaConfig:
				c.cleanupConfig(v)
			case interfaces.MetaDeployment:
				c.cleanupDeployment(v)
			}
			c.cleanupVersion(res)
			c.updateResourceGaugeMetrics()

		case <-c.processChangesCh:
			c.processChanges(gracefulChange)
			ChangesWaitingTotal.WithLabelValues().Set(float64(len(c.changes)))

		case <-c.stopCh:
			c.processChanges(memoryStateSensitiveChange)
			close(c.stopCh)
			c.stoppedCh <- struct{}{}
			return
		}
	}
}

func (c *RealConfigAgent) knownVersion(res interfaces.MetaResource) bool {
	name := res.FullName()
	version := res.Version()

	if c.versions[name] != version {
		glog.V(3).Infof("New version of %s: %s", name, version)
		c.versions[name] = version
		return false
	}

	glog.V(3).Infof("Version unchanged: %s", name)
	return true
}

func (c *RealConfigAgent) trackConfig(meta interfaces.MetaConfig) {
	name := meta.FullName()

	config, ok := c.configs[name]
	if ok {
		if !config.UpdateFromMeta(meta) {
			return
		}

		glog.V(1).Infof("Config %s updated. New checksum: %s", name, config.Checksum())
	} else {
		c.configs[name] = NewConfig(meta)

		glog.V(3).Infof("Config %s added", name)
	}

	c.trackResourceChange(name)
}

func (c *RealConfigAgent) trackDeployment(meta interfaces.MetaDeployment) {
	name := meta.FullName()

	if !meta.NeedsRestartOnConfigChange() {
		glog.V(3).Infof("Deployment %s does not participate in dynamic config", name)
		c.cleanupDeployment(meta)
		return
	}

	orphanedConfigs := make(map[string]struct{})

	deployment, ok := c.deployments[name]
	if ok {
		if !deployment.UpdateFromMeta(meta) {
			return
		}

		glog.V(3).Infof("Deployment %s updated", name)

		for k := range deployment.Configs {
			orphanedConfigs[k] = struct{}{} // any current config can become orphaned
		}
	} else {
		deployment = NewDeployment(meta)
		c.deployments[name] = deployment

		glog.V(3).Infof("Deployment %s added", name)
	}

	for _, configName := range meta.ReferencedConfigs() {
		c.linkConfigToDeployment(configName, name)
		delete(orphanedConfigs, configName) // a config is not orphaned if referenced
	}

	for configName := range orphanedConfigs {
		config := c.configs[configName]
		delete(config.Deployments, name)
		delete(deployment.Configs, configName)

		glog.V(3).Infof("Config %s is no longer referenced by deployment %s", configName, name)

		if config.Unused() {
			c.cleanupConfigByName(configName)
		}
	}

	c.trackResourceChange(name)
}

func (c *RealConfigAgent) linkConfigToDeployment(configName, deploymentName string) {
	deployment := c.deployments[deploymentName]

	if _, ok := deployment.Configs[configName]; ok {
		return
	}

	glog.V(3).Infof("Tracking config %s for %s ", configName, deploymentName)

	config, ok := c.configs[configName]
	if !ok {
		config = NewPendingConfig()
		c.configs[configName] = config

		glog.V(3).Infof("Config %s is pending", configName)
	}

	deployment.Configs[configName] = config
	config.Deployments[deploymentName] = deployment
}

func (c *RealConfigAgent) trackResourceChange(name string) {
	change, ok := c.changes[name]
	if ok {
		change.Observations++
		glog.V(3).Infof("Resource %s changed %d times", name, change.Observations)
	} else {
		c.changes[name] = NewChange()
		glog.V(3).Infof("Resource %s changed", name)
	}
}

func (c *RealConfigAgent) cleanupVersion(res interfaces.MetaResource) {
	name := res.FullName()
	delete(c.versions, name)

	glog.V(3).Infof("Deleted version information for %s", name)
}

func (c *RealConfigAgent) cleanupDeployment(metaDeployment interfaces.MetaDeployment) {
	deploymentName := metaDeployment.FullName()

	deployment, ok := c.deployments[deploymentName]
	if !ok {
		return
	}

	for configName, config := range deployment.Configs {
		delete(config.Deployments, deploymentName)

		if config.Unused() {
			c.cleanupConfigByName(configName)
		}
	}

	delete(c.deployments, deploymentName)
	delete(c.changes, deploymentName)

	glog.V(3).Infof("Cleaned up deployment %s", deploymentName)
}

func (c *RealConfigAgent) cleanupConfig(meta interfaces.MetaConfig) {
	c.cleanupConfigByName(meta.FullName())
}

func (c *RealConfigAgent) cleanupConfigByName(name string) {
	if config, ok := c.configs[name]; ok {
		for _, deployment := range config.Deployments {
			delete(deployment.Configs, name)
		}
	}

	delete(c.configs, name)
	delete(c.changes, name)

	glog.V(3).Infof("Cleaned up config %s", name)
}

func (c *RealConfigAgent) processChanges(applicable func(*Change) bool) {
	if len(c.changes) == 0 {
		return
	}

	var processedChanges []string
	deploymentsToBeUpdated := make(map[string]struct{})

	glog.V(3).Infof("Changes in the queue: %d", len(c.changes))

	for resourceName, change := range c.changes {
		if !applicable(change) {
			continue
		}

		processedChanges = append(processedChanges, resourceName)

		deployments := c.affectedDeployments(resourceName)
		if deployments == nil {
			glog.Warningf("Orphaned resource change ignored: %s", resourceName)
			continue
		}

		glog.V(2).Infof("Processing resource change: %s", resourceName)

		for deploymentName, deployment := range deployments {
			if deployment.NeedsUpdate() {
				deploymentsToBeUpdated[deploymentName] = struct{}{}
				glog.V(2).Infof("Deployment %s needs an update", deploymentName)
			}
		}
	}

	for deploymentName := range deploymentsToBeUpdated {
		c.updateDeployment(c.deployments[deploymentName])
		delete(c.changes, deploymentName) // Any potential deployment change has been applied
		DeploymentAnnotationUpdatesTotal.WithLabelValues().Inc()
	}

	ChangesProcessedTotal.WithLabelValues().Add(float64(len(processedChanges)))

	for _, v := range processedChanges {
		delete(c.changes, v)
	}
}

func (c *RealConfigAgent) affectedDeployments(resourceName string) map[string]*Deployment {
	if config, ok := c.configs[resourceName]; ok {
		return config.Deployments
	}

	if deployment, ok := c.deployments[resourceName]; ok {
		return map[string]*Deployment{resourceName: deployment}
	}

	return nil
}

func (c *RealConfigAgent) updateDeployment(deployment *Deployment) {
	restart := false

	for name, config := range deployment.Configs {
		if config.Pending() || config.Checksum() == deployment.AppliedChecksums[name] {
			continue
		}

		// TODO: consider this situation: deployment gets created, it references an
		// existing config. Shortly after that the config is modified. Deployment should
		// be restarted, but if the config change gets processed first, it will not be.
		// This needs to be fixed. Could be enough to order the changes by timestamp.

		if _, ok := deployment.AppliedChecksums[name]; ok {
			restart = true
		} else {
			change, ok := c.changes[name]
			if ok && change.Observations > 1 {
				// Normally, having no config checksum in deployment annotation would mean
				// the config was recently added to the deployment and is in fact already
				// applied (restart was triggered by spec change). However, multiple
				// observations mean the config was added to the deployment and then
				// updated, before the controller managed to react to the addition. In
				// that case deployment must be restarted to apply the change.
				restart = true
			}
		}

		deployment.AppliedChecksums[name] = config.Checksum()
	}

	// Purge checksums that reference unknown configs
	for name := range deployment.AppliedChecksums {
		if _, ok := deployment.Configs[name]; !ok {
			delete(deployment.AppliedChecksums, name)
		}
	}

	err := deployment.SaveChecksums(c.k8sClient, restart)
	if err != nil {
		c.stopWithError(err)
	}

	if restart {
		DeploymentRestartsTotal.WithLabelValues().Inc()
	}
}

func (c *RealConfigAgent) stopWithError(err error) {
	glog.Error(err)
	go func() { c.stopCh <- struct{}{} }()
	c.stopWithErrorCh <- struct{}{}
}

func (c *RealConfigAgent) updateResourceGaugeMetrics() {
	ConfigsTotal.WithLabelValues().Set(float64(len(c.configs)))
	DeploymentsTotal.WithLabelValues().Set(float64(len(c.deployments)))
}
