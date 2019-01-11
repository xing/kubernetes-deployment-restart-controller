package util

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/golang/glog"

	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc" // needed for local development with .kube/config
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"fmt"
)

// Clientset abstracts the cluster config loading both locally and on Kubernetes
func Clientset() kubernetes.Interface {
	// Try to load in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		glog.V(3).Infof("Could not load in-cluster config: %v", err)

		// Fall back to local config
		config, err = clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")
		if err != nil {
			glog.Fatalf("Failed to load client config: %v", err)
		}
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Failed to create kubernetes client: %v", err)
	}

	return client
}

// PrepareMergePatchData generates JSON merge patch payload to be used in k8s Patch methods
func PrepareMergePatchData(path string, value interface{}) (data []byte, err error) {
	patch := value
	a := strings.Split(path, ".")
	for i := len(a) - 1; i >= 0; i-- {
		patch = map[string]interface{}{a[i]: patch}
	}
	data, err = json.Marshal(patch)
	return
}

// PrepareUpdateMap alters the map of updates by adding the key/value in the given path
func PrepareUpdateMap(updates map[string]interface{}, path string, key string, value interface{}) error {
	path_parts := strings.Split(path, ".")

	for i := 0; i < len(path_parts); i++ {
		existing, ok := updates[path_parts[i]]
		if ok {
			existing_map, ok := existing.(map[string]interface{})
			if !ok {
				return fmt.Errorf("cannot prepare an update: %s, %s: %s already set to an incompatible type", path, key, existing)
			}
			updates = existing_map
		} else {
			next := map[string]interface{}{}
			updates[path_parts[i]] = next
			updates = next
		}
	}

	updates[key] = value
	return nil
}
