package repository_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-cluster/cluster/repository"
	"github.com/fabric8-services/fabric8-cluster/gormtestsupport"
	"github.com/fabric8-services/fabric8-cluster/test"
	"github.com/fabric8-services/fabric8-common/errors"
	"github.com/fabric8-services/fabric8-common/httpsupport"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type clusterRepositoryTestSuite struct {
	gormtestsupport.DBTestSuite
	repo repository.ClusterRepository
}

func TestClusterRepository(t *testing.T) {
	suite.Run(t, &clusterRepositoryTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *clusterRepositoryTestSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.repo = s.Application.Clusters()
}

func (s *clusterRepositoryTestSuite) TestCreateAndLoadClusterOK() {
	// given
	cluster1 := test.CreateCluster(s.T(), s.DB)
	test.CreateCluster(s.T(), s.DB) // noise
	// when
	loaded, err := s.repo.Load(context.Background(), cluster1.ClusterID)
	// then
	require.NoError(s.T(), err)
	require.NotNil(s.T(), loaded)
	test.AssertEqualCluster(s.T(), cluster1, *loaded, true)
}

func (s *clusterRepositoryTestSuite) TestCreateAndLoadClusterByURLOK() {
	// given
	cluster1 := test.CreateCluster(s.T(), s.DB)
	test.CreateCluster(s.T(), s.DB) // noise
	// make sure the URL has a trailing slash at this point
	require.True(s.T(), strings.HasSuffix(cluster1.URL, "/"))

	s.T().Run("search without trailing slash", func(t *testing.T) {
		// when
		loaded, err := s.repo.LoadByURL(context.Background(), cluster1.URL)
		// then
		require.NoError(t, err)
		require.NotNil(t, loaded)
		test.AssertEqualCluster(t, cluster1, *loaded, true)
	})

	s.T().Run("search with trailing slash", func(t *testing.T) {
		// when
		loaded, err := s.repo.LoadByURL(context.Background(), httpsupport.AddTrailingSlashToURL(cluster1.URL))
		// then
		require.NoError(t, err)
		require.NotNil(t, loaded)
		test.AssertEqualCluster(t, cluster1, *loaded, true)
	})

}

func (s *clusterRepositoryTestSuite) TestCreateAndLoadClusterByURLFail() {
	// given
	test.CreateCluster(s.T(), s.DB)
	test.CreateCluster(s.T(), s.DB) // noise
	// when
	clusterURL := uuid.NewV4().String()
	loaded, err := s.repo.LoadByURL(context.Background(), clusterURL)
	// then
	assert.Nil(s.T(), loaded)
	test.AssertError(s.T(), err, errors.NotFoundError{}, fmt.Sprintf("cluster with url '%s' not found", clusterURL))
}

func (s *clusterRepositoryTestSuite) TestCreateOKInCreateOrSave() {
	// given
	cluster := test.NewCluster()
	err := s.repo.CreateOrSave(context.Background(), &cluster)
	require.NoError(s.T(), err)
	test.CreateCluster(s.T(), s.DB) // noise
	// when
	loaded, err := s.repo.LoadByURL(context.Background(), cluster.URL)
	// then
	require.NoError(s.T(), err)
	require.NotNil(s.T(), loaded)
	test.AssertEqualCluster(s.T(), cluster, *loaded, true)
}

func (s *clusterRepositoryTestSuite) TestSaveOKInCreateOrSave() {
	// given
	cluster := test.NewCluster()
	test.CreateCluster(s.T(), s.DB) // noise
	err := s.repo.CreateOrSave(context.Background(), &cluster)
	require.NoError(s.T(), err)
	loaded, err := s.repo.LoadByURL(context.Background(), cluster.URL)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), loaded)
	test.AssertEqualCluster(s.T(), cluster, *loaded, true)
	// update cluster details
	cluster.AppDNS = uuid.NewV4().String()
	cluster.AuthClientID = uuid.NewV4().String()
	cluster.AuthClientSecret = uuid.NewV4().String()
	cluster.AuthDefaultScope = uuid.NewV4().String()
	cluster.ConsoleURL = uuid.NewV4().String()
	cluster.LoggingURL = uuid.NewV4().String()
	cluster.MetricsURL = uuid.NewV4().String()
	cluster.Name = uuid.NewV4().String()
	cluster.SAToken = uuid.NewV4().String()
	cluster.SAUsername = uuid.NewV4().String()
	cluster.TokenProviderID = uuid.NewV4().String()
	cluster.Type = uuid.NewV4().String()
	cluster.CapacityExhausted = true
	err = s.repo.CreateOrSave(context.Background(), &cluster)
	require.NoError(s.T(), err)
	// when
	loaded, err = s.repo.LoadByURL(context.Background(), cluster.URL)
	// then
	require.NoError(s.T(), err)
	require.NotNil(s.T(), loaded)
	test.AssertEqualCluster(s.T(), cluster, *loaded, true)
}

func (s *clusterRepositoryTestSuite) TestDelete() {

	s.T().Run("ok", func(t *testing.T) {
		// given
		cluster1 := test.CreateCluster(t, s.DB)
		cluster2 := test.CreateCluster(t, s.DB) // noise
		// when
		err := s.repo.Delete(context.Background(), cluster1.ClusterID)
		// then
		require.NoError(t, err)
		_, err = s.repo.Load(context.Background(), cluster1.ClusterID)
		test.AssertError(t, err, errors.NotFoundError{}, "cluster with id '%s' not found", cluster1.ClusterID)
		loaded, err := s.repo.Load(context.Background(), cluster2.ClusterID)
		require.NoError(t, err)
		require.NotNil(s.T(), loaded)
		test.AssertEqualCluster(t, cluster2, *loaded, true)
	})

	s.T().Run("failures", func(t *testing.T) {
		t.Run("unknown cluster", func(t *testing.T) {
			// given
			id := uuid.NewV4()
			// when
			err := s.repo.Delete(context.Background(), id)
			// then
			test.AssertError(t, err, errors.NotFoundError{}, "cluster with id '%s' not found", id)
		})
	})
}

func (s *clusterRepositoryTestSuite) TestLoadUnknownFails() {
	// given
	id := uuid.NewV4()
	// when
	_, err := s.repo.Load(context.Background(), id)
	// then
	test.AssertError(s.T(), err, errors.NotFoundError{}, "cluster with id '%s' not found", id)
}

func (s *clusterRepositoryTestSuite) TestSaveOK() {
	// given
	cluster1 := test.CreateCluster(s.T(), s.DB)
	cluster2 := test.CreateCluster(s.T(), s.DB) // noise
	cluster1.AppDNS = uuid.NewV4().String()
	cluster1.AuthClientID = uuid.NewV4().String()
	cluster1.AuthClientSecret = uuid.NewV4().String()
	cluster1.AuthDefaultScope = uuid.NewV4().String()
	cluster1.ConsoleURL = uuid.NewV4().String()
	cluster1.LoggingURL = uuid.NewV4().String()
	cluster1.MetricsURL = uuid.NewV4().String()
	cluster1.Name = uuid.NewV4().String()
	cluster1.SAToken = uuid.NewV4().String()
	cluster1.SAUsername = uuid.NewV4().String()
	cluster1.TokenProviderID = uuid.NewV4().String()
	cluster1.Type = uuid.NewV4().String()
	cluster1.URL = uuid.NewV4().String()
	cluster1.CapacityExhausted = true
	// when
	err := s.repo.Save(context.Background(), &cluster1)
	// then
	require.NoError(s.T(), err)
	loaded1, err := s.repo.Load(context.Background(), cluster1.ClusterID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), loaded1)
	test.AssertEqualCluster(s.T(), cluster1, *loaded1, true)
	loaded2, err := s.repo.Load(context.Background(), cluster2.ClusterID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), loaded2)
	test.AssertEqualCluster(s.T(), cluster2, *loaded2, true)
}

func (s *clusterRepositoryTestSuite) TestSaveUnknownFails() {
	// given
	id := uuid.NewV4()
	// when
	err := s.repo.Save(context.Background(), &repository.Cluster{ClusterID: id})
	// then
	test.AssertError(s.T(), err, errors.NotFoundError{}, "cluster with id '%s' not found", id)
}

func (s *clusterRepositoryTestSuite) TestExists() {
	// given
	id := uuid.NewV4()
	err := s.repo.CheckExists(context.Background(), id.String())
	test.AssertError(s.T(), err, errors.NotFoundError{}, "cluster with id '%s' not found", id)
	// when
	cluster := test.CreateCluster(s.T(), s.DB)
	// then
	err = s.repo.CheckExists(context.Background(), cluster.ClusterID.String())
	require.NoError(s.T(), err)
}

func (s *clusterRepositoryTestSuite) TestQueryOK() {
	// given
	cluster1 := test.CreateCluster(s.T(), s.DB)
	// when
	clusters, err := s.repo.Query(func(db *gorm.DB) *gorm.DB {
		return db.Where("cluster_id = ?", cluster1.ClusterID)
	})
	// then
	require.NoError(s.T(), err)
	require.Len(s.T(), clusters, 1)
	test.AssertEqualCluster(s.T(), cluster1, clusters[0], true)
}

func (s *clusterRepositoryTestSuite) TestList() {
	// given
	cluster1 := test.CreateCluster(s.T(), s.DB)
	cluster2 := test.CreateCluster(s.T(), s.DB)
	// when
	clusters, err := s.repo.List(context.Background())
	// then
	require.NoError(s.T(), err)
	require.Len(s.T(), clusters, 2)
	test.AssertClusters(s.T(), clusters, cluster1, true)
	test.AssertClusters(s.T(), clusters, cluster2, true)
}
