package controller

import (
	"testing"
	"time"
)

func TestNewChangeReturnsAChangeWithOneObservationAndPositiveAge(t *testing.T) {
	c := NewChange()

	equals(t, c.Observations, 1)
	equals(t, c.Age() > time.Duration(0), true)
}

func TestChangeAgeIncreasesOverTime(t *testing.T) {
	c := NewChange()
	age := c.Age()
	equals(t, c.Age() > age, true)
}
