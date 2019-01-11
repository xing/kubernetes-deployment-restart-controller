package interfaces

// ConfigAgent contains the actual implementation of controller logic
type ConfigAgent interface {
	ResourceUpdated(MetaResource)
	ResourceDeleted(MetaResource)
	Start(chan struct{})
	Stop()
}
