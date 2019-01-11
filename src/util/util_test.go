package util

import (
	"testing"
)

func sameKeyMap(a, b KeyMap) bool {
	if len(a) != len(b) {
		return false
	}
	for k, _ := range a {
		if a[k] != b[k] {
			return false
		}
	}
	return true
}

func TestConvertToKeyMap(t *testing.T) {
	raw := RawKeyMap{"v1": "YQ==", "v2": "Yg=="}
	actual := raw.convertToKeyMap()
	expected := KeyMap{1: "a", 2: "b"}
	if !sameKeyMap(expected, actual) {
		t.Errorf("Key conversion failed: expected %v, got: %v", expected, actual)
	}
}

func TestConvertRawDataToKeyMap(t *testing.T) {
	// happy path
	data := []byte(`{ "version": "1", "keys": { "v1": "YQ==", "v2": "Yg==" } }`)
	actual := ConvertRawDataToKeyMap(data)
	expected := KeyMap{1: "a", 2: "b"}
	if !sameKeyMap(expected, actual) {
		t.Errorf("Key conversion failed: expected %v, got: %v", expected, actual)
	}
	// wrong version
	data = []byte(`{"version": 0, "keys": { "v1": "YQ==", "v2": "Yg==" } }`)
	expected = nil
	actual = ConvertRawDataToKeyMap(data)
	if !sameKeyMap(expected, actual) {
		t.Errorf("Key conversion failed: expected %v, got: %v", expected, actual)
	}
	// invalid keys
	data = []byte(`{"version": "1", "keys": { "v": "YQ==" } }`)
	expected = nil
	actual = ConvertRawDataToKeyMap(data)
	if !sameKeyMap(expected, actual) {
		t.Errorf("Key conversion failed: expected %v, got: %v", expected, actual)
	}
}

func TestGetIngressNodeInfo(t *testing.T) {
	nodeName, nodeEnv, nodeDc, err := GetNodeInfo("ingress-external-2.kubernetes.ams1.xing.com")
	if err != nil {
		t.Errorf("Could not get node info: %v", err)
	}
	assertEqual(t, nodeName, "ingress-external-2", "nodeName is wrong")
	assertEqual(t, nodeDc, "ams1", "nodeDc is wrong")
	assertEqual(t, nodeEnv, "production", "nodeEnv is wrong")
}

func TestGetPreviewIngressNodeInfo(t *testing.T) {
	nodeName, nodeEnv, nodeDc, err := GetNodeInfo("ingress-external-2.kubernetes.preview.ams1.xing.com")
	if err != nil {
		t.Errorf("Could not get node info: %v", err)
	}
	assertEqual(t, nodeName, "ingress-external-2", "nodeName is wrong")
	assertEqual(t, nodeDc, "ams1", "nodeDc is wrong")
	assertEqual(t, nodeEnv, "preview", "nodeEnv is wrong")
}

func TestGetNodeInfo(t *testing.T) {
	nodeName, nodeEnv, nodeDc, err := GetNodeInfo("node-2.kubernetes.ams1.xing.com")
	if err != nil {
		t.Errorf("Could not get node info: %v", err)
	}
	assertEqual(t, nodeName, "node-2", "nodeName is wrong")
	assertEqual(t, nodeDc, "ams1", "nodeDc is wrong")
	assertEqual(t, nodeEnv, "production", "nodeEnv is wrong")
}

func assertEqual(t *testing.T, a interface{}, b interface{}, message string) {
	if a == b {
		return
	}
	t.Fatalf("%s, %v != %v", message, a, b)
}
