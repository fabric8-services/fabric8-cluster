package controller_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/fabric8-services/fabric8-cluster/app"

	"github.com/stretchr/testify/assert"

	"github.com/fabric8-services/fabric8-cluster/app/test"
	"github.com/fabric8-services/fabric8-cluster/application"
	appservice "github.com/fabric8-services/fabric8-cluster/application/service"
	servicecontext "github.com/fabric8-services/fabric8-cluster/application/service/context"
	"github.com/fabric8-services/fabric8-cluster/application/service/factory"
	"github.com/fabric8-services/fabric8-cluster/cluster/repository"
	clusterservice "github.com/fabric8-services/fabric8-cluster/cluster/service"
	"github.com/fabric8-services/fabric8-cluster/controller"
	"github.com/fabric8-services/fabric8-cluster/gormapplication"
	"github.com/fabric8-services/fabric8-cluster/gormtestsupport"
	testservice "github.com/fabric8-services/fabric8-cluster/test/generated/application/service"
	authsupport "github.com/fabric8-services/fabric8-common/auth"
	"github.com/fabric8-services/fabric8-common/httpsupport"
	authtestsupport "github.com/fabric8-services/fabric8-common/test/auth"

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

func newClusterServiceConstructor(clusterSvc appservice.ClusterService) factory.ClusterServiceConstructor {
	return func(context servicecontext.ServiceContext, loader clusterservice.ConfigLoader) appservice.ClusterService {
		return clusterSvc
	}
}

func (s *ClusterControllerTestSuite) newSecuredControllerWithServiceAccount(serviceAccount *authtestsupport.Identity, app application.Application) (*goa.Service, *controller.ClustersController) {
	svc, err := authtestsupport.ServiceAsServiceAccountUser("Token-Service", serviceAccount)
	require.NoError(s.T(), err)
	return svc, controller.NewClustersController(svc, s.Configuration, app)
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
	svc, ctrl := s.newSecuredControllerWithServiceAccount(sa, s.Application)
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
	svc, ctrl := s.newSecuredControllerWithServiceAccount(sa, s.Application)
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
	svc, ctrl := s.newSecuredControllerWithServiceAccount(sa, s.Application)
	test.ShowAuthClientClustersUnauthorized(s.T(), svc.Context, svc, ctrl)
}

func (s *ClusterControllerTestSuite) checkShowAuthForServiceAccount(saName string) {
	sa := &authtestsupport.Identity{
		Username: saName,
		ID:       uuid.NewV4(),
	}
	svc, ctrl := s.newSecuredControllerWithServiceAccount(sa, s.Application)
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

	// given
	clusterPayload := app.CreateClustersPayload{
		Data: &app.CreateClusterData{
			Name:                   "foo-cluster",
			APIURL:                 "https://api.foo.com",
			AppDNS:                 "foo.com",
			AuthClientDefaultScope: "foo",
			AuthClientID:           uuid.NewV4().String(),
			AuthClientSecret:       uuid.NewV4().String(),
			ServiceAccountToken:    uuid.NewV4().String(),
			ServiceAccountUsername: "foo-sa",
			TokenProviderID:        uuid.NewV4().String(),
			Type:                   "OSD",
		},
	}
	clusterSvc := testservice.NewClusterServiceMock(s.T())
	// default func behaviour: do not return an error
	clusterSvc.CreateOrSaveClusterFunc = func(ctx context.Context, cl *repository.Cluster) error {
		return nil
	}

	s.T().Run("ok", func(t *testing.T) {
		// given
		sa := &authtestsupport.Identity{
			Username: authsupport.ToolChainOperator,
			ID:       uuid.NewV4(),
		}
		svc, ctrl := s.newSecuredControllerWithServiceAccount(sa, gormapplication.NewGormDB(s.DB, s.Configuration, factory.WithClusterService(newClusterServiceConstructor(clusterSvc))))
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
			svc, ctrl := s.newSecuredControllerWithServiceAccount(sa, gormapplication.NewGormDB(s.DB, s.Configuration, factory.WithClusterService(newClusterServiceConstructor(clusterSvc))))
			// when/then
			test.CreateClustersUnauthorized(t, svc.Context, svc, ctrl, &clusterPayload)
		})

		t.Run("service error", func(t *testing.T) {
			// given
			sa := &authtestsupport.Identity{
				Username: authsupport.ToolChainOperator,
				ID:       uuid.NewV4(),
			}
			clusterSvc.CreateOrSaveClusterFunc = func(ctx context.Context, cl *repository.Cluster) error {
				return fmt.Errorf("mock error!")
			}
			svc, ctrl := s.newSecuredControllerWithServiceAccount(sa, gormapplication.NewGormDB(s.DB, s.Configuration, factory.WithClusterService(newClusterServiceConstructor(clusterSvc))))
			// when/then
			test.CreateClustersInternalServerError(t, svc.Context, svc, ctrl, &clusterPayload)
		})
	})
}
