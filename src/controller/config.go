package controller

import (
	"github.com/xing/kubernetes-deployment-restart-controller/src/controller/interfaces"
)

// Config represents a config instance and a map of deployments referencing it
type Config struct {
	checksum    string
	Deployments map[string]*Deployment
}

// NewPendingConfig returns a pending config with empty deployments map
func NewPendingConfig() *Config {
	return &Config{
		Deployments: make(map[string]*Deployment),
	}
}

// NewConfig returns a Config with an initialized checksum and empty deployments map
func NewConfig(meta interfaces.MetaConfig) *Config {
	return &Config{
		checksum:    meta.Checksum(),
		Deployments: make(map[string]*Deployment),
	}
}

// Checksum returns the checksum or an empty string if the checksum is not set
func (c *Config) Checksum() string {
	return c.checksum
}

// Pending returns true if this config's checksum is unknown
func (c *Config) Pending() bool {
	return c.checksum == ""
}

// UpdateFromMeta copies the checksum from a given MetaConfig. Returns true if the new
// checksum if different from the old one
func (c *Config) UpdateFromMeta(meta interfaces.MetaConfig) bool {
	oldChecksum := c.checksum
	c.checksum = meta.Checksum()
	return c.checksum != oldChecksum
}

// Unused returns true if the config is not used by any deployment and does not have
// a checksum
func (c *Config) Unused() bool {
	return c.Pending() && len(c.Deployments) == 0
}
