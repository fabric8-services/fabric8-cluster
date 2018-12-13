package controller_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-cluster/app"

	"github.com/stretchr/testify/assert"

	"github.com/fabric8-services/fabric8-cluster/app/test"
	"github.com/fabric8-services/fabric8-cluster/controller"
	. "github.com/fabric8-services/fabric8-cluster/controller"
	testsupport "github.com/fabric8-services/fabric8-cluster/test"
	authsupport "github.com/fabric8-services/fabric8-common/auth"
	"github.com/fabric8-services/fabric8-common/httpsupport"
	authtestsupport "github.com/fabric8-services/fabric8-common/test/auth"

	"github.com/fabric8-services/fabric8-cluster/gormtestsupport"
	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ClusterControllerTestSuite struct {
	gormtestsupport.DBTestSuite
}

func TestClusterController(t *testing.T) {
	suite.Run(t, &ClusterControllerTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *ClusterControllerTestSuite) newSecuredControllerWithServiceAccount(serviceAccount *authtestsupport.Identity) (*goa.Service, *controller.ClustersController) {
	svc, err := authtestsupport.ServiceAsServiceAccountUser("Token-Service", serviceAccount)
	require.NoError(s.T(), err)
	return svc, NewClustersController(svc, s.Configuration, s.Application)
}

func (s *ClusterControllerTestSuite) TestShowForServiceAccountsOK() {
	require.True(s.T(), len(s.Configuration.GetClusters()) > 0)
	s.checkShowForServiceAccount("fabric8-oso-proxy")
	s.checkShowForServiceAccount("fabric8-tenant")
	s.checkShowForServiceAccount("fabric8-jenkins-idler")
	s.checkShowForServiceAccount("fabric8-jenkins-proxy")
	s.checkShowForServiceAccount("fabric8-auth")
}

func (s *ClusterControllerTestSuite) checkShowForServiceAccount(saName string) {
	sa := &authtestsupport.Identity{
		Username: saName,
		ID:       uuid.NewV4(),
	}
	svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)
	_, clusters := test.ShowClustersOK(s.T(), svc.Context, svc, ctrl)
	require.NotNil(s.T(), clusters)
	require.NotNil(s.T(), clusters.Data)
	require.Equal(s.T(), len(s.Configuration.GetClusters()), len(clusters.Data))
	for _, cluster := range clusters.Data {
		configCluster := s.Configuration.GetClusterByURL(cluster.APIURL)
		require.NotNil(s.T(), configCluster)
		assert.Equal(s.T(), configCluster.Name, cluster.Name)
		assert.Equal(s.T(), httpsupport.AddTrailingSlashToURL(configCluster.APIURL), cluster.APIURL)
		assert.Equal(s.T(), httpsupport.AddTrailingSlashToURL(configCluster.ConsoleURL), cluster.ConsoleURL)
		assert.Equal(s.T(), httpsupport.AddTrailingSlashToURL(configCluster.MetricsURL), cluster.MetricsURL)
		assert.Equal(s.T(), httpsupport.AddTrailingSlashToURL(configCluster.LoggingURL), cluster.LoggingURL)
		assert.Equal(s.T(), configCluster.AppDNS, cluster.AppDNS)
		assert.Equal(s.T(), configCluster.Type, cluster.Type)
		assert.Equal(s.T(), configCluster.CapacityExhausted, cluster.CapacityExhausted)
	}
}

func (s *ClusterControllerTestSuite) TestShowForUnknownSAFails() {
	sa := &authtestsupport.Identity{
		Username: "unknown-sa",
		ID:       uuid.NewV4(),
	}
	svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)
	test.ShowClustersUnauthorized(s.T(), svc.Context, svc, ctrl)
}

func (s *ClusterControllerTestSuite) TestShowForAuthServiceAccountsOK() {
	require.NotEmpty(s.T(), s.Configuration.GetClusters())
	s.checkShowAuthForServiceAccount("fabric8-auth")
}

func (s *ClusterControllerTestSuite) TestShowAuthForUnknownSAFails() {
	sa := &authtestsupport.Identity{
		Username: "fabric8-tenant",
		ID:       uuid.NewV4(),
	}
	svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)
	test.ShowAuthClientClustersUnauthorized(s.T(), svc.Context, svc, ctrl)
}

func createLinkIdentityClusterPayload(clusterURL, identityID string, ignoreIfExists *bool) *app.LinkIdentityToClusterData {
	return &app.LinkIdentityToClusterData{ClusterURL: clusterURL, IdentityID: identityID, IgnoreIfAlreadyExists: ignoreIfExists}
}

func (s *ClusterControllerTestSuite) checkShowAuthForServiceAccount(saName string) {
	sa := &authtestsupport.Identity{
		Username: saName,
		ID:       uuid.NewV4(),
	}
	svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)
	_, clusters := test.ShowAuthClientClustersOK(s.T(), svc.Context, svc, ctrl)
	require.NotNil(s.T(), clusters)
	require.NotNil(s.T(), clusters.Data)
	require.Equal(s.T(), len(s.Configuration.GetClusters()), len(clusters.Data))
	for _, cluster := range clusters.Data {
		configCluster := s.Configuration.GetClusterByURL(cluster.APIURL)
		require.NotNil(s.T(), configCluster)
		assert.Equal(s.T(), configCluster.Name, cluster.Name)
		assert.Equal(s.T(), httpsupport.AddTrailingSlashToURL(configCluster.APIURL), cluster.APIURL)
		assert.Equal(s.T(), httpsupport.AddTrailingSlashToURL(configCluster.ConsoleURL), cluster.ConsoleURL)
		assert.Equal(s.T(), httpsupport.AddTrailingSlashToURL(configCluster.MetricsURL), cluster.MetricsURL)
		assert.Equal(s.T(), httpsupport.AddTrailingSlashToURL(configCluster.LoggingURL), cluster.LoggingURL)
		assert.Equal(s.T(), configCluster.AppDNS, cluster.AppDNS)
		assert.Equal(s.T(), configCluster.Type, cluster.Type)
		assert.Equal(s.T(), configCluster.CapacityExhausted, cluster.CapacityExhausted)
		assert.Equal(s.T(), configCluster.AuthClientDefaultScope, cluster.AuthClientDefaultScope)
		assert.Equal(s.T(), configCluster.AuthClientID, cluster.AuthClientID)
		assert.Equal(s.T(), configCluster.AuthClientSecret, cluster.AuthClientSecret)
		assert.Equal(s.T(), configCluster.ServiceAccountToken, cluster.ServiceAccountToken)
		assert.Equal(s.T(), configCluster.ServiceAccountUsername, cluster.ServiceAccountUsername)
		assert.Equal(s.T(), configCluster.TokenProviderID, cluster.TokenProviderID)
	}
}

func (s *ClusterControllerTestSuite) TestCreate() {

	s.T().Run("ok", func(t *testing.T) {
		// given
		sa := &authtestsupport.Identity{
			Username: authsupport.ToolChainOperator,
			ID:       uuid.NewV4(),
		}
		clusterPayload := newCreateClusterPayload()
		svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)
		// when/then
		test.CreateClustersCreated(t, svc.Context, svc, ctrl, &clusterPayload)
	})

	s.T().Run("failure", func(t *testing.T) {

		t.Run("invalid token account", func(t *testing.T) {
			// given
			sa := &authtestsupport.Identity{
				Username: authsupport.Auth, // use another, unaccepted SA token
				ID:       uuid.NewV4(),
			}
			clusterPayload := newCreateClusterPayload()
			svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)
			// when/then
			test.CreateClustersUnauthorized(t, svc.Context, svc, ctrl, &clusterPayload)
		})

		t.Run("bad request", func(t *testing.T) {
			// given
			sa := &authtestsupport.Identity{
				Username: authsupport.ToolChainOperator,
				ID:       uuid.NewV4(),
			}
			clusterPayload := newCreateClusterPayload()
			clusterPayload.Data.APIURL = " "
			svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)
			// when/then
			test.CreateClustersBadRequest(t, svc.Context, svc, ctrl, &clusterPayload)
		})
	})
}

func (s *ClusterControllerTestSuite) TestLinkIdentityClusters() {

	s.T().Run("ok", func(t *testing.T) {
		t.Run("ignore if exists - nil", func(t *testing.T) {
			// given
			sa := &authtestsupport.Identity{
				Username: "fabric8-auth",
				ID:       uuid.NewV4(),
			}

			svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)

			c := testsupport.CreateCluster(t, s.DB)

			payload := createLinkIdentityClusterPayload(c.URL, uuid.NewV4().String(), nil)

			// when/then
			test.LinkIdentityToClusterClustersNoContent(t, svc.Context, svc, ctrl, payload)
		})

		t.Run("ignore if exists - true", func(t *testing.T) {
			// given
			sa := &authtestsupport.Identity{
				Username: "fabric8-auth",
				ID:       uuid.NewV4(),
			}

			svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)

			c := testsupport.CreateCluster(t, s.DB)
			ignore := true
			payload := createLinkIdentityClusterPayload(c.URL, uuid.NewV4().String(), &ignore)

			// when/then
			test.LinkIdentityToClusterClustersNoContent(t, svc.Context, svc, ctrl, payload)
		})

	})

	s.T().Run("bad", func(t *testing.T) {

		t.Run("invalid uuid", func(t *testing.T) {
			// given
			sa := &authtestsupport.Identity{
				Username: "fabric8-auth",
				ID:       uuid.NewV4(),
			}

			svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)

			c := testsupport.CreateCluster(t, s.DB)
			payload := createLinkIdentityClusterPayload(c.URL, "foo", nil)

			// when/then
			test.LinkIdentityToClusterClustersBadRequest(t, svc.Context, svc, ctrl, payload)
		})

		t.Run("empty space uuid", func(t *testing.T) {
			// given
			sa := &authtestsupport.Identity{
				Username: "fabric8-auth",
				ID:       uuid.NewV4(),
			}

			svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)

			c := testsupport.CreateCluster(t, s.DB)
			payload := createLinkIdentityClusterPayload(c.URL, "  ", nil)

			// when/then
			test.LinkIdentityToClusterClustersBadRequest(t, svc.Context, svc, ctrl, payload)
		})

		t.Run("unknown cluster", func(t *testing.T) {
			// given
			sa := &authtestsupport.Identity{
				Username: "fabric8-auth",
				ID:       uuid.NewV4(),
			}

			svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)

			payload := createLinkIdentityClusterPayload("http://foo.com", uuid.NewV4().String(), nil)

			// when/then
			test.LinkIdentityToClusterClustersBadRequest(t, svc.Context, svc, ctrl, payload)
		})

		t.Run("invalid cluster url", func(t *testing.T) {
			// given
			sa := &authtestsupport.Identity{
				Username: "fabric8-auth",
				ID:       uuid.NewV4(),
			}

			svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)

			payload := createLinkIdentityClusterPayload("foo.com", uuid.NewV4().String(), nil)

			// when/then
			test.LinkIdentityToClusterClustersBadRequest(t, svc.Context, svc, ctrl, payload)
		})
	})

	s.T().Run("unauthorized", func(t *testing.T) {
		t.Run("unknown token", func(t *testing.T) {
			// given
			sa := &authtestsupport.Identity{
				Username: "unknown",
				ID:       uuid.NewV4(),
			}
			c := testsupport.CreateCluster(t, s.DB)
			payload := createLinkIdentityClusterPayload(c.URL, uuid.NewV4().String(), nil)
			svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)

			// when/then
			test.LinkIdentityToClusterClustersUnauthorized(t, svc.Context, svc, ctrl, payload)
		})
	})

	s.T().Run("internal error - ignore false", func(t *testing.T) {
		// given
		c := testsupport.CreateCluster(s.T(), s.DB)
		identityID := uuid.NewV4()

		testsupport.CreateIdentityCluster(t, s.DB, c, &identityID)

		sa := &authtestsupport.Identity{
			Username: "fabric8-auth",
			ID:       uuid.NewV4(),
		}

		ignore := false
		payload := createLinkIdentityClusterPayload(c.URL, identityID.String(), &ignore)
		svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)

		// when/then
		test.LinkIdentityToClusterClustersInternalServerError(t, svc.Context, svc, ctrl, payload)
	})
}

func (s *ClusterControllerTestSuite) TestRemoveIdentityToClustersLink() {

	s.T().Run("ok", func(t *testing.T) {
		t.Run("unlink", func(t *testing.T) {
			// given
			sa := &authtestsupport.Identity{
				Username: "fabric8-auth",
				ID:       uuid.NewV4(),
			}

			svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)

			c := testsupport.CreateCluster(t, s.DB)
			identityID := uuid.NewV4()

			testsupport.CreateIdentityCluster(t, s.DB, c, &identityID)

			payload := createUnLinkIdentityToClusterData(c.URL, identityID.String())

			// when/then
			test.RemoveIdentityToClusterLinkClustersNoContent(t, svc.Context, svc, ctrl, payload)
		})
	})

	s.T().Run("not found", func(t *testing.T) {
		t.Run("different identity", func(t *testing.T) {
			// given
			sa := &authtestsupport.Identity{
				Username: "fabric8-auth",
				ID:       uuid.NewV4(),
			}

			svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)

			c := testsupport.CreateCluster(t, s.DB)
			identityID := uuid.NewV4()

			testsupport.CreateIdentityCluster(t, s.DB, c, &identityID)
			payload := createUnLinkIdentityToClusterData(c.URL, uuid.NewV4().String())

			// when/then
			test.RemoveIdentityToClusterLinkClustersNotFound(t, svc.Context, svc, ctrl, payload)
		})
	})

	s.T().Run("bad", func(t *testing.T) {

		t.Run("empty space uuid", func(t *testing.T) {
			// given
			sa := &authtestsupport.Identity{
				Username: "fabric8-auth",
				ID:       uuid.NewV4(),
			}

			svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)

			c := testsupport.CreateCluster(t, s.DB)
			payload := createUnLinkIdentityToClusterData(c.URL, "  ")

			// when/then
			test.RemoveIdentityToClusterLinkClustersBadRequest(t, svc.Context, svc, ctrl, payload)
		})

		t.Run("unknown cluster", func(t *testing.T) {
			// given
			sa := &authtestsupport.Identity{
				Username: "fabric8-auth",
				ID:       uuid.NewV4(),
			}

			svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)

			payload := createUnLinkIdentityToClusterData("http://foo.com", uuid.NewV4().String())

			// when/then
			test.RemoveIdentityToClusterLinkClustersBadRequest(t, svc.Context, svc, ctrl, payload)
		})

		t.Run("invalid cluster url", func(t *testing.T) {
			// given
			sa := &authtestsupport.Identity{
				Username: "fabric8-auth",
				ID:       uuid.NewV4(),
			}

			svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)

			payload := createUnLinkIdentityToClusterData("foo.com", uuid.NewV4().String())

			// when/then
			test.RemoveIdentityToClusterLinkClustersBadRequest(t, svc.Context, svc, ctrl, payload)
		})
	})

	s.T().Run("unauthorized", func(t *testing.T) {
		t.Run("unknown token", func(t *testing.T) {
			// given
			sa := &authtestsupport.Identity{
				Username: "unknown",
				ID:       uuid.NewV4(),
			}
			c := testsupport.CreateCluster(t, s.DB)
			payload := createUnLinkIdentityToClusterData(c.URL, uuid.NewV4().String())
			svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)

			// when/then
			test.RemoveIdentityToClusterLinkClustersUnauthorized(t, svc.Context, svc, ctrl, payload)
		})
	})
}

func createUnLinkIdentityToClusterData(clusterURL, identityID string) *app.UnLinkIdentityToClusterdata {
	return &app.UnLinkIdentityToClusterdata{ClusterURL: clusterURL, IdentityID: identityID}
}

func newCreateClusterPayload() app.CreateClustersPayload {
	random := uuid.NewV4().String()
	return app.CreateClustersPayload{
		Data: &app.CreateClusterData{
			Name:                   "foo-cluster",
			APIURL:                 "https://api.foo.com",
			AppDNS:                 "foo.com",
			AuthClientDefaultScope: "foo",
			AuthClientID:           uuid.NewV4().String(),
			AuthClientSecret:       uuid.NewV4().String(),
			ServiceAccountToken:    uuid.NewV4().String(),
			ServiceAccountUsername: "foo-sa",
			TokenProviderID:        &random,
			Type:                   "OSD",
		},
	}
}
