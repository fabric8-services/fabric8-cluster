package controller_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-cluster/app/test"
	. "github.com/fabric8-services/fabric8-cluster/controller"
	testsupport "github.com/fabric8-services/fabric8-cluster/test"
	"github.com/fabric8-services/fabric8-common/test/auth"

	"github.com/fabric8-services/fabric8-cluster/app"
	"github.com/fabric8-services/fabric8-cluster/gormtestsupport"
	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type IdentityClustersControllerTestSuite struct {
	gormtestsupport.DBTestSuite
}

func TestIdentityClustersController(t *testing.T) {
	suite.Run(t, &IdentityClustersControllerTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *IdentityClustersControllerTestSuite) SecuredControllerWithServiceAccount(serviceAccount *auth.Identity) (*goa.Service, *IdentityClustersController) {
	svc, err := auth.ServiceAsServiceAccountUser("IdentityClusters-Service", serviceAccount)
	require.NoError(s.T(), err)
	return svc, NewIdentityClustersController(svc, s.Application)
}

func (s *IdentityClustersControllerTestSuite) TestCreateIdentityClusters() {

	s.T().Run("ok", func(t *testing.T) {
		// given
		sa := &auth.Identity{
			Username: "fabric8-auth",
			ID:       uuid.NewV4(),
		}

		service, controller := s.SecuredControllerWithServiceAccount(sa)

		cluster := testsupport.CreateCluster(t, s.DB)
		payload := createIdentityClusterPayload(cluster.URL, uuid.NewV4().String())

		// when/then
		test.CreateIdentityClustersNoContent(t, service.Context, service, controller, payload)
	})

	s.T().Run("bad", func(t *testing.T) {

		t.Run("invalid uuid", func(t *testing.T) {
			// given
			sa := &auth.Identity{
				Username: "fabric8-auth",
				ID:       uuid.NewV4(),
			}

			service, controller := s.SecuredControllerWithServiceAccount(sa)

			cluster := testsupport.CreateCluster(t, s.DB)
			payload := createIdentityClusterPayload(cluster.URL, "foo")

			// when/then
			test.CreateIdentityClustersBadRequest(t, service.Context, service, controller, payload)
		})

		t.Run("empty uuid", func(t *testing.T) {
			// given
			sa := &auth.Identity{
				Username: "fabric8-auth",
				ID:       uuid.NewV4(),
			}

			service, controller := s.SecuredControllerWithServiceAccount(sa)

			cluster := testsupport.CreateCluster(t, s.DB)
			payload := createIdentityClusterPayload(cluster.URL, " ")

			// when/then
			test.CreateIdentityClustersBadRequest(t, service.Context, service, controller, payload)
		})

		t.Run("unknown cluster", func(t *testing.T) {
			// given
			sa := &auth.Identity{
				Username: "fabric8-auth",
				ID:       uuid.NewV4(),
			}

			service, controller := s.SecuredControllerWithServiceAccount(sa)

			payload := createIdentityClusterPayload("http://foo.com", uuid.NewV4().String())

			// when/then
			test.CreateIdentityClustersBadRequest(t, service.Context, service, controller, payload)
		})
	})

	s.T().Run("unauthorized", func(t *testing.T) {
		t.Run("unknown token", func(t *testing.T) {
			// given
			sa := &auth.Identity{
				Username: "unknown",
				ID:       uuid.NewV4(),
			}
			cluster := testsupport.CreateCluster(t, s.DB)
			payload := createIdentityClusterPayload(cluster.URL, uuid.NewV4().String())
			service, controller := s.SecuredControllerWithServiceAccount(sa)

			// when/then
			test.CreateIdentityClustersUnauthorized(t, service.Context, service, controller, payload)
		})
	})
}

func createIdentityClusterPayload(clusterURL, identityID string) *app.CreateIdentityClusterData {
	attributes := app.CreateIdentityClusterAttributes{clusterURL, identityID}

	return &app.CreateIdentityClusterData{Type: "identityclusters", Attributes: &attributes}
}
