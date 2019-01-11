package controller

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"source.xing.com/olympus/kubernetes-deployment-restart-controller/src/controller/test"
)

func TestNewPendingConfigReturnsPendingConfig(t *testing.T) {
	c := NewPendingConfig()
	equals(t, c.Pending(), true)
}

func TestPendingConfigHasEmptyChecksum(t *testing.T) {
	c := NewPendingConfig()
	equals(t, c.Checksum(), "")
}

func TestNewConfigRetusnNotPendingConfig(t *testing.T) {
	c := NewConfig(test.NewDummyMetaConfig("checksum"))
	equals(t, c.Pending(), false)
}

func TestConfigChecksumReturnsMetaChecksum(t *testing.T) {
	meta := test.NewDummyMetaConfig("checksum")
	c := NewConfig(meta)
	equals(t, c.Checksum(), meta.Checksum())
}

func TestConfigUpdateMetaChangesTheChecksum(t *testing.T) {
	meta1 := test.NewDummyMetaConfig("checksum1")
	meta2 := test.NewDummyMetaConfig("checksum2")
	c := NewConfig(meta1)
	c.UpdateFromMeta(meta2)
	equals(t, c.Checksum(), meta2.Checksum())
}

func TestConfigUpdateMetaReturnsTrueIfChecksumChanged(t *testing.T) {
	meta1 := test.NewDummyMetaConfig("checksum1")
	meta2 := test.NewDummyMetaConfig("checksum2")
	c := NewConfig(meta1)
	equals(t, c.UpdateFromMeta(meta2), true)
}

func TestConfigUpdateMetaReturnsFalseIfChecksumIsNotChanged(t *testing.T) {
	meta1 := test.NewDummyMetaConfig("checksum")
	meta2 := test.NewDummyMetaConfig("checksum")
	c := NewConfig(meta1)
	equals(t, c.UpdateFromMeta(meta2), false)
}

func TestPendingConfigIsUnused(t *testing.T) {
	c := NewPendingConfig()
	equals(t, c.Unused(), true)
}

func TestConfigWithMetaIsNotUnused(t *testing.T) {
	c := NewConfig(test.NewDummyMetaConfig("checksum1"))
	equals(t, c.Unused(), false)
}

func TestPendingConfigWithDeploymentsIsNotUnused(t *testing.T) {
	c := NewPendingConfig()
	c.Deployments["whatever"] = NewDeployment(test.NewDummyMetaDeployment())
	equals(t, c.Unused(), false)
}

func equals(t *testing.T, got, expected interface{}, args ...interface{}) (err error) {
	t.Helper()
	if !reflect.DeepEqual(got, expected) {
		err = fmt.Errorf("\nExpected: %#v\nGot:      %#v", expected, got)
		if len(args) > 0 {
			err = errors.New(err.Error() + "\n" + fmt.Sprintf("%s", args))
		}
		t.Error(err)
	}
	return
}
