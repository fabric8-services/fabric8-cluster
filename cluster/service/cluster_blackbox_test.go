package service_test

import (
	"context"
	"github.com/fabric8-services/fabric8-cluster/cluster"
	"github.com/fabric8-services/fabric8-cluster/cluster/repository"
	"github.com/fabric8-services/fabric8-cluster/configuration"
	"github.com/fabric8-services/fabric8-cluster/gormapplication"
	"github.com/fabric8-services/fabric8-cluster/gormtestsupport"
	"github.com/fabric8-services/fabric8-cluster/test"
	"github.com/fabric8-services/fabric8-common/errors"
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestCluster(t *testing.T) {
	suite.Run(t, &ClusterServiceTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

type ClusterServiceTestSuite struct {
	gormtestsupport.DBTestSuite
}

func (s *ClusterServiceTestSuite) TestCreateOrSaveClusterOK() {
	err := s.Application.ClusterService().CreateOrSaveClusterFromConfig(context.Background())
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

func (s *ClusterServiceTestSuite) TestCreateIdentityCluster() {

	s.T().Run("ok", func(t *testing.T) {

		t.Run("ignore if exists", func(t *testing.T) {
			// given
			c1 := test.CreateCluster(s.T(), s.DB)
			identityID := uuid.NewV4()

			identityCluster1 := test.CreateIdentityCluster(s.T(), s.DB, c1, &identityID)

			// when
			err := s.Application.ClusterService().LinkIdentityToCluster(s.Ctx, identityID.String(), c1.URL)
			require.NoError(t, err)

			// then
			loaded1, err := s.Application.IdentityClusters().Load(s.Ctx, identityID, c1.ClusterID)
			require.NoError(t, err)
			test.AssertEqualClusters(t, c1, &loaded1.Cluster)
			test.AssertEqualIdentityClusters(t, identityCluster1, loaded1)

			clusters, err := s.Application.IdentityClusters().ListClustersForIdentity(s.Ctx, identityID)
			require.NoError(t, err)

			assert.Len(t, clusters, 1)
			test.AssertEqualClusters(t, c1, &clusters[0])
		})

		t.Run("link multiple clusters to single identity", func(t *testing.T) {
			// given
			c1 := test.CreateCluster(s.T(), s.DB)
			c2 := test.CreateCluster(s.T(), s.DB)
			identityID := uuid.NewV4()

			identityCluster1 := test.CreateIdentityCluster(s.T(), s.DB, c1, &identityID)

			identityCluster2 := &repository.IdentityCluster{
				ClusterID:  c2.ClusterID,
				IdentityID: identityID,
			}

			// when
			err := s.Application.ClusterService().LinkIdentityToCluster(s.Ctx, identityID.String(), c2.URL)
			require.NoError(t, err)

			// then
			loaded1, err := s.Application.IdentityClusters().Load(s.Ctx, identityID, c1.ClusterID)
			require.NoError(t, err)
			test.AssertEqualClusters(t, c1, &loaded1.Cluster)
			test.AssertEqualIdentityClusters(t, identityCluster1, loaded1)

			loaded2, err := s.Application.IdentityClusters().Load(s.Ctx, identityID, c2.ClusterID)
			require.NoError(t, err)
			test.AssertEqualClusters(t, c2, &loaded2.Cluster)
			test.AssertEqualIdentityClusters(t, identityCluster2, loaded2)
		})
	})

	s.T().Run("fail", func(t *testing.T) {
		t.Run("unknown identity", func(t *testing.T) {
			// given
			c := test.CreateCluster(t, s.DB)
			identityID := "unknown"

			// when
			err := s.Application.ClusterService().LinkIdentityToCluster(s.Ctx, identityID, c.URL)

			// then
			test.AssertError(t, err, errors.BadParameterError{}, "Bad value for parameter 'identity-id': 'identity-id %s is not a valid UUID'", identityID)
		})

		t.Run("random cluster url", func(t *testing.T) {
			// given
			url := "http://random.url"

			// when
			err := s.Application.ClusterService().LinkIdentityToCluster(s.Ctx, uuid.NewV4().String(), url)

			// then
			test.AssertError(t, err, errors.BadParameterError{}, "Bad value for parameter 'cluster-url': 'cluster with requested url %s doesn't exist'", url)
		})
	})
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
