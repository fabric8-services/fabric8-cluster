package service

import (
	"testing"

	"github.com/fabric8-services/fabric8-common/resource"
	"github.com/stretchr/testify/require"
)

// TestValidateURL verifies the `cluster.Validate()` func
func TestValidateURL(t *testing.T) {

	resource.Require(t, resource.UnitTest)

	t.Run("valid", func(t *testing.T) {
		// when
		err := validateURL("http://foo.com")
		// then
		require.NoError(t, err)
	})

	t.Run("invalid", func(t *testing.T) {

		t.Run("missing scheme", func(t *testing.T) {
			// when
			err := validateURL("foo.com")
			// then
			require.NoError(t, err)
		})

		t.Run("missing host", func(t *testing.T) {
			// when
			err := validateURL("http://")
			// then
			require.NoError(t, err)
		})

		t.Run("invalid host", func(t *testing.T) {
			// when
			err := validateURL("http://%")
			// then
			require.NoError(t, err)
		})
	})
}
