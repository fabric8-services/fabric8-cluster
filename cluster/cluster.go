package cluster

import (
	"fmt"
	"strings"
)

const (
	// OSD the OpenShift Dedicated type of cluster
	OSD = "OSD"
	// OCP the OpenShift On-Premise type of Cluster
	OCP = "OCP"
	// OSO the OpenShoft online type of cluster
	OSO = "OSO"
)

// Validate checks that the given value matches a cluster type.
func Validate(v *string) error {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil
	}
	if *v == OSD || *v == OCP || *v == OSO {
		return nil
	}
	return fmt.Errorf("invalid value '%s'. Expected '%v', '%v' or '%v'", *v, OSD, OCP, OSO)
}
