package controller_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-cluster/app/test"
	. "github.com/fabric8-services/fabric8-cluster/controller"
	testsupport "github.com/fabric8-services/fabric8-cluster/test"
	"github.com/fabric8-services/fabric8-common/httpsupport"
	"github.com/fabric8-services/fabric8-common/test/auth"

	"github.com/fabric8-services/fabric8-cluster/app"
	"github.com/fabric8-services/fabric8-cluster/gormtestsupport"
	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ClustersTestSuite struct {
	gormtestsupport.DBTestSuite
}

func TestRunClustersREST(t *testing.T) {
	suite.Run(t, &ClustersTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *ClustersTestSuite) SecuredControllerWithServiceAccount(serviceAccount *auth.Identity) (*goa.Service, *ClustersController) {
	svc, err := auth.ServiceAsServiceAccountUser("Token-Service", serviceAccount)
	require.NoError(s.T(), err)
	return svc, NewClustersController(svc, s.Configuration, s.Application)
}

func (s *ClustersTestSuite) TestShowForServiceAccountsOK() {
	require.True(s.T(), len(s.Configuration.GetClusters()) > 0)
	s.checkShowForServiceAccount("fabric8-oso-proxy")
	s.checkShowForServiceAccount("fabric8-tenant")
	s.checkShowForServiceAccount("fabric8-jenkins-idler")
	s.checkShowForServiceAccount("fabric8-jenkins-proxy")
	s.checkShowForServiceAccount("fabric8-auth")
}

func (s *ClustersTestSuite) checkShowForServiceAccount(saName string) {
	sa := &auth.Identity{
		Username: saName,
		ID:       uuid.NewV4(),
	}
	service, controller := s.SecuredControllerWithServiceAccount(sa)
	_, clusters := test.ShowClustersOK(s.T(), service.Context, service, controller)
	require.NotNil(s.T(), clusters)
	require.NotNil(s.T(), clusters.Data)
	require.Equal(s.T(), len(s.Configuration.GetClusters()), len(clusters.Data))
	for _, cluster := range clusters.Data {
		configCluster := s.Configuration.GetClusterByURL(cluster.APIURL)
		require.NotNil(s.T(), configCluster)
		require.Equal(s.T(), configCluster.Name, cluster.Name)
		require.Equal(s.T(), httpsupport.AddTrailingSlashToURL(configCluster.APIURL), cluster.APIURL)
		require.Equal(s.T(), httpsupport.AddTrailingSlashToURL(configCluster.ConsoleURL), cluster.ConsoleURL)
		require.Equal(s.T(), httpsupport.AddTrailingSlashToURL(configCluster.MetricsURL), cluster.MetricsURL)
		require.Equal(s.T(), httpsupport.AddTrailingSlashToURL(configCluster.LoggingURL), cluster.LoggingURL)
		require.Equal(s.T(), configCluster.AppDNS, cluster.AppDNS)
		require.Equal(s.T(), configCluster.Type, cluster.Type)
		require.Equal(s.T(), configCluster.CapacityExhausted, cluster.CapacityExhausted)
	}
}

func (s *ClustersTestSuite) TestShowForUnknownSAFails() {
	sa := &auth.Identity{
		Username: "unknown-sa",
		ID:       uuid.NewV4(),
	}
	service, controller := s.SecuredControllerWithServiceAccount(sa)
	test.ShowClustersUnauthorized(s.T(), service.Context, service, controller)
}

func (s *ClustersTestSuite) TestShowForAuthServiceAccountsOK() {
	require.True(s.T(), len(s.Configuration.GetClusters()) > 0)
	s.checkShowAuthForServiceAccount("fabric8-auth")
}

func (s *ClustersTestSuite) TestShowAuthForUnknownSAFails() {
	sa := &auth.Identity{
		Username: "fabric8-tenant",
		ID:       uuid.NewV4(),
	}
	service, controller := s.SecuredControllerWithServiceAccount(sa)
	test.ShowAuthClientClustersUnauthorized(s.T(), service.Context, service, controller)
}

func (s *ClustersTestSuite) TestLinkIdentityClusters() {

	s.T().Run("ok", func(t *testing.T) {
		// given
		sa := &auth.Identity{
			Username: "fabric8-auth",
			ID:       uuid.NewV4(),
		}

		service, controller := s.SecuredControllerWithServiceAccount(sa)

		cluster := testsupport.CreateCluster(t, s.DB)
		payload := createLinkIdentityClusterPayload(cluster.URL, uuid.NewV4().String())

		// when/then
		test.LinkIdentityToClusterClustersNoContent(t, service.Context, service, controller, payload)
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
			payload := createLinkIdentityClusterPayload(cluster.URL, "foo")

			// when/then
			test.LinkIdentityToClusterClustersBadRequest(t, service.Context, service, controller, payload)
		})

		t.Run("empty uuid", func(t *testing.T) {
			// given
			sa := &auth.Identity{
				Username: "fabric8-auth",
				ID:       uuid.NewV4(),
			}

			service, controller := s.SecuredControllerWithServiceAccount(sa)

			cluster := testsupport.CreateCluster(t, s.DB)
			payload := createLinkIdentityClusterPayload(cluster.URL, " ")

			// when/then
			test.LinkIdentityToClusterClustersBadRequest(t, service.Context, service, controller, payload)
		})

		t.Run("unknown cluster", func(t *testing.T) {
			// given
			sa := &auth.Identity{
				Username: "fabric8-auth",
				ID:       uuid.NewV4(),
			}

			service, controller := s.SecuredControllerWithServiceAccount(sa)

			payload := createLinkIdentityClusterPayload("http://foo.com", uuid.NewV4().String())

			// when/then
			test.LinkIdentityToClusterClustersBadRequest(t, service.Context, service, controller, payload)
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
			payload := createLinkIdentityClusterPayload(cluster.URL, uuid.NewV4().String())
			service, controller := s.SecuredControllerWithServiceAccount(sa)

			// when/then
			test.LinkIdentityToClusterClustersUnauthorized(t, service.Context, service, controller, payload)
		})
	})
}

func createLinkIdentityClusterPayload(clusterURL, identityID string) *app.LinkIdentityToClusterData {
	attributes := app.LinkIdentityToClusterAttributes{clusterURL, identityID, nil}

	return &app.LinkIdentityToClusterData{Type: "identityclusters", Attributes: &attributes}
}

func (s *ClustersTestSuite) checkShowAuthForServiceAccount(saName string) {
	sa := &auth.Identity{
		Username: saName,
		ID:       uuid.NewV4(),
	}
	service, controller := s.SecuredControllerWithServiceAccount(sa)
	_, clusters := test.ShowAuthClientClustersOK(s.T(), service.Context, service, controller)
	require.NotNil(s.T(), clusters)
	require.NotNil(s.T(), clusters.Data)
	require.Equal(s.T(), len(s.Configuration.GetClusters()), len(clusters.Data))
	for _, cluster := range clusters.Data {
		configCluster := s.Configuration.GetClusterByURL(cluster.APIURL)
		require.NotNil(s.T(), configCluster)
		require.Equal(s.T(), configCluster.Name, cluster.Name)
		require.Equal(s.T(), httpsupport.AddTrailingSlashToURL(configCluster.APIURL), cluster.APIURL)
		require.Equal(s.T(), httpsupport.AddTrailingSlashToURL(configCluster.ConsoleURL), cluster.ConsoleURL)
		require.Equal(s.T(), httpsupport.AddTrailingSlashToURL(configCluster.MetricsURL), cluster.MetricsURL)
		require.Equal(s.T(), httpsupport.AddTrailingSlashToURL(configCluster.LoggingURL), cluster.LoggingURL)
		require.Equal(s.T(), configCluster.AppDNS, cluster.AppDNS)
		require.Equal(s.T(), configCluster.Type, cluster.Type)
		require.Equal(s.T(), configCluster.CapacityExhausted, cluster.CapacityExhausted)

		require.Equal(s.T(), configCluster.AuthClientDefaultScope, cluster.AuthClientDefaultScope)
		require.Equal(s.T(), configCluster.AuthClientID, cluster.AuthClientID)
		require.Equal(s.T(), configCluster.AuthClientSecret, cluster.AuthClientSecret)
		require.Equal(s.T(), configCluster.ServiceAccountToken, cluster.ServiceAccountToken)
		require.Equal(s.T(), configCluster.ServiceAccountUsername, cluster.ServiceAccountUsername)
		require.Equal(s.T(), configCluster.TokenProviderID, cluster.TokenProviderID)
	}
}
