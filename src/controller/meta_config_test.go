package controller

import (
	"fmt"
	"testing"

	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMetaConfigFromConfigMapReturnsValidMetaConfig(t *testing.T) {
	namespace := "test-namespace"
	name := "test-name"
	version := "1029384756"
	data := map[string]string{"key": "value"}
	c := newConfigMap(namespace, name, version, data)

	mc := MetaConfigFromConfigMap(c)
	expectedChecksum := "e43abcf337524483" // echo -n '{"key":"value"}' | shasum -a 256 | cut -c1-16

	equals(t, mc.FullName(), fmt.Sprintf("configmap/%s/%s", namespace, name))
	equals(t, mc.Version(), version)
	equals(t, mc.Checksum(), expectedChecksum)
}

func TestMetaConfigFromSecretReturnsValidMetaConfig(t *testing.T) {
	namespace := "test-namespace"
	name := "test-name"
	version := "1029384756"
	data := map[string][]byte{"key": []byte("value")}
	s := newSecret(namespace, name, version, data)

	mc := MetaConfigFromSecret(s)
	expectedChecksum := "fed7a27106a07691" // echo -n "{\"key\":\"`echo -n "value" | base64`\"}" | shasum -a 256 | cut -c1-16

	equals(t, mc.FullName(), fmt.Sprintf("secret/%s/%s", namespace, name))
	equals(t, mc.Version(), version)
	equals(t, mc.Checksum(), expectedChecksum)
}

func newConfigMap(namespace, name, version string, data map[string]string) *core.ConfigMap {
	if data == nil {
		data = map[string]string{}
	}

	return &core.ConfigMap{
		ObjectMeta: meta.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			ResourceVersion: version,
		},
		Data: data,
	}
}

func newSecret(namespace, name, version string, data map[string][]byte) *core.Secret {
	if data == nil {
		data = map[string][]byte{}
	}

	return &core.Secret{
		ObjectMeta: meta.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			ResourceVersion: version,
		},
		Data: data,
	}
}
