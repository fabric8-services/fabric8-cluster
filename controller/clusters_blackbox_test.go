package controller_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-cluster/app"

	"github.com/stretchr/testify/assert"

	"github.com/fabric8-services/fabric8-cluster/app/test"
	"github.com/fabric8-services/fabric8-cluster/controller"
	"github.com/fabric8-services/fabric8-cluster/gormtestsupport"
	"github.com/fabric8-services/fabric8-common/auth"
	authsupport "github.com/fabric8-services/fabric8-common/auth"
	"github.com/fabric8-services/fabric8-common/httpsupport"
	authtestsupport "github.com/fabric8-services/fabric8-common/test/auth"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
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
	return svc, controller.NewClustersController(svc, s.Configuration, s.Application)
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
	_, clusters := test.ListClustersOK(s.T(), svc.Context, svc, ctrl)
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
	test.ListClustersUnauthorized(s.T(), svc.Context, svc, ctrl)
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
	test.ListForAuthClientClustersUnauthorized(s.T(), svc.Context, svc, ctrl)
}

func (s *ClusterControllerTestSuite) checkShowAuthForServiceAccount(saName string) {
	sa := &authtestsupport.Identity{
		Username: saName,
		ID:       uuid.NewV4(),
	}
	svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)
	_, clusters := test.ListForAuthClientClustersOK(s.T(), svc.Context, svc, ctrl)
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
		// when
		resp := test.CreateClustersCreated(t, svc.Context, svc, ctrl, &clusterPayload)
		//then
		location := resp.Header().Get("location")
		require.NotEmpty(t, location)
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

func newCreateClusterPayload() app.CreateClustersPayload {
	name := uuid.NewV4().String()
	tokenProviderID := uuid.NewV4().String()
	return app.CreateClustersPayload{
		Data: &app.CreateClusterData{
			Name:                   name,
			APIURL:                 fmt.Sprintf("https://api.cluster.%s", name),
			AppDNS:                 "foo.com",
			AuthClientDefaultScope: "foo",
			AuthClientID:           uuid.NewV4().String(),
			AuthClientSecret:       uuid.NewV4().String(),
			ServiceAccountToken:    uuid.NewV4().String(),
			ServiceAccountUsername: "foo-sa",
			TokenProviderID:        &tokenProviderID,
			Type:                   "OSD",
		},
	}
}

func (s *ClusterControllerTestSuite) TestShow() {

	s.T().Run("ok", func(t *testing.T) {
		// given
		sa := &authtestsupport.Identity{
			Username: authsupport.ToolChainOperator,
			ID:       uuid.NewV4(),
		}
		clusterPayload := newCreateClusterPayload()
		svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)
		resp := test.CreateClustersCreated(t, svc.Context, svc, ctrl, &clusterPayload)
		location := resp.Header().Get("location")
		require.NotEmpty(t, location)
		// when accessing the created cluster with another identity
		sa = &authtestsupport.Identity{
			Username: authsupport.Auth,
			ID:       uuid.NewV4(),
		}
		svc, ctrl = s.newSecuredControllerWithServiceAccount(sa)
		splits := strings.Split(location, "/")
		clusterID, err := uuid.FromString(splits[len(splits)-1])
		require.NoError(t, err)
		_, result := test.ShowClustersOK(t, svc.Context, svc, ctrl, clusterID)
		// then
		require.NotNil(t, result)
		require.NotNil(t, result.Data)
		assert.Equal(t, clusterPayload.Data.Name, result.Data.Name)
		name := result.Data.Name
		assert.Equal(t, httpsupport.AddTrailingSlashToURL(clusterPayload.Data.APIURL), result.Data.APIURL)
		assert.Equal(t, httpsupport.AddTrailingSlashToURL(clusterPayload.Data.AppDNS), result.Data.AppDNS)
		assert.Equal(t, false, result.Data.CapacityExhausted)
		assert.Equal(t, fmt.Sprintf("https://console.cluster.%s/console/", name), result.Data.ConsoleURL)
		assert.Equal(t, fmt.Sprintf("https://metrics.cluster.%s/", name), result.Data.MetricsURL)
		assert.Equal(t, fmt.Sprintf("https://console.cluster.%s/console/", name), result.Data.LoggingURL)
		assert.Equal(t, clusterPayload.Data.Type, result.Data.Type)

	})

	s.T().Run("failure", func(t *testing.T) {

		t.Run("not found", func(t *testing.T) {
			// given
			sa := &authtestsupport.Identity{
				Username: authsupport.Auth,
				ID:       uuid.NewV4(),
			}
			svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)
			// when/then
			test.ShowClustersNotFound(t, svc.Context, svc, ctrl, uuid.NewV4())
		})

		t.Run("not allowed", func(t *testing.T) {
			// given
			sa := &authtestsupport.Identity{
				Username: "foo",
				ID:       uuid.NewV4(),
			}
			svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)
			// when/then
			test.ShowClustersUnauthorized(t, svc.Context, svc, ctrl, uuid.NewV4())
		})
	})

}
func (s *ClusterControllerTestSuite) TestShowForAuthClient() {

	s.T().Run("ok", func(t *testing.T) {
		// given
		sa := &authtestsupport.Identity{
			Username: authsupport.ToolChainOperator,
			ID:       uuid.NewV4(),
		}
		clusterPayload := newCreateClusterPayload()
		svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)
		resp := test.CreateClustersCreated(t, svc.Context, svc, ctrl, &clusterPayload)
		location := resp.Header().Get("location")
		require.NotEmpty(t, location)
		// when accessing the created cluster with another identity
		sa = &authtestsupport.Identity{
			Username: authsupport.Auth,
			ID:       uuid.NewV4(),
		}
		svc, ctrl = s.newSecuredControllerWithServiceAccount(sa)
		splits := strings.Split(location, "/")
		clusterID, err := uuid.FromString(splits[len(splits)-1])
		require.NoError(t, err)
		_, result := test.ShowForAuthClientClustersOK(t, svc.Context, svc, ctrl, clusterID)
		// then
		require.NotNil(t, result)
		require.NotNil(t, result.Data)
		assert.Equal(t, clusterPayload.Data.Name, result.Data.Name)
		name := result.Data.Name
		assert.Equal(t, httpsupport.AddTrailingSlashToURL(clusterPayload.Data.APIURL), result.Data.APIURL)
		assert.Equal(t, httpsupport.AddTrailingSlashToURL(clusterPayload.Data.AppDNS), result.Data.AppDNS)
		assert.Equal(t, false, result.Data.CapacityExhausted)
		assert.Equal(t, fmt.Sprintf("https://console.cluster.%s/console/", name), result.Data.ConsoleURL)
		assert.Equal(t, fmt.Sprintf("https://metrics.cluster.%s/", name), result.Data.MetricsURL)
		assert.Equal(t, fmt.Sprintf("https://console.cluster.%s/console/", name), result.Data.LoggingURL)
		assert.Equal(t, clusterPayload.Data.Type, result.Data.Type)
		assert.Equal(t, clusterPayload.Data.AuthClientDefaultScope, result.Data.AuthClientDefaultScope)
		assert.Equal(t, clusterPayload.Data.AuthClientID, result.Data.AuthClientID)
		assert.Equal(t, clusterPayload.Data.AuthClientSecret, result.Data.AuthClientSecret)
		assert.Equal(t, clusterPayload.Data.ServiceAccountToken, result.Data.ServiceAccountToken)
		assert.Equal(t, clusterPayload.Data.ServiceAccountUsername, result.Data.ServiceAccountUsername)
	})

	s.T().Run("failure", func(t *testing.T) {

		t.Run("not found", func(t *testing.T) {
			// given
			sa := &authtestsupport.Identity{
				Username: authsupport.Auth,
				ID:       uuid.NewV4(),
			}
			svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)
			// when/then
			test.ShowForAuthClientClustersNotFound(t, svc.Context, svc, ctrl, uuid.NewV4())
		})

		t.Run("not allowed", func(t *testing.T) {
			// given
			sa := &authtestsupport.Identity{
				Username: auth.Tenant,
				ID:       uuid.NewV4(),
			}
			svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)
			// when/then
			test.ShowForAuthClientClustersUnauthorized(t, svc.Context, svc, ctrl, uuid.NewV4())
		})
	})

}
