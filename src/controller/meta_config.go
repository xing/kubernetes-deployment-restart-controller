package controller

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"source.xing.com/olympus/kubernetes-deployment-restart-controller/src/controller/interfaces"
)

const (
	configTypeSecret    = "secret"
	configTypeConfigMap = "configmap"
)

type metaConfig struct {
	meta    metav1.ObjectMeta
	typ     string
	dataSha string
}

func (c *metaConfig) FullName() string { return FullName(c.typ, c.meta.Namespace, c.meta.Name) }
func (c *metaConfig) Version() string  { return c.meta.ResourceVersion }
func (c *metaConfig) Checksum() string { return c.dataSha }

// MetaConfigFromConfigMap converts a ConfigMap into MetaConfig
func MetaConfigFromConfigMap(cm *v1.ConfigMap) interfaces.MetaConfig {
	return &metaConfig{
		meta:    cm.ObjectMeta,
		typ:     configTypeConfigMap,
		dataSha: getSha(cm.Data),
	}
}

// MetaConfigFromSecret converts a Secret into MetaConfig
func MetaConfigFromSecret(s *v1.Secret) interfaces.MetaConfig {
	return &metaConfig{
		meta:    s.ObjectMeta,
		typ:     configTypeSecret,
		dataSha: getSha(s.Data),
	}
}

// FullName builds a full name to identify a MetaResource
func FullName(typ, namespace, name string) string {
	return fmt.Sprintf("%s/%s/%s", typ, namespace, name)
}

func getSha(data interface{}) string {
	bytes, _ := json.Marshal(data)
	shaBytes := sha256.Sum256(bytes)
	return fmt.Sprintf("%x", shaBytes[:8])
}
