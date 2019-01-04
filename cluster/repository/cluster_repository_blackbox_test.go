package repository_test

import (
	"context"
	"testing"

	uuid "github.com/satori/go.uuid"

	"github.com/fabric8-services/fabric8-cluster/cluster/repository"
	"github.com/fabric8-services/fabric8-cluster/gormtestsupport"
	"github.com/fabric8-services/fabric8-cluster/test"
	"github.com/fabric8-services/fabric8-common/errors"

	"fmt"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type clusterTestSuite struct {
	gormtestsupport.DBTestSuite
	repo repository.ClusterRepository
}

func TestClusterRepository(t *testing.T) {
	suite.Run(t, &clusterTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *clusterTestSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.repo = s.Application.Clusters()
}

func (s *clusterTestSuite) TestCreateAndLoadClusterOK() {
	// given
	cluster1 := test.CreateCluster(s.T(), s.DB)
	test.CreateCluster(s.T(), s.DB) // noise
	// when
	loaded, err := s.repo.Load(context.Background(), cluster1.ClusterID)
	// then
	require.NoError(s.T(), err)
	test.AssertEqualClusters(s.T(), cluster1, loaded)
}

func (s *clusterTestSuite) TestCreateAndLoadClusterByURLOK() {
	// given
	cluster1 := test.CreateCluster(s.T(), s.DB)
	test.CreateCluster(s.T(), s.DB) // noise
	// when
	loaded, err := s.repo.LoadClusterByURL(context.Background(), cluster1.URL)
	// then
	require.NoError(s.T(), err)
	test.AssertEqualClusters(s.T(), cluster1, loaded)
}

func (s *clusterTestSuite) TestCreateAndLoadClusterByURLFail() {
	// given
	test.CreateCluster(s.T(), s.DB)
	test.CreateCluster(s.T(), s.DB) // noise
	// when
	clusterURL := uuid.NewV4().String()
	loaded, err := s.repo.LoadClusterByURL(context.Background(), clusterURL)
	// then
	assert.Nil(s.T(), loaded)
	test.AssertError(s.T(), err, errors.NotFoundError{}, fmt.Sprintf("cluster with url %s not found", clusterURL))
}

func (s *clusterTestSuite) TestCreateOKInCreateOrSave() {
	// given
	cluster := test.NewCluster()
	s.repo.CreateOrSave(context.Background(), cluster)
	test.CreateCluster(s.T(), s.DB) // noise
	// when
	loaded, err := s.repo.LoadClusterByURL(context.Background(), cluster.URL)
	// then
	require.NoError(s.T(), err)
	test.AssertEqualClusters(s.T(), cluster, loaded)
}

func (s *clusterTestSuite) TestSaveOKInCreateOrSave() {
	// given
	cluster := test.NewCluster()
	test.CreateCluster(s.T(), s.DB) // noise
	s.repo.CreateOrSave(context.Background(), cluster)
	loaded, err := s.repo.LoadClusterByURL(context.Background(), cluster.URL)
	require.NoError(s.T(), err)
	test.AssertEqualClusters(s.T(), cluster, loaded)
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
	err = s.repo.CreateOrSave(context.Background(), cluster)
	require.NoError(s.T(), err)
	// when
	loaded, err = s.repo.LoadClusterByURL(context.Background(), cluster.URL)
	// then
	require.NoError(s.T(), err)
	test.AssertEqualClusters(s.T(), cluster, loaded)
}

func (s *clusterTestSuite) TestDeleteOK() {
	// given
	cluster1 := test.CreateCluster(s.T(), s.DB)
	cluster2 := test.CreateCluster(s.T(), s.DB) // noise
	// when
	err := s.repo.Delete(context.Background(), cluster1.ClusterID)
	// then
	require.NoError(s.T(), err)
	_, err = s.repo.Load(context.Background(), cluster1.ClusterID)
	test.AssertError(s.T(), err, errors.NotFoundError{}, "cluster with id '%s' not found", cluster1.ClusterID)
	loaded, err := s.repo.Load(context.Background(), cluster2.ClusterID)
	require.NoError(s.T(), err)
	test.AssertEqualClusters(s.T(), cluster2, loaded)
}

func (s *clusterTestSuite) TestDeleteUnknownFails() {
	// given
	id := uuid.NewV4()
	// when
	err := s.repo.Delete(context.Background(), id)
	// then
	test.AssertError(s.T(), err, errors.NotFoundError{}, "cluster with id '%s' not found", id)
}

func (s *clusterTestSuite) TestLoadUnknownFails() {
	// given
	id := uuid.NewV4()
	// when
	_, err := s.repo.Load(context.Background(), id)
	// then
	test.AssertError(s.T(), err, errors.NotFoundError{}, "cluster with id '%s' not found", id)
}

func (s *clusterTestSuite) TestSaveOK() {
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
	err := s.repo.Save(context.Background(), cluster1)
	// then
	require.NoError(s.T(), err)
	loaded1, err := s.repo.Load(context.Background(), cluster1.ClusterID)
	require.NoError(s.T(), err)
	test.AssertEqualClusters(s.T(), cluster1, loaded1)
	loaded2, err := s.repo.Load(context.Background(), cluster2.ClusterID)
	require.NoError(s.T(), err)
	test.AssertEqualClusters(s.T(), cluster2, loaded2)
}

func (s *clusterTestSuite) TestSaveUnknownFails() {
	// given
	id := uuid.NewV4()
	// when
	err := s.repo.Save(context.Background(), &repository.Cluster{ClusterID: id})
	// then
	test.AssertError(s.T(), err, errors.NotFoundError{}, "cluster with id '%s' not found", id)
}

func (s *clusterTestSuite) TestExists() {
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

func (s *clusterTestSuite) TestQueryOK() {
	// given
	cluster1 := test.CreateCluster(s.T(), s.DB)
	// when
	clusters, err := s.repo.Query(func(db *gorm.DB) *gorm.DB {
		return db.Where("cluster_id = ?", cluster1.ClusterID)
	})
	// then
	require.NoError(s.T(), err)
	require.Len(s.T(), clusters, 1)
	test.AssertEqualClusters(s.T(), cluster1, &clusters[0])
}

func (s *clusterTestSuite) TestList() {
	// given
	cluster1 := &repository.Cluster{
		AppDNS:            uuid.NewV4().String(),
		AuthClientID:      uuid.NewV4().String(),
		AuthClientSecret:  uuid.NewV4().String(),
		AuthDefaultScope:  uuid.NewV4().String(),
		ConsoleURL:        uuid.NewV4().String(),
		LoggingURL:        uuid.NewV4().String(),
		MetricsURL:        uuid.NewV4().String(),
		Name:              uuid.NewV4().String(),
		SAToken:           uuid.NewV4().String(),
		SAUsername:        uuid.NewV4().String(),
		SATokenEncrypted:  true,
		TokenProviderID:   uuid.NewV4().String(),
		Type:              "bar",
		URL:               "http://" + uuid.NewV4().String() + "/",
		CapacityExhausted: false,
	}
	err := s.repo.Create(context.Background(), cluster1)
	require.NoError(s.T(), err)
	cluster2 := &repository.Cluster{
		AppDNS:            uuid.NewV4().String(),
		AuthClientID:      uuid.NewV4().String(),
		AuthClientSecret:  uuid.NewV4().String(),
		AuthDefaultScope:  uuid.NewV4().String(),
		ConsoleURL:        uuid.NewV4().String(),
		LoggingURL:        uuid.NewV4().String(),
		MetricsURL:        uuid.NewV4().String(),
		Name:              uuid.NewV4().String(),
		SAToken:           uuid.NewV4().String(),
		SAUsername:        uuid.NewV4().String(),
		SATokenEncrypted:  true,
		TokenProviderID:   uuid.NewV4().String(),
		Type:              "bar",
		URL:               "http://" + uuid.NewV4().String() + "/",
		CapacityExhausted: false,
	}
	err = s.repo.Create(context.Background(), cluster2)
	require.NoError(s.T(), err)
	// when
	clusters, err := s.repo.List(context.Background())
	// then
	require.NoError(s.T(), err)
	require.Len(s.T(), clusters, 2)
	test.AssertEqualClusters(s.T(), cluster1, &clusters[0])
	test.AssertEqualClusters(s.T(), cluster2, &clusters[1])
}
