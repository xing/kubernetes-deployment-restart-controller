package controller

import (
	"errors"
	"testing"

	"github.com/xing/kubernetes-deployment-restart-controller/src/controller/test"
)

func TestNewDeploymentReturnsDeploymentWithConfigChecksumsFromMeta(t *testing.T) {
	meta := test.NewDummyMetaDeployment()
	meta.AppliedChecksumsValue = map[string]string{"name": "value"}

	d := NewDeployment(meta)
	equals(t, d.AppliedChecksums, meta.AppliedChecksums())
}

func TestDeploymentUpdateFromMetaUpdatesConfigChecksumsFromMeta(t *testing.T) {
	meta1 := test.NewDummyMetaDeployment()
	meta1.AppliedChecksumsValue = map[string]string{"name1": "value1"}

	meta2 := test.NewDummyMetaDeployment()
	meta2.AppliedChecksumsValue = map[string]string{"name2": "value2"}

	d := NewDeployment(meta1)
	d.UpdateFromMeta(meta2)
	equals(t, d.AppliedChecksums, meta2.AppliedChecksums())
}

func TestDeploymentUpdateFromMetaReturnsFalseWhenConfigChecksumsAndReferencedConfigsAreTheSame(t *testing.T) {
	meta1 := test.NewDummyMetaDeployment()
	meta1.AppliedChecksumsValue = map[string]string{"name1": "value1"}
	meta1.ReferencedConfigsValue = []string{"config1"}

	meta2 := test.NewDummyMetaDeployment()
	meta2.AppliedChecksumsValue = map[string]string{"name1": "value1"}
	meta2.ReferencedConfigsValue = []string{"config1"}

	d := NewDeployment(meta1)
	equals(t, d.UpdateFromMeta(meta2), false)
}

func TestDeploymentUpdateFromMetaReturnsTrueWhenConfigChecksumsAreDifferent(t *testing.T) {
	meta1 := test.NewDummyMetaDeployment()
	meta1.AppliedChecksumsValue = map[string]string{"name1": "value1"}

	meta2 := test.NewDummyMetaDeployment()
	meta2.AppliedChecksumsValue = map[string]string{"name1": "value1", "name2": "value2"}

	d := NewDeployment(meta1)
	equals(t, d.UpdateFromMeta(meta2), true)
}

func TestDeploymentUpdateFromMetaReturnsTrueWhenReferecedConfigsAreDifferent(t *testing.T) {
	meta1 := test.NewDummyMetaDeployment()
	meta1.ReferencedConfigsValue = []string{"config1"}

	meta2 := test.NewDummyMetaDeployment()
	meta2.ReferencedConfigsValue = []string{"config1", "config2"}

	d := NewDeployment(meta1)
	equals(t, d.UpdateFromMeta(meta2), true)
}

func TestDeploymentUpdateFromMetaReturnsTrueWhenReferecingTheSameNumberOfConfigsButConfigsAreDifferent(t *testing.T) {
	meta1 := test.NewDummyMetaDeployment()
	meta1.ReferencedConfigsValue = []string{"config1"}

	meta2 := test.NewDummyMetaDeployment()
	meta2.ReferencedConfigsValue = []string{"config2"}

	d := NewDeployment(meta1)
	equals(t, d.UpdateFromMeta(meta2), true)
}

func TestDeploymentNeedsUpdateReturnsFalseWhenAllConfigsArePending(t *testing.T) {
	meta := test.NewDummyMetaDeployment()

	d := NewDeployment(meta)
	d.Configs = map[string]*Config{"config1": NewPendingConfig()}

	equals(t, d.NeedsUpdate(), false)
}

func TestDeploymentNeedsUpdateReturnsFalseWhenAllConfigsHaveCorrectChecksums(t *testing.T) {
	metaD := test.NewDummyMetaDeployment()
	metaD.AppliedChecksumsValue = map[string]string{"config": "checksum"}

	metaC := test.NewDummyMetaConfig("checksum")

	d := NewDeployment(metaD)
	d.Configs = map[string]*Config{"config": NewConfig(metaC)}

	equals(t, d.NeedsUpdate(), false)
}

func TestDeploymentNeedsUpdateReturnsTrueWhenThereIsAConfigWithoutChecksum(t *testing.T) {
	metaD := test.NewDummyMetaDeployment()
	metaC := test.NewDummyMetaConfig("checksum")

	d := NewDeployment(metaD)
	d.Configs = map[string]*Config{"config": NewConfig(metaC)}

	equals(t, d.NeedsUpdate(), true)
}

func TestDeploymentNeedsUpdateReturnsTrueWhenThereIsAnOrphanedChecksum(t *testing.T) {
	metaD := test.NewDummyMetaDeployment()
	metaD.AppliedChecksumsValue = map[string]string{"config": "checksum", "other": "checksome"}

	metaC := test.NewDummyMetaConfig("checksum")

	d := NewDeployment(metaD)
	d.Configs = map[string]*Config{"config": NewConfig(metaC)}

	equals(t, d.NeedsUpdate(), true)
}

func TestDeploymentNeedsUpdateReturnsTrueWhenChecksumDoesNotMatch(t *testing.T) {
	metaD := test.NewDummyMetaDeployment()
	metaD.AppliedChecksumsValue = map[string]string{"config": "checksum"}

	metaC := test.NewDummyMetaConfig("different checksum")

	d := NewDeployment(metaD)
	d.Configs = map[string]*Config{"config": NewConfig(metaC)}

	equals(t, d.NeedsUpdate(), true)
}

func TestDeploymentSaveChecksumsCallsMetaUpdateConfigChecksums(t *testing.T) {
	meta := test.NewDummyMetaDeployment()
	meta.AppliedChecksumsValue = map[string]string{"config": "checksum"}

	d := NewDeployment(meta)
	err := d.SaveChecksums(nil, true)

	equals(t, err, nil)
	equals(t, meta.UpdatedChecksums, d.AppliedChecksums)
	equals(t, meta.UpdatedRestart, true)
}

func TestDeploymentSaveChecksumsForwardsTheError(t *testing.T) {
	meta := test.NewDummyMetaDeployment()
	err := errors.New("Oh no")
	meta.UpdateError = err

	d := NewDeployment(meta)
	e := d.SaveChecksums(nil, true)

	equals(t, e, err)
}
