package test

import (
	"github.com/xing/kubernetes-deployment-restart-controller/src/controller/interfaces"
)

// DummyConfigAgent replaces RealConfigAgent in tests
type DummyConfigAgent struct {
	UpdatedResources []interfaces.MetaResource
	DeletedResources []interfaces.MetaResource
}

// NewDummyConfigAgent returns a new DummyConfigAgent instance
func NewDummyConfigAgent() interfaces.ConfigAgent {
	return &DummyConfigAgent{}
}

func (d *DummyConfigAgent) Start(controllerStopCh chan struct{}) {}
func (d *DummyConfigAgent) Stop()                                {}

func (d *DummyConfigAgent) ResourceUpdated(res interfaces.MetaResource) {
	d.UpdatedResources = append(d.UpdatedResources, res)
}

func (d *DummyConfigAgent) ResourceDeleted(res interfaces.MetaResource) {
	d.DeletedResources = append(d.DeletedResources, res)
}
