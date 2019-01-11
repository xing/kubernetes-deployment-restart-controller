package util

import (
	"testing"
	"runtime"
	"fmt"
	"path/filepath"
	"reflect"
	"errors"
)

func TestPrepareMergePatchDataWithSimpleValue(t *testing.T) {
	path := "such.nested.much.layers"
	value := "wow"

	actual, _ := PrepareMergePatchData(path, value)
	expected := []byte(`{"such":{"nested":{"much":{"layers":"wow"}}}}`)
	if string(actual) != string(expected) {
		t.Errorf("PrepareMergePatchData failed: expected %s, got: %s", expected, actual)
	}
}

func TestPrepareMergePatchDataWithComplexValue(t *testing.T) {
	path := "such.nested"
	value := map[string]interface{}{"much": []string{"layers", "wow"}}

	actual, _ := PrepareMergePatchData(path, value)
	expected := []byte(`{"such":{"nested":{"much":["layers","wow"]}}}`)
	if string(actual) != string(expected) {
		t.Errorf("PrepareMergePatchData failed: expected %s, got: %s", expected, actual)
	}
}

func TestPrepareUpdateMapWithStringValue(t *testing.T) {
	expected := map[string]interface{}{
		"top1": map[string]interface{}{
			"key1": "simple string 1",
		},
	}

	updates := map[string]interface{}{}
	_, exists := updates["top1"]
	equals(t, exists, false)

	path := "top1"
	key := "key1"
	valueString := "simple string 1"
	err := PrepareUpdateMap(updates, path, key, valueString)
	ok(t, err)
	value, exists := updates["top1"]

	nestedMap, ok := value.(map[string]interface{})
	equals(t, ok, true)
	equals(t, exists, true)

	value, ok = nestedMap["key1"]
	equals(t, ok, true)

	equals(t, value, valueString)
	equals(t, updates, expected)
}

func TestPrepareUpdateMapWithMapValue(t *testing.T) {
	expected := map[string]interface{}{
		"top1": map[string]interface{}{
			"key1": map[string]interface{}{
				"nestedKey": "nestedValue",
			},
		},
	}

	updates := map[string]interface{}{}

	_, exists := updates["top1"]
	equals(t, exists, false)

	path := "top1"
	key := "key1"
	valueMap := map[string]interface{}{"nestedKey": "nestedValue",}
	err := PrepareUpdateMap(updates, path, key, valueMap)
	ok(t, err)
	value, exists := updates["top1"]

	nestedMap, ok := value.(map[string]interface{})
	equals(t, ok, true)
	equals(t, exists, true)

	value, ok = nestedMap["key1"]
	equals(t, ok, true)
	equals(t, value, valueMap)

	equals(t, value, valueMap)
	equals(t, updates, expected)
}

func TestPrepareUpdateMap(t *testing.T) {
	expected := map[string]interface{}{
		"top1": map[string]interface{}{
			"attr1": map[string]interface{}{
				"key1": "simple string 1",
			},
			"attr2": map[string]interface{}{
				"key2": map[string]interface{}{
					"nestedKey": "nestedValue",
				},
			},
		},
		"top2": map[string]interface{}{
			"attr1": map[string]interface{}{
				"key3": "simple string 2",
			},
		},
	}
	updates := map[string]interface{}{}
	path := "top1.attr1"
	key := "key1"
	valueString := "simple string 1"
	err := PrepareUpdateMap(updates, path, key, valueString)
	ok(t, err)

	path = "top1.attr2"
	key = "key2"
	valueMap := map[string]interface{}{"nestedKey": "nestedValue",}
	err = PrepareUpdateMap(updates, path, key, valueMap)
	ok(t, err)

	path = "top2.attr1"
	key = "key3"
	valueString = "simple string 2"
	err = PrepareUpdateMap(updates, path, key, valueString)
	ok(t, err)

	equals(t, expected, updates)
}

func TestChangeValueTypeInUpdateMap(t *testing.T) {
	updates := map[string]interface{}{}

	path := "top1.attr1"
	key := "key1"
	valueString := "simple string 1"
	err := PrepareUpdateMap(updates, path, key, valueString)
	ok(t, err)

	path = "top1.attr1.key1"
	key = "key2"
	valueMap := map[string]interface{}{"nested": "nestedValue",}
	err = PrepareUpdateMap(updates, path, key, valueMap)
	if err == nil {
		t.FailNow()
	}
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// equals fails the test if exp is not equal to act
func equals(t *testing.T, got, expected interface{}, args ...interface{}) (err error) {
	t.Helper()
	if !reflect.DeepEqual(got, expected) {
		err = fmt.Errorf("\nExpected: %#v\nGot:      %#v", expected, got)
		if len(args) > 0 {
			err = errors.New(err.Error() + "\n" + fmt.Sprintf("%s",args))
		}
		t.Error(err)
	}
	return
}
