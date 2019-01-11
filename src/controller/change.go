package controller

import (
	"time"
)

// Change stores a timestamp when the change was observed and the number of observations
type Change struct {
	createdAt    time.Time
	Observations int
}

// NewChange returns a new change instance
func NewChange() *Change {
	return &Change{
		createdAt:    time.Now(),
		Observations: 1,
	}
}

// Age returns the duration since when the change was instantiated
func (c *Change) Age() time.Duration {
	return time.Now().Sub(c.createdAt)
}
