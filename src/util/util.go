package util

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

// Verbose logging
var Verbose = false

// The name of the password file is hard wired, to be able to remove
// it before doing anything which might crash the program.  Note that
// the name must contain the string 'secret'.
const (
	masterKeyFile  = "olympus_master_secrets"
	nodeNameRegExp = "^([a-z-]+-\\d+)\\.kubernetes\\.(?:(\\w+)\\.)?(\\w{3}[12])\\.xing\\.com"
)

type warnings int

const (
	logWarnings      warnings = 1
	suppressWarnings warnings = 2
)

type KeyMap map[int]string
type RawKeyMap map[string]string
type KeyFileStruct struct {
	Version string    `json:"version"`
	Keys    RawKeyMap `json:"keys"`
}

// Convert a raw keymap with string keys to a key map with int keys
func (r *RawKeyMap) convertToKeyMap() KeyMap {
	m := make(KeyMap)
	for k, v := range *r {
		if d, err := strconv.Atoi(k[1:]); err != nil {
			log.Printf("wrong key format: %s: should be 'v\\d+'\n", k)
		} else {
			if v, err := base64.StdEncoding.DecodeString(v); err != nil {
				log.Printf("could not decode value for key %s, error=%s\n", k, err)
			} else {
				m[d] = string(v)
			}
		}
	}
	return m
}

// Convert file content to a key map
func ConvertRawDataToKeyMap(bytes []byte) KeyMap {
	var data KeyFileStruct
	if err := json.Unmarshal(bytes, &data); err != nil {
		log.Printf("could not parse JSON from key file, error=%s\n", err)
		return nil
	}
	if data.Version != "1" {
		log.Printf("wrong key file version %s: should be '1'\n", data.Version)
		return nil
	}
	return data.Keys.convertToKeyMap()
}

// Called as one of the first steps of agent/server programs (before
// it can crash). When running under Mesos, it removes the password
// file from the Mesos sandbox.
func GetMasterKeysAndRemoveKeyFile() KeyMap {
	bytes := readAndRemoveSecretsFile(masterKeyFile, logWarnings)
	if len(bytes) == 0 {
		return KeyMap{}
	} else {
		return ConvertRawDataToKeyMap(bytes)
	}
}

// Read a file and return its contents. Warnings can be supressed.
func readAndRemoveSecretsFile(filename string, w warnings) []byte {
	sandbox := os.Getenv("MESOS_SANDBOX")
	if sandbox == "" {
		return nil
	}
	filename = filepath.Join(sandbox, filename)
	warn := w == logWarnings
	bytes, err := ioutil.ReadFile(filename)
	if err != nil && warn {
		log.Printf("could not read secrets file(%s), error = %s\n", filename, err)
	}
	if err := os.Remove(filename); err != nil {
		if warn {
			log.Printf("could not remove secrets file(%s), error = %s\n", filename, err)
		}
	} else if Verbose {
		log.Printf("removed secrets file: %s\n", filename)
	}
	return bytes
}

// GetNodeInfo returns the node name, environment and datacenter from a fqdn
func GetNodeInfo(name string) (string, string, string, error) {
	nodeRegexp := regexp.MustCompile(nodeNameRegExp)
	match := nodeRegexp.FindAllStringSubmatch(name, -1)

	if len(match) > 0 {
		if len(match[0][2]) > 0 {
			return match[0][1], match[0][2], match[0][3], nil
		}
		return match[0][1], "production", match[0][3], nil
	}

	return "", "", "", errors.New("could not determine node info")
}
