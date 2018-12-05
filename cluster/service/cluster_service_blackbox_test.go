package service_test

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-cluster/cluster"
	"github.com/fabric8-services/fabric8-cluster/cluster/repository"
	"github.com/fabric8-services/fabric8-cluster/configuration"
	"github.com/fabric8-services/fabric8-cluster/gormapplication"
	"github.com/fabric8-services/fabric8-cluster/gormtestsupport"
	"github.com/fabric8-services/fabric8-cluster/test"
	"github.com/fabric8-services/fabric8-common/errors"

	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestClusterService(t *testing.T) {
	suite.Run(t, &ClusterServiceTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

type ClusterServiceTestSuite struct {
	gormtestsupport.DBTestSuite
}

func (s *ClusterServiceTestSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
}

func (s *ClusterServiceTestSuite) TestCreateOrSaveClusterFromConfigOK() {
	// when
	err := s.Application.ClusterService().CreateOrSaveClusterFromConfig(context.Background())
	// then
	require.NoError(s.T(), err)

	osoClusters, err := s.Application.Clusters().Query(func(db *gorm.DB) *gorm.DB {
		return db.Where("type = ?", cluster.OSO)
	})
	require.NoError(s.T(), err)
	assert.Len(s.T(), osoClusters, 3)

	osdClusters, err := s.Application.Clusters().Query(func(db *gorm.DB) *gorm.DB {
		return db.Where("type = ?", cluster.OSD)
	})
	require.NoError(s.T(), err)
	assert.Len(s.T(), osdClusters, 1)

	verifyClusters(s.T(), append(osoClusters, osdClusters...), s.Configuration.GetClusters())
}

func (s *ClusterServiceTestSuite) TestCreateOrSaveCluster() {

	s.T().Run("valid", func(t *testing.T) {

		t.Run("valid with missing URLs", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.ConsoleURL = " "
			c.LoggingURL = " "
			c.MetricsURL = " "
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), &c)
			// then
			require.NoError(t, err)
			assert.NotNil(t, c.ClusterID) // then
			assert.Equal(t, "https://console.cluster-foo.com/console", c.ConsoleURL)
			assert.Equal(t, "https://metrics.cluster-foo.com", c.MetricsURL)
			assert.Equal(t, "https://console.cluster-foo.com/console", c.LoggingURL)
		})

		t.Run("valid with all URLs", func(t *testing.T) {
			// given
			c := newTestCluster()
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), &c)
			// then
			require.NoError(t, err)
			assert.NotNil(t, c.ClusterID) // then
		})
	})

	s.T().Run("invalid", func(t *testing.T) {

		t.Run("empty name", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.Name = " "
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), &c)
			// then
			require.Error(t, err)
			err = errs.Cause(err)
			require.IsType(t, errors.BadParameterError{}, err)
			assert.Equal(t, "empty field 'name' is not allowed", err.(errors.BadParameterError).Error())
		})

		t.Run("empty service-account-token", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.SAToken = " "
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), &c)
			// then
			require.Error(t, err)
			err = errs.Cause(err)
			require.IsType(t, errors.BadParameterError{}, err)
			assert.Equal(t, "empty field 'service-account-token' is not allowed", err.(errors.BadParameterError).Error())
		})

		t.Run("empty service-account-username", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.SAUsername = " "
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), &c)
			// then
			require.Error(t, err)
			err = errs.Cause(err)
			require.IsType(t, errors.BadParameterError{}, err)
			assert.Equal(t, "empty field 'service-account-username' is not allowed", err.(errors.BadParameterError).Error())
		})

		t.Run("token-provider-id", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.TokenProviderID = " "
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), &c)
			// then
			require.Error(t, err)
			err = errs.Cause(err)
			require.IsType(t, errors.BadParameterError{}, err)
			assert.Equal(t, "empty field 'token-provider-id' is not allowed", err.(errors.BadParameterError).Error())
		})

		t.Run("auth-client-id", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.AuthClientID = " "
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), &c)
			// then
			require.Error(t, err)
			err = errs.Cause(err)
			require.IsType(t, errors.BadParameterError{}, err)
			assert.Equal(t, "empty field 'auth-client-id' is not allowed", err.(errors.BadParameterError).Error())
		})

		t.Run("token-provider-id", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.AuthClientSecret = " "
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), &c)
			// then
			require.Error(t, err)
			err = errs.Cause(err)
			require.IsType(t, errors.BadParameterError{}, err)
			assert.Equal(t, "empty field 'auth-client-secret' is not allowed", err.(errors.BadParameterError).Error())
		})

		t.Run("auth-client-default-scope", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.AuthDefaultScope = " "
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), &c)
			// then
			require.Error(t, err)
			err = errs.Cause(err)
			require.IsType(t, errors.BadParameterError{}, err)
			assert.Equal(t, "empty field 'auth-client-default-scope' is not allowed", err.(errors.BadParameterError).Error())
		})

		t.Run("invalid API URL", func(t *testing.T) {

			t.Run("empty", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.URL = " "
				// when
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), &c)
				// then
				require.Error(t, err)
				err = errs.Cause(err)
				require.IsType(t, errors.BadParameterError{}, err)
				assert.Equal(t, "'API' URL ' ' is invalid: missing scheme or host", err.(errors.BadParameterError).Error())
			})

			t.Run("missing scheme", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.URL = "api.cluster.com"
				// when
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), &c)
				// then
				require.Error(t, err)
				err = errs.Cause(err)
				require.IsType(t, errors.BadParameterError{}, err)
				assert.Equal(t, "'API' URL 'api.cluster.com' is invalid: missing scheme or host", err.(errors.BadParameterError).Error())
			})

			t.Run("missing host", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.URL = "https://"
				// when
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), &c)
				// then
				require.Error(t, err)
				err = errs.Cause(err)
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
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), &c)
				// then
				require.Error(t, err)
				err = errs.Cause(err)
				require.IsType(t, errors.BadParameterError{}, err)
				assert.Equal(t, "'console' URL 'console.cluster-foo.com' is invalid: missing scheme or host", err.(errors.BadParameterError).Error())
			})

			t.Run("missing host", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.ConsoleURL = "https://"
				// when
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), &c)
				// then
				require.Error(t, err)
				err = errs.Cause(err)
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
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), &c)
				// then
				require.Error(t, err)
				err = errs.Cause(err)
				require.IsType(t, errors.BadParameterError{}, err)
				assert.Equal(t, "'logging' URL 'logging.cluster-foo.com' is invalid: missing scheme or host", err.(errors.BadParameterError).Error())
			})

			t.Run("missing host", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.LoggingURL = "https://"
				// when
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), &c)
				// then
				require.Error(t, err)
				err = errs.Cause(err)
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
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), &c)
				// then
				require.Error(t, err)
				err = errs.Cause(err)
				require.IsType(t, errors.BadParameterError{}, err)
				assert.Equal(t, "'metrics' URL 'metrics.cluster-foo.com' is invalid: missing scheme or host", err.(errors.BadParameterError).Error())
			})

			t.Run("missing host", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.MetricsURL = "https://"
				// when
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), &c)
				// then
				require.Error(t, err)
				err = errs.Cause(err)
				require.IsType(t, errors.BadParameterError{}, err)
				assert.Equal(t, "'metrics' URL 'https://' is invalid: missing scheme or host", err.(errors.BadParameterError).Error())
			})

		})

		t.Run("invalid type", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.Type = "FOO"
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), &c)
			// then
			require.Error(t, err)
			err = errs.Cause(err)
			require.IsType(t, errors.BadParameterError{}, err)
			assert.Equal(t, "invalid type of cluster: 'FOO' (expected 'OSO', 'OCP' or 'OSD')", err.(errors.BadParameterError).Error())

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

func (s *ClusterServiceTestSuite) TestClusterConfigurationWatcher() {
	t := s.T()
	// Create a temp file with content from ./conf-files/oso-clusters-custom.conf
	tmpFileName := createTempClusterConfigFile(t)
	defer os.Remove(tmpFileName)

	// Load configuration from the temp file
	config, err := configuration.NewConfigurationData("", tmpFileName)
	require.NoError(t, err)
	c := config.GetClusterByURL("https://api.starter-us-east-2a.openshift.com")
	require.NotNil(t, c)

	original := c.CapacityExhausted

	// initialize application with new config
	application := gormapplication.NewGormDB(s.DB, config)
	// Start watching
	haltWatcher, err := application.ClusterService().InitializeClusterWatcher()
	require.NoError(t, err)
	defer haltWatcher()

	// Update the config file
	updateClusterConfigFile(t, tmpFileName, "./configuration/conf-files/tests/oso-clusters-capacity-updated.conf")
	// Check if it has been updated
	waitForConfigUpdate(t, config, !original)

	// Update the config file back to the original
	updateClusterConfigFile(t, tmpFileName, "./configuration/conf-files/oso-clusters.conf")
	// Check if it has been updated
	waitForConfigUpdate(t, config, original)

	// Update the config file to some invalid data
	updateClusterConfigFile(t, tmpFileName, "./configuration/conf-files/tests/oso-clusters-invalid.conf")
	// The configuration should not change
	waitForConfigUpdate(t, config, original)
	updateClusterConfigFile(t, tmpFileName, "./configuration/conf-files/tests/oso-clusters-empty.conf")
	// The configuration should not change
	waitForConfigUpdate(t, config, original)

	// Fix the file
	updateClusterConfigFile(t, tmpFileName, "./configuration/conf-files/tests/oso-clusters-capacity-updated.conf")
	// Now configuration should be updated
	waitForConfigUpdate(t, config, !original)
}

func (s *ClusterServiceTestSuite) TestClusterConfigurationWatcherNoErrorForDefaultConfig() {
	s.Application = gormapplication.NewGormDB(s.DB, s.Configuration)
	haltWatcher, err := s.Application.ClusterService().InitializeClusterWatcher()
	require.NoError(s.T(), err)
	defer haltWatcher()
}

func createTempClusterConfigFile(t *testing.T) string {
	to, err := ioutil.TempFile("", "oso-clusters.conf")
	require.NoError(t, err)
	defer to.Close()

	from, err := os.Open("./../../configuration/conf-files/oso-clusters.conf")
	require.NoError(t, err)
	defer from.Close()

	_, err = io.Copy(to, from)
	require.NoError(t, err)
	return to.Name()
}

func updateClusterConfigFile(t *testing.T, to, from string) {
	fromFile, err := os.Open("./../../" + from)
	require.NoError(t, err)
	defer fromFile.Close()

	toFile, err := os.OpenFile(to, os.O_RDWR|os.O_CREATE, 0666)
	require.NoError(t, err)
	defer toFile.Close()

	_, err = io.Copy(toFile, fromFile)
	require.NoError(t, err)
}

func waitForConfigUpdate(t *testing.T, config *configuration.ConfigurationData, expected bool) {
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		c := config.GetClusterByURL("https://api.starter-us-east-2a.openshift.com")
		require.NotNil(t, c)

		// verify that cluster type set to OSO in case of not present in config
		require.Equal(t, cluster.OSO, c.Type)
		if expected == c.CapacityExhausted {
			return
		}
	}
	require.Fail(t, "cluster config has not been reloaded within 3s")
}

func verifyClusters(t *testing.T, clusters []repository.Cluster, configClusters map[string]configuration.Cluster) {
	for _, configCluster := range configClusters {
		verifyCluster(t, clusters, test.ClusterFromConfigurationCluster(configCluster))
	}
}

func verifyCluster(t *testing.T, clusters []repository.Cluster, expected *repository.Cluster) {
	actual := test.FilterClusterByURL(expected.URL, clusters)
	test.AssertEqualClusterDetails(t, expected, actual)
}
