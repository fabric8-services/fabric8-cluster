package repository_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-cluster/cluster/repository"
	"github.com/fabric8-services/fabric8-cluster/gormtestsupport"

	"github.com/satori/go.uuid"
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

func (s *clusterTestSuite) TestCreateRoleScopeOK() {
	rt, err := s.resourceTypeRepo.Lookup(s.Ctx, "openshift.io/resource/area")
	require.NoError(s.T(), err)
	require.NotNil(s.T(), rt)

	rts, err := testsupport.CreateTestScope(s.Ctx, s.DB, *rt, uuid.NewV4().String())
	require.NoError(s.T(), err)
	require.NotNil(s.T(), rts)

	randomRole, err := testsupport.CreateTestRole(s.Ctx, s.DB, *rt, "collab-"+uuid.NewV4().String())
	require.NoError(s.T(), err)

	rs := rolescope.RoleScope{
		ResourceTypeScopeID: rts.ResourceTypeScopeID,
		RoleID:              randomRole.RoleID,
	}

	err = s.repo.Create(s.Ctx, &rs)
	require.NoError(s.T(), err)
}

func (s *clusterTestSuite) TestListRoleScopeByRoleOK() {
	rt, err := s.resourceTypeRepo.Lookup(s.Ctx, "openshift.io/resource/area")
	require.NoError(s.T(), err)
	require.NotNil(s.T(), rt)

	rts, err := testsupport.CreateTestScope(s.Ctx, s.DB, *rt, uuid.NewV4().String())
	require.NoError(s.T(), err)
	require.NotNil(s.T(), rts)

	randomRole, err := testsupport.CreateTestRole(s.Ctx, s.DB, *rt, "collab-"+uuid.NewV4().String())
	require.NoError(s.T(), err)

	rs := rolescope.RoleScope{
		ResourceTypeScopeID: rts.ResourceTypeScopeID,
		RoleID:              randomRole.RoleID,
	}

	err = s.repo.Create(s.Ctx, &rs)
	require.NoError(s.T(), err)

	retrievedRoles, err := s.repo.LoadByRole(s.Ctx, randomRole.RoleID)
	require.NoError(s.T(), err)
	require.Len(s.T(), retrievedRoles, 1)
	require.Equal(s.T(), randomRole.RoleID, retrievedRoles[0].RoleID)
	require.Equal(s.T(), rs.ResourceTypeScopeID, retrievedRoles[0].ResourceTypeScopeID)

}

func (s *clusterTestSuite) TestListRoleScopeByScopeOK() {
	rt, err := s.resourceTypeRepo.Lookup(s.Ctx, "openshift.io/resource/area")
	require.NoError(s.T(), err)
	require.NotNil(s.T(), rt)

	// TODO: move to test/authorization.go
	rts := resourcetype.ResourceTypeScope{
		ResourceTypeScopeID: uuid.NewV4(),
		ResourceTypeID:      rt.ResourceTypeID,
		Name:                uuid.NewV4().String(),
	}

	err = s.resourceTypeScopeRepo.Create(s.Ctx, &rts)

	randomRole, err := testsupport.CreateTestRole(s.Ctx, s.DB, *rt, "collab-"+uuid.NewV4().String())
	require.NoError(s.T(), err)

	rs := rolescope.RoleScope{
		ResourceTypeScopeID: rts.ResourceTypeScopeID,
		RoleID:              randomRole.RoleID,
	}

	err = s.repo.Create(s.Ctx, &rs)
	require.NoError(s.T(), err)

	retrievedRoles, err := s.repo.LoadByScope(s.Ctx, rs.ResourceTypeScopeID)
	require.NoError(s.T(), err)
	require.Len(s.T(), retrievedRoles, 1)
	require.Equal(s.T(), randomRole.RoleID, retrievedRoles[0].RoleID)
	require.Equal(s.T(), rs.ResourceTypeScopeID, retrievedRoles[0].ResourceTypeScopeID)
}
