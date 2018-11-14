package repository_test

import (
	"context"
	"testing"

	"github.com/fabric8-services/fabric8-cluster/cluster/repository"
	"github.com/fabric8-services/fabric8-cluster/gormtestsupport"
	"github.com/fabric8-services/fabric8-cluster/test"
	"github.com/fabric8-services/fabric8-common/errors"

	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type identityClusterTestSuite struct {
	gormtestsupport.DBTestSuite
	repo repository.IdentityClusterRepository
}

func TestIdentityCluster(t *testing.T) {
	suite.Run(t, &identityClusterTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *identityClusterTestSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.repo = repository.NewIdentityClusterRepository(s.DB)
}

func (s *identityClusterTestSuite) TestCreateAndListIdentityClusterOK() {
	// Create two identities for the same cluster
	idCluster1 := test.CreateIdentityCluster(s.T(), s.DB, nil, nil)
	idCluster2 := test.CreateIdentityCluster(s.T(), s.DB, nil, &idCluster1.IdentityID)

	// Noise
	test.CreateIdentityCluster(s.T(), s.DB, &idCluster1.Cluster, nil)
	test.CreateIdentityCluster(s.T(), s.DB, nil, nil)

	clusters, err := s.repo.ListClustersForIdentity(context.Background(), idCluster1.IdentityID)
	require.NoError(s.T(), err)
	assert.Len(s.T(), clusters, 2)
	assertContainsCluster(s.T(), clusters, &idCluster1.Cluster)
	assertContainsCluster(s.T(), clusters, &idCluster2.Cluster)
}

func (s *identityClusterTestSuite) TestListClustersForUnknownIdentityOK() {
	clusters, err := s.repo.ListClustersForIdentity(context.Background(), uuid.NewV4())
	require.NoError(s.T(), err)
	assert.Len(s.T(), clusters, 0)
}

func assertContainsCluster(t *testing.T, clusters []repository.Cluster, cluster *repository.Cluster) {
	require.NotEqual(t, uuid.UUID{}, cluster.ClusterID)
	for _, cls := range clusters {
		if cls.ClusterID == cluster.ClusterID {
			return
		}
	}
	assert.Fail(t, "didn't find cluster")
}

func (s *identityClusterTestSuite) TestDeleteOK() {
	idCluster1 := test.CreateIdentityCluster(s.T(), s.DB, nil, nil)

	// Noise
	idCluster2 := test.CreateIdentityCluster(s.T(), s.DB, nil, &idCluster1.IdentityID)
	idCluster3 := test.CreateIdentityCluster(s.T(), s.DB, &idCluster1.Cluster, nil)
	idCluster4 := test.CreateIdentityCluster(s.T(), s.DB, nil, nil)

	err := s.repo.Delete(context.Background(), idCluster1.IdentityID, idCluster1.ClusterID)
	require.NoError(s.T(), err)

	_, err = s.repo.Load(context.Background(), idCluster1.IdentityID, idCluster1.ClusterID)
	test.AssertError(s.T(), err, errors.NotFoundError{}, "identity_cluster with identity ID %s and cluster ID %s not found", idCluster1.IdentityID, idCluster1.ClusterID)

	// Noise is still here
	loaded, err := s.repo.Load(context.Background(), idCluster2.IdentityID, idCluster2.ClusterID)
	require.NoError(s.T(), err)
	test.AssertEqualIdentityClusters(s.T(), idCluster2, loaded)
	loaded, err = s.repo.Load(context.Background(), idCluster3.IdentityID, idCluster3.ClusterID)
	require.NoError(s.T(), err)
	test.AssertEqualIdentityClusters(s.T(), idCluster3, loaded)
	loaded, err = s.repo.Load(context.Background(), idCluster4.IdentityID, idCluster4.ClusterID)
	require.NoError(s.T(), err)
	test.AssertEqualIdentityClusters(s.T(), idCluster4, loaded)
}

func (s *identityClusterTestSuite) TestDeleteUnknownFails() {
	id := uuid.NewV4()
	cluster := uuid.NewV4()
	err := s.repo.Delete(context.Background(), id, cluster)
	test.AssertError(s.T(), err, errors.NotFoundError{}, "nothing to delete: identity cluster not found (identityID:\"%s\", clusterID:\"%s\")", id, cluster)
}

func (s *identityClusterTestSuite) TestOnDeleteCascade() {
	idCluster1 := test.CreateIdentityCluster(s.T(), s.DB, nil, nil)
	idCluster2 := test.CreateIdentityCluster(s.T(), s.DB, &idCluster1.Cluster, nil)

	// Noise
	idCluster3 := test.CreateIdentityCluster(s.T(), s.DB, nil, &idCluster1.IdentityID)
	idCluster4 := test.CreateIdentityCluster(s.T(), s.DB, nil, nil)

	// Hard delete cluster
	repo := repository.NewClusterRepository(s.DB)
	repo.Delete(context.Background(), idCluster1.ClusterID)

	// Identity clusters are gone
	_, err := s.repo.Load(context.Background(), idCluster1.IdentityID, idCluster1.ClusterID)
	test.AssertError(s.T(), err, errors.NotFoundError{}, "identity_cluster with identity ID %s and cluster ID %s not found", idCluster1.IdentityID, idCluster1.ClusterID)
	_, err = s.repo.Load(context.Background(), idCluster2.IdentityID, idCluster2.ClusterID)
	test.AssertError(s.T(), err, errors.NotFoundError{}, "identity_cluster with identity ID %s and cluster ID %s not found", idCluster2.IdentityID, idCluster2.ClusterID)

	// Noise is still here
	loaded, err := s.repo.Load(context.Background(), idCluster3.IdentityID, idCluster3.ClusterID)
	require.NoError(s.T(), err)
	test.AssertEqualIdentityClusters(s.T(), idCluster3, loaded)
	loaded, err = s.repo.Load(context.Background(), idCluster4.IdentityID, idCluster4.ClusterID)
	require.NoError(s.T(), err)
	test.AssertEqualIdentityClusters(s.T(), idCluster4, loaded)
}

func (s *identityClusterTestSuite) TestLoadUnknownFails() {
	id := uuid.NewV4()
	cluster := uuid.NewV4()
	_, err := s.repo.Load(context.Background(), id, cluster)
	test.AssertError(s.T(), err, errors.NotFoundError{}, "identity_cluster with identity ID %s and cluster ID %s not found", id, cluster)
}
