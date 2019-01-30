package cluster_test

import (
	"fmt"
	"testing"

	"github.com/fabric8-services/fabric8-common/resource"

	"github.com/stretchr/testify/assert"

	"github.com/fabric8-services/fabric8-cluster/cluster"
)

func TestValidate(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	// prepare dataset for sub-tests
	osd := "OSD"
	ocp := "OCP"
	oso := "OSO"
	other := "other"
	blank := "  "
	data := []struct {
		name     string
		value    *string
		expected error
	}{
		{"nil", nil, nil},
		{"blank", &blank, nil},
		{"OSD", &osd, nil},
		{"OCP", &ocp, nil},
		{"OSO", &oso, nil},
		{"other", &other, fmt.Errorf("invalid value '%s'. Expected '%v', '%v' or '%v'", "other", cluster.OSD, cluster.OCP, cluster.OSO)},
	}
	// run tests
	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			// when
			err := cluster.Validate(d.value)
			// then
			assert.Equal(t, d.expected, err)
		})
	}

}
