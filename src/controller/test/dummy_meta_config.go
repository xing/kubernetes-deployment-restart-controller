package test

type DummyMetaConfig struct {
	FullNameValue string
	VersionValue  string
	ChecksumValue string
}

// NewDummyK8sClient returns a dummy implementation
func NewDummyMetaConfig(checksum string) *DummyMetaConfig {
	return &DummyMetaConfig{
		ChecksumValue: checksum,
	}
}

// NewDummyMetaConfigWithParams returns a dummy implementation
func NewMetaConfigWithParams(name, version, checksum string) *DummyMetaConfig {
	return &DummyMetaConfig{
		FullNameValue: name,
		VersionValue:  version,
		ChecksumValue: checksum,
	}
}

func (c *DummyMetaConfig) FullName() string { return c.FullNameValue }
func (c *DummyMetaConfig) Version() string  { return c.VersionValue }
func (c *DummyMetaConfig) Checksum() string { return c.ChecksumValue }
