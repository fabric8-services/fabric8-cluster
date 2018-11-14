package repository_test

import (
	"context"
	"github.com/satori/go.uuid"
	"testing"

	"github.com/fabric8-services/fabric8-cluster/cluster/repository"
	"github.com/fabric8-services/fabric8-cluster/gormtestsupport"
	"github.com/fabric8-services/fabric8-cluster/test"
	"github.com/fabric8-services/fabric8-common/errors"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type clusterTestSuite struct {
	gormtestsupport.DBTestSuite
	repo repository.ClusterRepository
}

func TestCluster(t *testing.T) {
	suite.Run(t, &clusterTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *clusterTestSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.repo = repository.NewClusterRepository(s.DB)
}

func (s *clusterTestSuite) TestCreateAndLoadClusterOK() {
	cluster1 := test.CreateCluster(s.T(), s.DB)
	test.CreateCluster(s.T(), s.DB) // noise
	loaded, err := s.repo.Load(context.Background(), cluster1.ClusterID)
	require.NoError(s.T(), err)

	test.AssertEqualClusters(s.T(), cluster1, loaded)
}

func (s *clusterTestSuite) TestDeleteOK() {
	cluster1 := test.CreateCluster(s.T(), s.DB)
	cluster2 := test.CreateCluster(s.T(), s.DB) // noise
	err := s.repo.Delete(context.Background(), cluster1.ClusterID)
	require.NoError(s.T(), err)

	_, err = s.repo.Load(context.Background(), cluster1.ClusterID)
	test.AssertError(s.T(), err, errors.NotFoundError{}, "cluster with id '%s' not found", cluster1.ClusterID)

	loaded, err := s.repo.Load(context.Background(), cluster2.ClusterID)
	require.NoError(s.T(), err)
	test.AssertEqualClusters(s.T(), cluster2, loaded)
}

func (s *clusterTestSuite) TestDeleteUnknownFails() {
	id := uuid.NewV4()
	err := s.repo.Delete(context.Background(), id)
	test.AssertError(s.T(), err, errors.NotFoundError{}, "cluster with id '%s' not found", id)
}

func (s *clusterTestSuite) TestLoadUnknownFails() {
	id := uuid.NewV4()
	_, err := s.repo.Load(context.Background(), id)
	test.AssertError(s.T(), err, errors.NotFoundError{}, "cluster with id '%s' not found", id)
}

func (s *clusterTestSuite) TestSaveOK() {
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
	cluster1.SaToken = uuid.NewV4().String()
	cluster1.SaUsername = uuid.NewV4().String()
	cluster1.TokenProviderID = uuid.NewV4().String()
	cluster1.Type = uuid.NewV4().String()
	cluster1.URL = uuid.NewV4().String()

	err := s.repo.Save(context.Background(), cluster1)
	require.NoError(s.T(), err)

	loaded1, err := s.repo.Load(context.Background(), cluster1.ClusterID)
	test.AssertEqualClusters(s.T(), cluster1, loaded1)

	loaded2, err := s.repo.Load(context.Background(), cluster2.ClusterID)
	test.AssertEqualClusters(s.T(), cluster2, loaded2)
}

func (s *clusterTestSuite) TestSaveUnknownFails() {
	id := uuid.NewV4()
	err := s.repo.Save(context.Background(), &repository.Cluster{ClusterID: id})
	test.AssertError(s.T(), err, errors.NotFoundError{}, "cluster with id '%s' not found", id)
}

func (s *clusterTestSuite) TestExists() {
	id := uuid.NewV4()
	err := s.repo.CheckExists(context.Background(), id.String())
	test.AssertError(s.T(), err, errors.NotFoundError{}, "cluster with id '%s' not found", id)

	cluster := test.CreateCluster(s.T(), s.DB)
	err = s.repo.CheckExists(context.Background(), cluster.ClusterID.String())
	require.NoError(s.T(), err)
}
