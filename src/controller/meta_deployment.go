package controller

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/golang/glog"
	"source.xing.com/olympus/kubernetes-deployment-restart-controller/src/controller/interfaces"
)

const (
	enabledAnnotation                  = "com.xing.deployment-restart"
	configChecksumsAnnotation          = "com.xing.deployment-restart.applied-config-checksums"
	deploymentRestartTriggerAnnotation = "com.xing.deployment-restart.timestamp"

	deploymentTypeDeployment  = "deployment"
	deploymentTypeStatefulSet = "statefulset"
)

type metaDeployment struct {
	typ               string
	meta              metav1.ObjectMeta
	specTemplate      v1.PodTemplateSpec
	referencedConfigs []string
	configChecksums   map[string]string
}

// MetaDeploymentFromDeployment instantiates a meta deployment from a k8s Deployment
func MetaDeploymentFromDeployment(deployment *appsv1.Deployment) interfaces.MetaDeployment {
	return &metaDeployment{
		typ:          deploymentTypeDeployment,
		meta:         deployment.ObjectMeta,
		specTemplate: deployment.Spec.Template,
	}
}

// MetaDeploymentFromStatefulSet instantiates a meta deployment from a k8s StatefulSet
func MetaDeploymentFromStatefulSet(statefulSet *appsv1.StatefulSet) interfaces.MetaDeployment {
	return &metaDeployment{
		typ:          deploymentTypeStatefulSet,
		meta:         statefulSet.ObjectMeta,
		specTemplate: statefulSet.Spec.Template,
	}
}

func (d *metaDeployment) Version() string  { return d.meta.ResourceVersion }
func (d *metaDeployment) FullName() string { return FullName(d.typ, d.meta.Namespace, d.meta.Name) }

// ReferencedConfigs returns a list of full names of all config-like objects referenced in
// the deployment pod spec
func (d *metaDeployment) ReferencedConfigs() []string {
	if d.referencedConfigs == nil {
		d.referencedConfigs = configNamesFromTemplate(d.specTemplate, d.meta)
	}
	return d.referencedConfigs
}

// AppliedChecksums returns parsed config checksum annotation value
func (d *metaDeployment) AppliedChecksums() map[string]string {
	if d.configChecksums == nil {
		d.configChecksums = configChecksumsFromMeta(d.meta)
	}
	return d.configChecksums
}

// NeedsRestartOnConfigChange returns true if the deployment is configured to be restarted
// when any of its configuration resources is changed
func (d *metaDeployment) NeedsRestartOnConfigChange() bool {
	value, ok := d.meta.Annotations[enabledAnnotation]
	return ok && value == "enabled"
}

// UpdateConfigChecksums patches the underlying k8s object with given checksum annotations
// and optionally triggers a restart by changing a template annotation
func (d *metaDeployment) UpdateConfigChecksums(c interfaces.K8sClient, checksums map[string]string, restart bool) error {
	encodedChecksums, _ := json.Marshal(checksums) // checksums is always a map[string]string

	patchData := map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]interface{}{
				configChecksumsAnnotation: string(encodedChecksums),
			},
		},
	}

	if restart {
		patchData["spec"] = map[string]interface{}{
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						deploymentRestartTriggerAnnotation: time.Now().Format(time.Stamp),
					},
				},
			},
		}
	}

	switch d.typ {
	case deploymentTypeDeployment:
		return c.PatchDeployment(d.meta.Namespace, d.meta.Name, patchData)
	case deploymentTypeStatefulSet:
		return c.PatchStatefulSet(d.meta.Namespace, d.meta.Name, patchData)
	}

	return fmt.Errorf("Unknown meta deployment type %s", d.typ)
}

func configNamesFromTemplate(templateSpec v1.PodTemplateSpec, meta metav1.ObjectMeta) []string {
	var configs []string
	namespace := meta.Namespace

	for _, container := range templateSpec.Spec.Containers {
		for _, envFromSource := range container.EnvFrom {
			switch {
			case envFromSource.ConfigMapRef != nil:
				name := envFromSource.ConfigMapRef.LocalObjectReference.Name
				configs = append(configs, FullName(configTypeConfigMap, namespace, name))
			case envFromSource.SecretRef != nil:
				name := envFromSource.SecretRef.LocalObjectReference.Name
				configs = append(configs, FullName(configTypeSecret, namespace, name))
			}
		}
	}

	for _, volume := range templateSpec.Spec.Volumes {
		if cm := volume.ConfigMap; cm != nil {
			configs = append(configs, FullName(configTypeConfigMap, namespace, cm.Name))
		}
	}

	sort.Strings(configs)

	return configs
}

func configChecksumsFromMeta(meta metav1.ObjectMeta) map[string]string {
	value, ok := meta.Annotations[configChecksumsAnnotation]
	if !ok {
		glog.V(3).Infof("Config checksums annotation for %s/%s not found. Assuming an empty map", meta.Namespace, meta.Name)
		return make(map[string]string)
	}

	var checksums map[string]string
	err := json.Unmarshal([]byte(value), &checksums)
	if err != nil {
		glog.Warningf("Failed to unmarshal config checksums annotation for %s/%s: %s. Assuming an empty map", meta.Namespace, meta.Name, err)
		return make(map[string]string)
	}

	return checksums
}
