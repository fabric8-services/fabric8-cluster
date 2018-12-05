package service

import (
	"fmt"
	"testing"

	"github.com/fabric8-services/fabric8-cluster/cluster"

	"github.com/fabric8-services/fabric8-common/errors"

	"github.com/fabric8-services/fabric8-common/resource"

	"github.com/stretchr/testify/assert"

	"github.com/fabric8-services/fabric8-cluster/cluster/repository"
	"github.com/stretchr/testify/require"
)

func TestValidation(t *testing.T) {

	resource.Require(t, resource.UnitTest)

	t.Run("valid", func(t *testing.T) {

		t.Run("valid with missing URLs", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.ConsoleURL = ""
			c.LoggingURL = ""
			c.MetricsURL = ""
			// when
			err := validate(&c)
			// then
			require.NoError(t, err)
			assert.Equal(t, "https://console.cluster-foo.com/console", c.ConsoleURL)
			assert.Equal(t, "https://metrics.cluster-foo.com", c.MetricsURL)
			assert.Equal(t, "https://console.cluster-foo.com/console", c.LoggingURL)
		})

		t.Run("valid with all URLs", func(t *testing.T) {
			// given
			c := newTestCluster()
			// when
			err := validate(&c)
			// then
			require.NoError(t, err)
		})
	})

	t.Run("invalid", func(t *testing.T) {

		t.Run("empty name", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.Name = ""
			// when
			err := validate(&c)
			// then
			require.Error(t, err)
			require.IsType(t, errors.BadParameterError{}, err)
			assert.Equal(t, fmt.Sprintf(errEmptyFieldMsg, "name"), err.(errors.BadParameterError).Error())
		})

		t.Run("empty service-account-token", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.SAToken = ""
			// when
			err := validate(&c)
			// then
			require.Error(t, err)
			require.IsType(t, errors.BadParameterError{}, err)
			assert.Equal(t, fmt.Sprintf(errEmptyFieldMsg, "service-account-token"), err.(errors.BadParameterError).Error())
		})

		t.Run("empty service-account-username", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.SAUsername = ""
			// when
			err := validate(&c)
			// then
			require.Error(t, err)
			require.IsType(t, errors.BadParameterError{}, err)
			assert.Equal(t, fmt.Sprintf(errEmptyFieldMsg, "service-account-username"), err.(errors.BadParameterError).Error())
		})

		t.Run("token-provider-id", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.TokenProviderID = ""
			// when
			err := validate(&c)
			// then
			require.Error(t, err)
			require.IsType(t, errors.BadParameterError{}, err)
			assert.Equal(t, fmt.Sprintf(errEmptyFieldMsg, "token-provider-id"), err.(errors.BadParameterError).Error())
		})

		t.Run("auth-client-id", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.AuthClientID = ""
			// when
			err := validate(&c)
			// then
			require.Error(t, err)
			require.IsType(t, errors.BadParameterError{}, err)
			assert.Equal(t, fmt.Sprintf(errEmptyFieldMsg, "auth-client-id"), err.(errors.BadParameterError).Error())
		})

		t.Run("token-provider-id", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.AuthClientSecret = ""
			// when
			err := validate(&c)
			// then
			require.Error(t, err)
			require.IsType(t, errors.BadParameterError{}, err)
			assert.Equal(t, fmt.Sprintf(errEmptyFieldMsg, "auth-client-secret"), err.(errors.BadParameterError).Error())
		})

		t.Run("auth-client-default-scope", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.AuthDefaultScope = ""
			// when
			err := validate(&c)
			// then
			require.Error(t, err)
			require.IsType(t, errors.BadParameterError{}, err)
			assert.Equal(t, fmt.Sprintf(errEmptyFieldMsg, "auth-client-default-scope"), err.(errors.BadParameterError).Error())
		})

		t.Run("invalid API URL", func(t *testing.T) {

			t.Run("empty", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.URL = ""
				// when
				err := validate(&c)
				// then
				require.Error(t, err)
				require.IsType(t, errors.BadParameterError{}, err)
				assert.Equal(t, "'API' URL '' is invalid: missing scheme or host", err.(errors.BadParameterError).Error())
			})

			t.Run("missing scheme", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.URL = "api.cluster.com"
				// when
				err := validate(&c)
				// then
				require.Error(t, err)
				require.IsType(t, errors.BadParameterError{}, err)
				assert.Equal(t, "'API' URL 'api.cluster.com' is invalid: missing scheme or host", err.(errors.BadParameterError).Error())
			})

			t.Run("missing host", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.URL = "https://"
				// when
				err := validate(&c)
				// then
				require.Error(t, err)
				require.IsType(t, errors.BadParameterError{}, err)
				assert.Equal(t, "'API' URL 'https://' is invalid: missing scheme or host", err.(errors.BadParameterError).Error())
			})
		})

		t.Run("invalid console URL", func(t *testing.T) {

			t.Run("missing scheme", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.ConsoleURL = "console.cluster-foo.com"
				// when
				err := validate(&c)
				// then
				require.Error(t, err)
				require.IsType(t, errors.BadParameterError{}, err)
				assert.Equal(t, "'console' URL 'console.cluster-foo.com' is invalid: missing scheme or host", err.(errors.BadParameterError).Error())
			})

			t.Run("missing host", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.ConsoleURL = "https://"
				// when
				err := validate(&c)
				// then
				require.Error(t, err)
				require.IsType(t, errors.BadParameterError{}, err)
				assert.Equal(t, "'console' URL 'https://' is invalid: missing scheme or host", err.(errors.BadParameterError).Error())
			})

		})

		t.Run("invalid logging URL", func(t *testing.T) {

			t.Run("missing scheme", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.LoggingURL = "logging.cluster-foo.com"
				// when
				err := validate(&c)
				// then
				require.Error(t, err)
				require.IsType(t, errors.BadParameterError{}, err)
				assert.Equal(t, "'logging' URL 'logging.cluster-foo.com' is invalid: missing scheme or host", err.(errors.BadParameterError).Error())
			})

			t.Run("missing host", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.LoggingURL = "https://"
				// when
				err := validate(&c)
				// then
				require.Error(t, err)
				require.IsType(t, errors.BadParameterError{}, err)
				assert.Equal(t, "'logging' URL 'https://' is invalid: missing scheme or host", err.(errors.BadParameterError).Error())
			})

		})

		t.Run("invalid metrics URL", func(t *testing.T) {

			t.Run("missing scheme", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.MetricsURL = "metrics.cluster-foo.com"
				// when
				err := validate(&c)
				// then
				require.Error(t, err)
				require.IsType(t, errors.BadParameterError{}, err)
				assert.Equal(t, "'metrics' URL 'metrics.cluster-foo.com' is invalid: missing scheme or host", err.(errors.BadParameterError).Error())
			})

			t.Run("missing host", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.MetricsURL = "https://"
				// when
				err := validate(&c)
				// then
				require.Error(t, err)
				require.IsType(t, errors.BadParameterError{}, err)
				assert.Equal(t, "'metrics' URL 'https://' is invalid: missing scheme or host", err.(errors.BadParameterError).Error())
			})

		})

		t.Run("invalid type", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.Type = "FOO"
			// when
			err := validate(&c)
			// then
			require.Error(t, err)
			require.IsType(t, errors.BadParameterError{}, err)
			assert.Equal(t, fmt.Sprintf(errInvalidTypeMsg, c.Type), err.(errors.BadParameterError).Error())

		})
	})
}

func newTestCluster() repository.Cluster {
	return repository.Cluster{
		Name:              "foo",
		Type:              cluster.OCP,
		AppDNS:            "https://cluster-foo.com",
		URL:               "https://api.cluster-foo.com",
		ConsoleURL:        "https://console.cluster-foo.com",
		LoggingURL:        "https://logging.cluster-foo.com",
		MetricsURL:        "https://metrics.cluster-foo.com",
		CapacityExhausted: false,
		SAToken:           "ServiceAccountToken",
		SAUsername:        "ServiceAccountUsername",
		TokenProviderID:   "TokenProviderID",
		AuthClientID:      "AuthClientID",
		AuthClientSecret:  "AuthClientSecret",
		AuthDefaultScope:  "AuthClientDefaultScope",
	}
}
