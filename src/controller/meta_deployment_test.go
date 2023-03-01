package controller

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	yaml "gopkg.in/yaml.v3"
	apps "k8s.io/api/apps/v1"

	"github.com/xing/kubernetes-deployment-restart-controller/src/controller/test"
)

func TestMetaDeploymentFromDeploymentReturnsValidMetaDeployment(t *testing.T) {
	d := newDeploymentFromYAML(`
---
metadata:
  name: test-name
  namespace: test-namespace
  resourceVersion: "123456"
  annotations:
    com.xing.deployment-restart.applied-config-checksums: |
        {"config-one":"checksum-one"}
spec:
  template:
    spec:
      containers:
      - envFrom:
        - configMapRef:
            name: config-one`)

	md := MetaDeploymentFromDeployment(d)

	equals(t, md.FullName(), "deployment/test-namespace/test-name")
	equals(t, md.Version(), "123456")
	equals(t, md.ReferencedConfigs(), []string{"configmap/test-namespace/config-one"})
	equals(t, md.AppliedChecksums(), map[string]string{"config-one": "checksum-one"})
}

func TestMetaDeploymentFromStatefulSetReturnsValidMetaDeployment(t *testing.T) {
	s := newStatefulSetFromYAML(`
---
metadata:
  name: test-name
  namespace: test-namespace
  resourceVersion: "123456"
  annotations:
    com.xing.deployment-restart.applied-config-checksums: |
        {"config-one":"checksum-one"}
spec:
  template:
    spec:
      containers:
      - envFrom:
        - configMapRef:
            name: config-one`)

	md := MetaDeploymentFromStatefulSet(s)

	equals(t, md.FullName(), "statefulset/test-namespace/test-name")
	equals(t, md.Version(), "123456")
	equals(t, md.ReferencedConfigs(), []string{"configmap/test-namespace/config-one"})
	equals(t, md.AppliedChecksums(), map[string]string{"config-one": "checksum-one"})
}

func TestMetaDeploymentNeedsRestartOnConfigChangeReturnsTrueWhenAnnotationHasTheRightValue(t *testing.T) {
	d := newDeploymentFromYAML(`
---
metadata:
  name: test-name
  namespace: test-namespace
  annotations:
    com.xing.deployment-restart: enabled
`)

	md := MetaDeploymentFromDeployment(d)
	equals(t, md.NeedsRestartOnConfigChange(), true)
}

func TestMetaDeploymentNeedsRestartOnConfigChangeReturnsFalseWhenAnnotationIsNotSet(t *testing.T) {
	d := newDeploymentFromYAML(`
---
metadata:
  name: test-name
  namespace: test-namespace
`)

	md := MetaDeploymentFromDeployment(d)
	equals(t, md.NeedsRestartOnConfigChange(), false)
}

func TestMetaDeploymentConfigChecksumsRetursChecksumsFromAnnotation(t *testing.T) {
	d := newDeploymentFromYAML(`
---
metadata:
  name: test-name
  namespace: test-namespace
  annotations:
    com.xing.deployment-restart.applied-config-checksums: |
        {"config-one":"checksum-one","config-two":"checksum-two"}
`)

	md := MetaDeploymentFromDeployment(d)
	expected := map[string]string{
		"config-one": "checksum-one",
		"config-two": "checksum-two",
	}

	equals(t, md.AppliedChecksums(), expected)
}

func TestMetaDeploymentConfigChecksumsRetursEmptyMapWhenAnnotationDoesNotExist(t *testing.T) {
	d := newDeploymentFromYAML(`
---
metadata:
  name: test-name
  namespace: test-namespace
`)

	md := MetaDeploymentFromDeployment(d)
	expected := map[string]string{}

	equals(t, md.AppliedChecksums(), expected)
}

func TestMetaDeploymentConfigChecksumsRetursEmptyMapWhenAnnotationContainsInvalidJSON(t *testing.T) {
	d := newDeploymentFromYAML(`
---
metadata:
  name: test-name
  namespace: test-namespace
  annotations:
    com.xing.deployment-restart.applied-config-checksums: |
        NOT A JSON STRING
`)

	md := MetaDeploymentFromDeployment(d)
	expected := map[string]string{}

	equals(t, md.AppliedChecksums(), expected)
}

func TestMetaDeploymentReferencedConfigsCollectsAllConfigReferencesInAlphabeticalOrder(t *testing.T) {
	d := newDeploymentFromYAML(`
---
metadata:
  name: test-name
  namespace: test-namespace
spec:
  template:
    spec:
      containers:
      - envFrom:
        - secretRef:
            name: secret-b
        - secretRef:
            name: secret-a
        - configMapRef:
            name: config-c
        - configMapRef:
            name: config-a
      volumes:
        - name: volumeOne
          configMap:
            name: config-b`)

	md := MetaDeploymentFromDeployment(d)
	expected := []string{
		"configmap/test-namespace/config-a",
		"configmap/test-namespace/config-b",
		"configmap/test-namespace/config-c",
		"secret/test-namespace/secret-a",
		"secret/test-namespace/secret-b",
	}

	equals(t, md.ReferencedConfigs(), expected)
	equals(t, md.ReferencedConfigs(), expected)
}

func TestMetaDeploymentUpdateConfigChecksumsPatchesDeploymentAnnotation(t *testing.T) {
	c := test.NewDummyK8sClient()
	d := newDeploymentFromYAML(`
---
metadata:
  name: test-name
  namespace: test-namespace
`)
	checksums := map[string]string{"config-one": "checksum-one"}

	md := MetaDeploymentFromDeployment(d)
	err := md.UpdateConfigChecksums(c, checksums, false)

	expectedPatchData := map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]interface{}{
				"com.xing.deployment-restart.applied-config-checksums": "{\"config-one\":\"checksum-one\"}",
			},
		},
	}

	equals(t, err, nil)
	equals(t, c.Patches[0].Path, "deployment/test-namespace/test-name")
	equals(t, c.Patches[0].Data, expectedPatchData)
}

func TestMetaDeploymentUpdateConfigChecksumsPatchesDeploymentAnnotationAndTemplateAnnotationWhenRestarting(t *testing.T) {
	c := test.NewDummyK8sClient()
	d := newStatefulSetFromYAML(`
---
metadata:
  name: test-name
  namespace: test-namespace
`)

	now := time.Now()
	currentTimestamp := now.Format(time.Stamp)
	nextSecondTimestamp := now.Add(time.Duration(-1) * time.Second).Format(time.Stamp)

	md := MetaDeploymentFromStatefulSet(d)
	checksums := map[string]string{"config-one": "checksum-one"}

	err := md.UpdateConfigChecksums(c, checksums, true)

	patchDataAsJSON, _ := json.Marshal(c.Patches[0].Data) // mocking time: DO NOT WANT
	usedTimestamp := currentTimestamp
	if strings.Contains(string(patchDataAsJSON), nextSecondTimestamp) {
		usedTimestamp = nextSecondTimestamp
	}

	expectedPatchData := map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]interface{}{
				"com.xing.deployment-restart.applied-config-checksums": "{\"config-one\":\"checksum-one\"}",
			},
		},
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						"com.xing.deployment-restart.timestamp": usedTimestamp,
					},
				},
			},
		},
	}

	equals(t, err, nil)
	equals(t, c.Patches[0].Path, "statefulset/test-namespace/test-name")
	equals(t, c.Patches[0].Data, expectedPatchData)
}

func TestMetaDeploymentUpdateConfigChecksumsReturnsErrorWhenDeploymentTypeIsUnknown(t *testing.T) {
	c := test.NewDummyK8sClient()
	d := newDeploymentFromYAML(`
---
metadata:
  name: test-name
  namespace: test-namespace
`)
	md := &metaDeployment{
		typ:  "whatever",
		meta: d.ObjectMeta,
	}
	checksums := map[string]string{}

	err := md.UpdateConfigChecksums(c, checksums, false)
	equals(t, err.Error(), "Unknown meta deployment type whatever")
	equals(t, len(c.Patches), 0)
}

func TestMetaDeploymentUpdateConfigChecksumsReturnsErrorWhenPatchFails(t *testing.T) {
	c := test.NewDummyK8sClient()
	d := newDeploymentFromYAML(`
---
metadata:
  name: test-name
  namespace: test-namespace
`)
	md := MetaDeploymentFromDeployment(d)
	checksums := map[string]string{}
	c.Error = errors.New("Oh no")

	err := md.UpdateConfigChecksums(c, checksums, false)
	equals(t, err.Error(), "Oh no")
	equals(t, len(c.Patches), 1)
}

func newDeploymentFromYAML(manifest string) (response *apps.Deployment) {
	createFromYAMLManifest(manifest, &response)
	return
}

func newStatefulSetFromYAML(manifest string) (response *apps.StatefulSet) {
	createFromYAMLManifest(manifest, &response)
	return
}

func createFromYAMLManifest(manifest string, result interface{}) {
	var body interface{}
	yaml.Unmarshal([]byte(manifest), &body)
	body = typedKeys(body)
	manifestJSON, _ := json.Marshal(body)
	json.Unmarshal(manifestJSON, result)
}

func typedKeys(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[k.(string)] = typedKeys(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = typedKeys(v)
		}
	}
	return i
}
