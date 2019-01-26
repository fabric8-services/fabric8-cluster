package repository_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-common/resource"

	"github.com/fabric8-services/fabric8-cluster/test"
	"github.com/fabric8-services/fabric8-common/errors"

	"github.com/fabric8-services/fabric8-cluster/cluster/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertAPI(t *testing.T) {

	resource.Require(t, resource.UnitTest)

	t.Run("ok", func(t *testing.T) {
		t.Run("with path", func(t *testing.T) {
			// when
			result, err := repository.ConvertAPIURL("https://api.domain.com", "sub", "path/")
			// then
			require.NoError(t, err)
			assert.Equal(t, "https://sub.domain.com/path/", result)
		})

		t.Run("without path", func(t *testing.T) {
			// when
			result, err := repository.ConvertAPIURL("https://api.domain.com", "sub", "")
			// then
			require.NoError(t, err)
			assert.Equal(t, "https://sub.domain.com", result)
		})

		t.Run("without subdomain", func(t *testing.T) {
			// when
			result, err := repository.ConvertAPIURL("https://api.domain.com", "", "path")
			// then
			require.NoError(t, err)
			assert.Equal(t, "https://domain.com/path", result)
		})
	})

	t.Run("failures", func(t *testing.T) {

		t.Run("too-short domain", func(t *testing.T) {
			// when
			_, err := repository.ConvertAPIURL("https://domain", "sub", "path")
			// then
			test.AssertError(t, err, errors.BadParameterError{}, "Bad value for parameter 'host': 'domain' (expected: 'must contain more than one domain')")

		})

		t.Run("empty domain", func(t *testing.T) {
			// when
			_, err := repository.ConvertAPIURL("https://", "sub", "path")
			// then
			test.AssertError(t, err, errors.BadParameterError{}, "Bad value for parameter 'host': '' (expected: 'must contain more than one domain')")
		})

		t.Run("invalid URL", func(t *testing.T) {
			// when
			_, err := repository.ConvertAPIURL("%", "sub", "path")
			// then
			test.AssertError(t, err, errors.BadParameterError{}, "parse %: invalid URL escape \"%\"")
		})

	})
}
