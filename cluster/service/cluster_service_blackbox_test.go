package service_test

import (
	"context"
	"fmt"
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
	testsupport "github.com/fabric8-services/fabric8-common/test"

	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
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

func (s *ClusterServiceTestSuite) TestCreateOrSaveClusterFromEndpoint() {

	s.T().Run("create", func(t *testing.T) {

		t.Run("valid with missing URLs", func(t *testing.T) {
			// given
			c := newTestCluster()
			name := c.Name
			c.ConsoleURL = " "
			c.LoggingURL = " "
			c.MetricsURL = " "
			c.TokenProviderID = " "
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			// then
			require.NoError(t, err)
			assert.NotNil(t, c.ClusterID)
			assert.Equal(t, name, c.Name)
			assert.Equal(t, cluster.OCP, c.Type)
			assert.Equal(t, fmt.Sprintf("https://cluster.%s", name), c.AppDNS)
			assert.Equal(t, fmt.Sprintf("https://api.cluster.%s", name), c.URL)
			assert.Equal(t, false, c.CapacityExhausted)
			assert.Equal(t, "ServiceAccountToken", c.SAToken)
			assert.Equal(t, "ServiceAccountUsername", c.SAUsername)
			assert.Equal(t, "AuthClientID", c.AuthClientID)
			assert.Equal(t, "AuthClientSecret", c.AuthClientSecret)
			assert.Equal(t, "AuthClientDefaultScope", c.AuthDefaultScope)
			// optional fields: generated values
			assert.Equal(t, fmt.Sprintf("https://console.cluster.%s/console", name), c.ConsoleURL)
			assert.Equal(t, fmt.Sprintf("https://metrics.cluster.%s", name), c.MetricsURL)
			assert.Equal(t, fmt.Sprintf("https://console.cluster.%s/console", name), c.LoggingURL)
			assert.Equal(t, c.ClusterID.String(), c.TokenProviderID)
		})

		t.Run("valid with all URLs", func(t *testing.T) {
			// given
			c := newTestCluster()
			name := c.Name
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			// then
			require.NoError(t, err)
			assert.NotNil(t, c.ClusterID)
			assert.Equal(t, name, c.Name)
			assert.Equal(t, cluster.OCP, c.Type)
			assert.Equal(t, fmt.Sprintf("https://cluster.%s", name), c.AppDNS)
			assert.Equal(t, fmt.Sprintf("https://api.cluster.%s", name), c.URL)
			assert.Equal(t, false, c.CapacityExhausted)
			assert.Equal(t, "ServiceAccountToken", c.SAToken)
			assert.Equal(t, "ServiceAccountUsername", c.SAUsername)
			assert.Equal(t, "AuthClientID", c.AuthClientID)
			assert.Equal(t, "AuthClientSecret", c.AuthClientSecret)
			assert.Equal(t, "AuthClientDefaultScope", c.AuthDefaultScope)
			// optional fields: keep provided values
			assert.Equal(t, fmt.Sprintf("https://console.cluster.%s", name), c.ConsoleURL)
			assert.Equal(t, fmt.Sprintf("https://metrics.cluster.%s", name), c.MetricsURL)
			assert.Equal(t, fmt.Sprintf("https://logging.cluster.%s", name), c.LoggingURL)
			assert.Equal(t, "TokenProviderID", c.TokenProviderID)
		})
	})

	s.T().Run("save existing", func(t *testing.T) {

		t.Run("with updated TokenProviderID", func(t *testing.T) {
			// given an existing cluster
			c := newTestCluster()
			require.Equal(t, uuid.Nil, c.ClusterID)
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			require.NoError(t, err)
			t.Logf("created cluster ID: %v", c.ClusterID)
			require.NotEqual(t, uuid.Nil, c.ClusterID)
			// when updating with an updated TokenProviderID value
			c, err = s.Application.Clusters().LoadClusterByURL(context.Background(), c.URL)
			require.NoError(t, err)
			c.TokenProviderID = "UpdatedTokenProviderID"
			err = s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			// then
			require.NoError(t, err)
			// read again from DB
			reloaded, err := s.Application.Clusters().LoadClusterByURL(context.Background(), c.URL)
			require.NoError(t, err)
			assert.Equal(t, c.ClusterID, reloaded.ClusterID)
			assert.Equal(t, "UpdatedTokenProviderID", reloaded.TokenProviderID)
		})

		t.Run("with missing TokenProviderID", func(t *testing.T) {
			// given an existing cluster
			c := newTestCluster()
			require.Equal(t, uuid.Nil, c.ClusterID)
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			require.NoError(t, err)
			t.Logf("created cluster ID: %v", c.ClusterID)
			require.NotEqual(t, uuid.Nil, c.ClusterID)
			// when updating without any TokenProviderID value
			c, err = s.Application.Clusters().LoadClusterByURL(context.Background(), c.URL)
			require.NoError(t, err)
			c.TokenProviderID = ""
			err = s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			// then
			require.NoError(t, err)
			// read again from DB
			reloaded, err := s.Application.Clusters().LoadClusterByURL(context.Background(), c.URL)
			require.NoError(t, err)
			assert.Equal(t, c.ClusterID, reloaded.ClusterID)
			// expect TokenProviderID to be equal to old value
			assert.Equal(t, c.TokenProviderID, reloaded.TokenProviderID)

		})

	})

	s.T().Run("invalid", func(t *testing.T) {

		t.Run("empty name", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.Name = " "
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			// then
			testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': empty field '%s' is not allowed", c.Name, "name"))
		})

		t.Run("empty service-account-token", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.SAToken = " "
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			// then
			testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': empty field '%s' is not allowed", c.Name, "service-account-token"))
		})

		t.Run("empty service-account-username", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.SAUsername = " "
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			// then
			testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': empty field '%s' is not allowed", c.Name, "service-account-username"))
		})

		t.Run("auth-client-id", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.AuthClientID = " "
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			// then
			testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': empty field '%s' is not allowed", c.Name, "auth-client-id"))
		})

		t.Run("token-provider-id", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.AuthClientSecret = " "
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			// then
			testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': empty field '%s' is not allowed", c.Name, "auth-client-secret"))
		})

		t.Run("auth-client-default-scope", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.AuthDefaultScope = " "
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			// then
			testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': empty field '%s' is not allowed", c.Name, "auth-client-default-scope"))
		})

		t.Run("invalid API URL", func(t *testing.T) {

			t.Run("empty", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.URL = " "
				// when
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
				// then
				testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': '%s' URL '%s' is invalid: missing scheme or host", c.Name, "API", c.URL))
			})

			t.Run("missing scheme", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.URL = "api.cluster.com"
				// when
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
				// then
				testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': '%s' URL '%s' is invalid: missing scheme or host", c.Name, "API", c.URL))
			})

			t.Run("missing host", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.URL = "https://"
				// when
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
				// then
				testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': '%s' URL '%s' is invalid: missing scheme or host", c.Name, "API", c.URL))
			})
		})

		t.Run("invalid console URL", func(t *testing.T) {

			t.Run("missing scheme", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.ConsoleURL = "console.cluster-foo.com"
				// when
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
				// then
				testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': '%s' URL '%s' is invalid: missing scheme or host", c.Name, "console", c.ConsoleURL))
			})

			t.Run("missing host", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.ConsoleURL = "https://"
				// when
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
				// then
				testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': '%s' URL '%s' is invalid: missing scheme or host", c.Name, "console", c.ConsoleURL))
			})

		})

		t.Run("invalid logging URL", func(t *testing.T) {

			t.Run("missing scheme", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.LoggingURL = "logging.cluster-foo.com"
				// when
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
				// then
				testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': '%s' URL '%s' is invalid: missing scheme or host", c.Name, "logging", c.LoggingURL))
			})

			t.Run("missing host", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.LoggingURL = "https://"
				// when
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
				// then
				testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': '%s' URL '%s' is invalid: missing scheme or host", c.Name, "logging", c.LoggingURL))
			})

		})

		t.Run("invalid metrics URL", func(t *testing.T) {

			t.Run("missing scheme", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.MetricsURL = "metrics.cluster-foo.com"
				// when
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
				// then
				testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': '%s' URL '%s' is invalid: missing scheme or host", c.Name, "metrics", c.MetricsURL))
			})

			t.Run("missing host", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.MetricsURL = "https://"
				// when
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
				// then
				testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': '%s' URL '%s' is invalid: missing scheme or host", c.Name, "metrics", c.MetricsURL))
			})

		})

		t.Run("invalid type", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.Type = "FOO"
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			// then
			testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': invalid type of cluster: '%s' (expected 'OSO', 'OCP' or 'OSD')", c.Name, c.Type))
		})
	})
}

func newTestCluster() *repository.Cluster {
	name := uuid.NewV4().String()
	return &repository.Cluster{
		Name:              name,
		Type:              cluster.OCP,
		URL:               fmt.Sprintf("https://api.cluster.%s", name),
		AppDNS:            fmt.Sprintf("https://cluster.%s", name),
		ConsoleURL:        fmt.Sprintf("https://console.cluster.%s", name),
		LoggingURL:        fmt.Sprintf("https://logging.cluster.%s", name),
		MetricsURL:        fmt.Sprintf("https://metrics.cluster.%s", name),
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
