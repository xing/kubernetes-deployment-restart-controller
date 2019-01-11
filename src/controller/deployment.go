package controller

import (
	"github.com/golang/glog"
	"source.xing.com/olympus/kubernetes-deployment-restart-controller/src/controller/interfaces"
)

// Deployment stores a MetaDeployment instance and a map of configs referenced by it
type Deployment struct {
	meta             interfaces.MetaDeployment
	Configs          map[string]*Config
	AppliedChecksums map[string]string
}

// NewDeployment creates a new deployment with a bound MetaDeployment object
func NewDeployment(meta interfaces.MetaDeployment) *Deployment {
	return &Deployment{
		meta:             meta,
		Configs:          make(map[string]*Config),
		AppliedChecksums: meta.AppliedChecksums(),
	}
}

// UpdateFromMeta replaces the underlying MetaDeployment object and populates
// AppliedChecksums. Returns true if referenced configs or config checksums changed,
// otherwise returns false
func (d *Deployment) UpdateFromMeta(meta interfaces.MetaDeployment) bool {
	configInfoChanged := !stringSlicesEqual(d.meta.ReferencedConfigs(), meta.ReferencedConfigs()) || !stringMapsEqual(d.AppliedChecksums, meta.AppliedChecksums())

	d.meta = meta
	d.AppliedChecksums = meta.AppliedChecksums()

	return configInfoChanged
}

// NeedsUpdate returns true if underlying k8s resource needs an update according to the
// current known state of config checksums and related configs
func (d *Deployment) NeedsUpdate() bool {
	for name, config := range d.Configs {
		if config.Pending() {
			continue
		}

		if config.Checksum() != d.AppliedChecksums[name] {
			return true
		}
	}

	for name := range d.AppliedChecksums {
		if _, ok := d.Configs[name]; !ok {
			return true
		}
	}

	return false
}

// SaveChecksums saves config checksums stored in the deployment instance as annotations
// on the k8s resource, optionally triggering a restart
func (d *Deployment) SaveChecksums(c interfaces.K8sClient, restart bool) error {
	glog.V(2).Infof("Deployment %s will have config checksums updated", d.meta.FullName())

	if restart {
		glog.V(1).Infof("Deployment %s will be restarted", d.meta.FullName())
	}

	return d.meta.UpdateConfigChecksums(c, d.AppliedChecksums, restart)
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v := range a {
		if b[k] != v {
			return false
		}
	}

	return true
}

func stringMapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v := range a {
		if b[k] != v {
			return false
		}
	}

	return true
}
