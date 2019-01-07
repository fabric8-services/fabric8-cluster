package controller_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-cluster/app"

	"github.com/stretchr/testify/assert"

	"github.com/fabric8-services/fabric8-cluster/app/test"
	"github.com/fabric8-services/fabric8-cluster/controller"
	. "github.com/fabric8-services/fabric8-cluster/controller"
	"github.com/fabric8-services/fabric8-cluster/gormtestsupport"
	testsupport "github.com/fabric8-services/fabric8-cluster/test"
	"github.com/fabric8-services/fabric8-common/auth"
	authsupport "github.com/fabric8-services/fabric8-common/auth"
	"github.com/fabric8-services/fabric8-common/httpsupport"
	authtestsupport "github.com/fabric8-services/fabric8-common/test/auth"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ClustersControllerTestSuite struct {
	gormtestsupport.DBTestSuite
}

func TestClustersController(t *testing.T) {
	suite.Run(t, &ClustersControllerTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *ClustersControllerTestSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	// save clusters from config in DB
	err := s.Application.ClusterService().CreateOrSaveClusterFromConfig(context.Background())
	require.NoError(s.T(), err)
}

func (s *ClustersControllerTestSuite) newSecuredControllerWithServiceAccount(serviceAccount *authtestsupport.Identity) (*goa.Service, *controller.ClustersController) {
	svc, err := authtestsupport.ServiceAsServiceAccountUser("Token-Service", serviceAccount)
	require.NoError(s.T(), err)
	return svc, NewClustersController(svc, s.Application)
}

func (s *ClustersControllerTestSuite) TestShow() {

	// given
	sa := &authtestsupport.Identity{
		Username: authsupport.ToolChainOperator,
		ID:       uuid.NewV4(),
	}
	clusterPayload := newCreateClusterPayload()
	svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)
	resp := test.CreateClustersCreated(s.T(), svc.Context, svc, ctrl, &clusterPayload)
	location := resp.Header().Get("location")
	require.NotEmpty(s.T(), location)
	splits := strings.Split(location, "/")
	clusterID, err := uuid.FromString(splits[len(splits)-1])
	require.NoError(s.T(), err)

	s.T().Run("ok", func(t *testing.T) {
		for _, saName := range []string{"fabric8-oso-proxy", "fabric8-tenant", "fabric8-jenkins-idler", "fabric8-jenkins-proxy", "fabric8-auth"} {
			t.Run(saName, func(t *testing.T) {
				// when accessing the created cluster with another identity
				sa = &authtestsupport.Identity{
					Username: saName,
					ID:       uuid.NewV4(),
				}
				svc, ctrl = s.newSecuredControllerWithServiceAccount(sa)
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
		}
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
func (s *ClustersControllerTestSuite) TestShowForAuthClient() {

	// given
	sa := &authtestsupport.Identity{
		Username: authsupport.ToolChainOperator,
		ID:       uuid.NewV4(),
	}
	clusterPayload := newCreateClusterPayload()
	svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)
	resp := test.CreateClustersCreated(s.T(), svc.Context, svc, ctrl, &clusterPayload)
	location := resp.Header().Get("location")
	require.NotEmpty(s.T(), location)
	splits := strings.Split(location, "/")
	clusterID, err := uuid.FromString(splits[len(splits)-1])
	require.NoError(s.T(), err)

	s.T().Run("ok", func(t *testing.T) {
		// when accessing the created cluster with another identity
		sa = &authtestsupport.Identity{
			Username: authsupport.Auth,
			ID:       uuid.NewV4(),
		}
		svc, ctrl = s.newSecuredControllerWithServiceAccount(sa)
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
			test.ShowForAuthClientClustersUnauthorized(t, svc.Context, svc, ctrl, clusterID)
		})
	})

}

func (s *ClustersControllerTestSuite) TestList() {

	require.NotEmpty(s.T(), s.Configuration.GetClusters())
	// also add an extra cluster in the DB, to be returned by the endpoint, along with clusters from config file
	extra := testsupport.CreateCluster(s.T(), s.DB)

	s.T().Run("ok", func(t *testing.T) {
		for _, saName := range []string{"fabric8-oso-proxy", "fabric8-tenant", "fabric8-jenkins-idler", "fabric8-jenkins-proxy", "fabric8-auth"} {
			t.Run(saName, func(t *testing.T) {
				// given
				sa := &authtestsupport.Identity{
					Username: saName,
					ID:       uuid.NewV4(),
				}
				svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)
				// when
				_, clusters := test.ListClustersOK(t, svc.Context, svc, ctrl)
				// then
				require.NotNil(t, clusters)
				require.NotNil(t, clusters.Data)
				require.Len(t, clusters.Data, len(s.Configuration.GetClusters())+1)

				for _, c := range clusters.Data {
					if c.Name == extra.Name {
						testsupport.AssertEqualClusterData(t, extra, c)
						continue
					}
					configCluster := s.Configuration.GetClusterByURL(c.APIURL)
					require.NotNil(t, configCluster)
					assert.Equal(t, configCluster.Name, c.Name)
					assert.Equal(t, httpsupport.AddTrailingSlashToURL(configCluster.APIURL), c.APIURL)
					assert.Equal(t, httpsupport.AddTrailingSlashToURL(configCluster.ConsoleURL), c.ConsoleURL)
					assert.Equal(t, httpsupport.AddTrailingSlashToURL(configCluster.MetricsURL), c.MetricsURL)
					assert.Equal(t, httpsupport.AddTrailingSlashToURL(configCluster.LoggingURL), c.LoggingURL)
					assert.Equal(t, configCluster.AppDNS, c.AppDNS)
					assert.Equal(t, configCluster.Type, c.Type)
					assert.Equal(t, configCluster.CapacityExhausted, c.CapacityExhausted)
				}
			})
		}
	})

	s.T().Run("failures", func(t *testing.T) {

		t.Run("unauthorized", func(t *testing.T) {
			// given
			sa := &authtestsupport.Identity{
				Username: "unknown-sa",
				ID:       uuid.NewV4(),
			}
			svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)
			// when/then
			test.ListClustersUnauthorized(s.T(), svc.Context, svc, ctrl)
		})
	})
}

func (s *ClustersControllerTestSuite) TestListForAuth() {
	// given
	require.NotEmpty(s.T(), s.Configuration.GetClusters())

	s.T().Run("authorized", func(t *testing.T) {

		t.Run("fabric8-auth", func(t *testing.T) {
			sa := &authtestsupport.Identity{
				Username: "fabric8-auth",
				ID:       uuid.NewV4(),
			}
			svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)
			_, clusters := test.ListForAuthClientClustersOK(t, svc.Context, svc, ctrl)
			require.NotNil(t, clusters)
			require.NotNil(t, clusters.Data)
			require.Equal(t, len(s.Configuration.GetClusters()), len(clusters.Data))
			for _, cluster := range clusters.Data {
				t.Logf("checking cluster '%s' (%s)", cluster.Name, cluster.APIURL)
				configCluster := s.Configuration.GetClusterByURL(cluster.APIURL)
				require.NotNil(t, configCluster)
				assert.Equal(t, configCluster.Name, cluster.Name)
				assert.Equal(t, httpsupport.AddTrailingSlashToURL(configCluster.APIURL), cluster.APIURL)
				assert.Equal(t, httpsupport.AddTrailingSlashToURL(configCluster.ConsoleURL), cluster.ConsoleURL)
				assert.Equal(t, httpsupport.AddTrailingSlashToURL(configCluster.MetricsURL), cluster.MetricsURL)
				assert.Equal(t, httpsupport.AddTrailingSlashToURL(configCluster.LoggingURL), cluster.LoggingURL)
				assert.Equal(t, configCluster.AppDNS, cluster.AppDNS)
				assert.Equal(t, configCluster.Type, cluster.Type)
				assert.Equal(t, configCluster.CapacityExhausted, cluster.CapacityExhausted)
				assert.Equal(t, configCluster.AuthClientDefaultScope, cluster.AuthClientDefaultScope)
				assert.Equal(t, configCluster.AuthClientID, cluster.AuthClientID)
				assert.Equal(t, configCluster.AuthClientSecret, cluster.AuthClientSecret)
				assert.Equal(t, configCluster.ServiceAccountToken, cluster.ServiceAccountToken)
				assert.Equal(t, configCluster.ServiceAccountUsername, cluster.ServiceAccountUsername)
				assert.Equal(t, configCluster.TokenProviderID, cluster.TokenProviderID)
			}
		})
	})

	s.T().Run("unauthorized", func(t *testing.T) {
		t.Run("fabric8-tenant", func(t *testing.T) {
			sa := &authtestsupport.Identity{
				Username: "fabric8-tenant",
				ID:       uuid.NewV4(),
			}
			svc, ctrl := s.newSecuredControllerWithServiceAccount(sa)
			test.ListForAuthClientClustersUnauthorized(s.T(), svc.Context, svc, ctrl)
		})
	})
}

func createLinkIdentityClusterPayload(clusterURL, identityID string, ignoreIfExists *bool) *app.LinkIdentityToClusterData {
	return &app.LinkIdentityToClusterData{ClusterURL: clusterURL, IdentityID: identityID, IgnoreIfAlreadyExists: ignoreIfExists}
}

func (s *ClustersControllerTestSuite) TestCreate() {

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

func (s *ClustersControllerTestSuite) TestLinkIdentityClusters() {

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

func (s *ClustersControllerTestSuite) TestRemoveIdentityToClustersLink() {

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
