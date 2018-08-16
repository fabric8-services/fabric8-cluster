package controller_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-cluster/app/test"
	. "github.com/fabric8-services/fabric8-cluster/controller"
	"github.com/fabric8-services/fabric8-cluster/rest"
	testsupport "github.com/fabric8-services/fabric8-cluster/test"
	testsuite "github.com/fabric8-services/fabric8-cluster/test/suite"

	"github.com/goadesign/goa"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ClustersTestSuite struct {
	testsuite.UnitTestSuite
}

func TestRunClustersREST(t *testing.T) {
	suite.Run(t, &ClustersTestSuite{UnitTestSuite: testsuite.NewUnitTestSuite()})
}

func (s *ClustersTestSuite) SecuredControllerWithServiceAccount(serviceAccount account.Identity) (*goa.Service, *ClustersController) {
	svc := testsupport.ServiceAsServiceAccountUser("Token-Service", serviceAccount)
	return svc, NewClustersController(svc, s.Config)
}

func (s *ClustersTestSuite) TestShowForServiceAccountsOK() {
	require.True(rest.T(), len(rest.Config.GetOSOClusters()) > 0)
	s.checkShowForServiceAccount("fabric8-oso-proxy")
	s.checkShowForServiceAccount("fabric8-tenant")
	s.checkShowForServiceAccount("fabric8-jenkins-idler")
	s.checkShowForServiceAccount("fabric8-jenkins-proxy")
}

func (s *ClustersTestSuite) checkShowForServiceAccount(saName string) {
	sa := account.Identity{
		Username: saName,
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
		require.Equal(s.T(), rest.AddTrailingSlashToURL(configCluster.APIURL), cluster.APIURL)
		require.Equal(s.T(), rest.AddTrailingSlashToURL(configCluster.ConsoleURL), cluster.ConsoleURL)
		require.Equal(s.T(), rest.AddTrailingSlashToURL(configCluster.MetricsURL), cluster.MetricsURL)
		require.Equal(s.T(), rest.AddTrailingSlashToURL(configCluster.LoggingURL), cluster.LoggingURL)
		require.Equal(s.T(), configCluster.AppDNS, cluster.AppDNS)
		require.Equal(s.T(), configCluster.CapacityExhausted, cluster.CapacityExhausted)
	}
}

func (s *ClustersTestSuite) TestShowForUnknownSAFails() {
	sa := account.Identity{
		Username: "unknown-sa",
	}
	service, controller := s.SecuredControllerWithServiceAccount(sa)
	test.ShowClustersUnauthorized(s.T(), service.Context, service, controller)
}
