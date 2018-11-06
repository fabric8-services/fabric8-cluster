package controller_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-cluster/app/test"
	. "github.com/fabric8-services/fabric8-cluster/controller"
	testsuite "github.com/fabric8-services/fabric8-cluster/test/suite"
	"github.com/fabric8-services/fabric8-common/httpsupport"
	"github.com/fabric8-services/fabric8-common/test/auth"

	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ClustersTestSuite struct {
	testsuite.UnitTestSuite
}

func TestRunClustersREST(t *testing.T) {
	suite.Run(t, &ClustersTestSuite{UnitTestSuite: testsuite.NewUnitTestSuite()})
}

func (s *ClustersTestSuite) SecuredControllerWithServiceAccount(serviceAccount *auth.Identity) (*goa.Service, *ClustersController) {
	svc, err := auth.ServiceAsServiceAccountUser("Token-Service", serviceAccount)
	require.NoError(s.T(), err)
	return svc, NewClustersController(svc, s.Config)
}

func (s *ClustersTestSuite) TestShowForServiceAccountsOK() {
	require.True(s.T(), len(s.Config.GetOSOClusters()) > 0)
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
	require.Equal(s.T(), len(s.Config.GetOSOClusters()), len(clusters.Data))
	for _, cluster := range clusters.Data {
		configCluster := s.Config.GetOSOClusterByURL(cluster.APIURL)
		require.NotNil(s.T(), configCluster)
		require.Equal(s.T(), configCluster.Name, cluster.Name)
		require.Equal(s.T(), httpsupport.AddTrailingSlashToURL(configCluster.APIURL), cluster.APIURL)
		require.Equal(s.T(), httpsupport.AddTrailingSlashToURL(configCluster.ConsoleURL), cluster.ConsoleURL)
		require.Equal(s.T(), httpsupport.AddTrailingSlashToURL(configCluster.MetricsURL), cluster.MetricsURL)
		require.Equal(s.T(), httpsupport.AddTrailingSlashToURL(configCluster.LoggingURL), cluster.LoggingURL)
		require.Equal(s.T(), configCluster.AppDNS, cluster.AppDNS)
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
	require.True(s.T(), len(s.Config.GetOSOClusters()) > 0)
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

func (s *ClustersTestSuite) checkShowAuthForServiceAccount(saName string) {
	sa := &auth.Identity{
		Username: saName,
		ID:       uuid.NewV4(),
	}
	service, controller := s.SecuredControllerWithServiceAccount(sa)
	_, clusters := test.ShowAuthClientClustersOK(s.T(), service.Context, service, controller)
	require.NotNil(s.T(), clusters)
	require.NotNil(s.T(), clusters.Data)
	require.Equal(s.T(), len(s.Config.GetOSOClusters()), len(clusters.Data))
	for _, cluster := range clusters.Data {
		configCluster := s.Config.GetOSOClusterByURL(cluster.APIURL)
		require.NotNil(s.T(), configCluster)
		require.Equal(s.T(), configCluster.Name, cluster.Name)
		require.Equal(s.T(), httpsupport.AddTrailingSlashToURL(configCluster.APIURL), cluster.APIURL)
		require.Equal(s.T(), httpsupport.AddTrailingSlashToURL(configCluster.ConsoleURL), cluster.ConsoleURL)
		require.Equal(s.T(), httpsupport.AddTrailingSlashToURL(configCluster.MetricsURL), cluster.MetricsURL)
		require.Equal(s.T(), httpsupport.AddTrailingSlashToURL(configCluster.LoggingURL), cluster.LoggingURL)
		require.Equal(s.T(), configCluster.AppDNS, cluster.AppDNS)
		require.Equal(s.T(), configCluster.CapacityExhausted, cluster.CapacityExhausted)

		require.Equal(s.T(), configCluster.AuthClientDefaultScope, cluster.AuthClientDefaultScope)
		require.Equal(s.T(), configCluster.AuthClientID, cluster.AuthClientID)
		require.Equal(s.T(), configCluster.AuthClientSecret, cluster.AuthClientSecret)
		require.Equal(s.T(), configCluster.ServiceAccountToken, cluster.ServiceAccountToken)
		require.Equal(s.T(), configCluster.ServiceAccountUsername, cluster.ServiceAccountUsername)
		require.Equal(s.T(), configCluster.TokenProviderID, cluster.TokenProviderID)
	}
}
