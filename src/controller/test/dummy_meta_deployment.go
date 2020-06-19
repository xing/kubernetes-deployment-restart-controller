package test

import (
	"github.com/xing/kubernetes-deployment-restart-controller/src/controller/interfaces"
)

type DummyMetaDeployment struct {
	FullNameValue                   string
	VersionValue                    string
	NeedsRestartOnConfigChangeValue bool
	ReferencedConfigsValue          []string
	AppliedChecksumsValue           map[string]string

	UpdateError      error
	UpdatedChecksums map[string]string
	UpdatedRestart   bool
}

// NewDummyK8sClient returns a dummy implementation
func NewDummyMetaDeployment() *DummyMetaDeployment {
	return &DummyMetaDeployment{}
}

func (d *DummyMetaDeployment) FullName() string { return d.FullNameValue }
func (d *DummyMetaDeployment) Version() string  { return d.VersionValue }
func (d *DummyMetaDeployment) NeedsRestartOnConfigChange() bool {
	return d.NeedsRestartOnConfigChangeValue
}
func (d *DummyMetaDeployment) ReferencedConfigs() []string         { return d.ReferencedConfigsValue }
func (d *DummyMetaDeployment) AppliedChecksums() map[string]string { return d.AppliedChecksumsValue }

func (d *DummyMetaDeployment) UpdateConfigChecksums(k8sClient interfaces.K8sClient, checksums map[string]string, restart bool) error {
	d.UpdatedChecksums = checksums
	d.UpdatedRestart = restart
	return d.UpdateError
}
