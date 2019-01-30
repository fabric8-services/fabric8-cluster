package controller_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-common/errors"

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

func (s *ClustersControllerTestSuite) newSecuredControllerWithServiceAccount(username string) (*goa.Service, *controller.ClustersController) {
	svc, err := authtestsupport.ServiceAsServiceAccountUser("Token-Service", &authtestsupport.Identity{
		Username: username,
		ID:       uuid.NewV4(),
	})
	require.NoError(s.T(), err)
	return svc, NewClustersController(svc, s.Application)
}

func (s *ClustersControllerTestSuite) TestShow() {

	// given
	clusterPayload := newCreateClusterPayload()
	svc, ctrl := s.newSecuredControllerWithServiceAccount(authsupport.ToolChainOperator)
	resp := test.CreateClustersCreated(s.T(), svc.Context, svc, ctrl, &clusterPayload)
	location := resp.Header().Get("location")
	require.NotEmpty(s.T(), location)
	splits := strings.Split(location, "/")
	clusterID, err := uuid.FromString(splits[len(splits)-1])
	require.NoError(s.T(), err)

	s.T().Run("ok", func(t *testing.T) {
		for _, username := range []string{auth.OsoProxy, auth.Tenant, auth.JenkinsIdler, auth.JenkinsProxy, auth.Auth} {
			t.Run(username, func(t *testing.T) {
				// when accessing the created cluster with another identity
				svc, ctrl = s.newSecuredControllerWithServiceAccount(username)
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

	s.T().Run("failures", func(t *testing.T) {

		t.Run("not found", func(t *testing.T) {
			// given
			svc, ctrl := s.newSecuredControllerWithServiceAccount(authsupport.Auth)
			// when/then
			test.ShowClustersNotFound(t, svc.Context, svc, ctrl, uuid.NewV4())
		})

		t.Run("unauthorized", func(t *testing.T) {
			// given
			svc, ctrl := s.newSecuredControllerWithServiceAccount("foo")
			// when/then
			test.ShowClustersUnauthorized(t, svc.Context, svc, ctrl, uuid.NewV4())
		})
	})
}

func (s *ClustersControllerTestSuite) TestShowForAuthClient() {

	// given
	c := testsupport.CreateCluster(s.T(), s.DB)

	s.T().Run("ok", func(t *testing.T) {
		// when accessing the created cluster with another identity
		svc, ctrl := s.newSecuredControllerWithServiceAccount(authsupport.Auth)
		_, result := test.ShowForAuthClientClustersOK(t, svc.Context, svc, ctrl, c.ClusterID)
		// then
		require.NotNil(t, result)
		require.NotNil(t, result.Data)
		testsupport.AssertEqualFullClusterData(t, c, *result.Data)
	})

	s.T().Run("failures", func(t *testing.T) {

		t.Run("not found", func(t *testing.T) {
			// given
			svc, ctrl := s.newSecuredControllerWithServiceAccount(authsupport.Auth)
			// when/then
			test.ShowForAuthClientClustersNotFound(t, svc.Context, svc, ctrl, uuid.NewV4())
		})

		t.Run("unauthorized", func(t *testing.T) {
			for _, username := range []string{auth.OsoProxy, auth.Tenant, auth.JenkinsIdler, auth.JenkinsProxy, "other"} {
				t.Run(username, func(t *testing.T) {
					// given
					svc, ctrl := s.newSecuredControllerWithServiceAccount(username)
					// when/then
					test.ShowForAuthClientClustersUnauthorized(t, svc.Context, svc, ctrl, c.ClusterID)
				})
			}
		})
	})
}

func (s *ClustersControllerTestSuite) TestFindByURL() {

	// given
	c := testsupport.CreateCluster(s.T(), s.DB)

	s.T().Run("ok", func(t *testing.T) {
		for _, username := range []string{auth.OsoProxy, auth.Tenant, auth.JenkinsIdler, auth.JenkinsProxy, auth.Auth} {
			t.Run(username, func(t *testing.T) {
				// when accessing the created cluster with another identity
				svc, ctrl := s.newSecuredControllerWithServiceAccount(username)
				_, result := test.FindByURLClustersOK(t, svc.Context, svc, ctrl, c.URL)
				// then
				require.NotNil(t, result)
				require.NotNil(t, result.Data)
				assert.Equal(t, c.Name, result.Data.Name)
				assert.Equal(t, httpsupport.AddTrailingSlashToURL(c.URL), result.Data.APIURL)
				assert.Equal(t, c.AppDNS, result.Data.AppDNS)
				assert.Equal(t, false, result.Data.CapacityExhausted)
				assert.Equal(t, httpsupport.AddTrailingSlashToURL(c.ConsoleURL), result.Data.ConsoleURL)
				assert.Equal(t, httpsupport.AddTrailingSlashToURL(c.MetricsURL), result.Data.MetricsURL)
				assert.Equal(t, httpsupport.AddTrailingSlashToURL(c.LoggingURL), result.Data.LoggingURL)
				assert.Equal(t, c.Type, result.Data.Type)
			})
		}
	})

	s.T().Run("failures", func(t *testing.T) {

		t.Run("bad request", func(t *testing.T) {
			// given
			svc, ctrl := s.newSecuredControllerWithServiceAccount(authsupport.Auth)
			// when/then
			test.FindByURLClustersBadRequest(t, svc.Context, svc, ctrl, "foo.com")
		})

		t.Run("not found", func(t *testing.T) {
			// given
			svc, ctrl := s.newSecuredControllerWithServiceAccount(authsupport.Auth)
			// when/then
			test.FindByURLClustersNotFound(t, svc.Context, svc, ctrl, "http://foo.com")
		})

		t.Run("unauthorized", func(t *testing.T) {
			// given
			svc, ctrl := s.newSecuredControllerWithServiceAccount("foo")
			// when/then
			test.FindByURLClustersUnauthorized(t, svc.Context, svc, ctrl, c.URL)
		})
	})
}

func (s *ClustersControllerTestSuite) TestFindByURLForAuth() {

	// given
	c := testsupport.CreateCluster(s.T(), s.DB)

	s.T().Run("ok", func(t *testing.T) {
		for _, username := range []string{auth.Auth} {
			t.Run(username, func(t *testing.T) {
				// when accessing the created cluster with another identity
				svc, ctrl := s.newSecuredControllerWithServiceAccount(username)
				_, result := test.FindByURLForAuthClustersOK(t, svc.Context, svc, ctrl, c.URL)
				// then
				require.NotNil(t, result)
				require.NotNil(t, result.Data)
				assert.Equal(t, c.Name, result.Data.Name)
				assert.Equal(t, httpsupport.AddTrailingSlashToURL(c.URL), result.Data.APIURL)
				assert.Equal(t, c.AppDNS, result.Data.AppDNS)
				assert.Equal(t, false, result.Data.CapacityExhausted)
				assert.Equal(t, httpsupport.AddTrailingSlashToURL(c.ConsoleURL), result.Data.ConsoleURL)
				assert.Equal(t, httpsupport.AddTrailingSlashToURL(c.MetricsURL), result.Data.MetricsURL)
				assert.Equal(t, httpsupport.AddTrailingSlashToURL(c.LoggingURL), result.Data.LoggingURL)
				assert.Equal(t, c.Type, result.Data.Type)
			})
		}
	})

	s.T().Run("failures", func(t *testing.T) {

		t.Run("bad request", func(t *testing.T) {
			// given
			svc, ctrl := s.newSecuredControllerWithServiceAccount(authsupport.Auth)
			// when/then
			test.FindByURLForAuthClustersBadRequest(t, svc.Context, svc, ctrl, "foo.com") // missing scheme
		})

		t.Run("not found", func(t *testing.T) {
			// given
			svc, ctrl := s.newSecuredControllerWithServiceAccount(authsupport.Auth)
			// when/then
			test.FindByURLForAuthClustersNotFound(t, svc.Context, svc, ctrl, "http://foo.com")
		})

		t.Run("unauthorized", func(t *testing.T) {
			for _, username := range []string{auth.OsoProxy, auth.Tenant, auth.JenkinsIdler, auth.JenkinsProxy, "foo"} {
				t.Run(username, func(t *testing.T) {
					// given
					svc, ctrl := s.newSecuredControllerWithServiceAccount(username)
					// when/then
					test.FindByURLForAuthClustersUnauthorized(t, svc.Context, svc, ctrl, c.URL)
				})
			}
		})
	})
}

func (s *ClustersControllerTestSuite) TestList() {

	require.NotEmpty(s.T(), s.Configuration.GetClusters())
	// also add an extra cluster in the DB, to be returned by the endpoint, along with clusters from config file
	testsupport.CreateCluster(s.T(), s.DB)

	s.T().Run("ok", func(t *testing.T) {
		for _, username := range []string{auth.OsoProxy, auth.Tenant, auth.JenkinsIdler, auth.JenkinsProxy, auth.Auth} {
			t.Run(username, func(t *testing.T) {
				// given
				svc, ctrl := s.newSecuredControllerWithServiceAccount(username)
				// when
				_, result := test.ListClustersOK(t, svc.Context, svc, ctrl, nil)
				// then
				require.NotNil(t, result)
				require.NotNil(t, result.Data)
				expected, err := s.Application.ClusterService().List(svc.Context, nil) // also needs SA in context to list the expected clusters
				require.NoError(t, err)
				testsupport.AssertEqualClustersData(t, expected, result.Data)
			})
		}
	})

	s.T().Run("failures", func(t *testing.T) {

		t.Run("unauthorized", func(t *testing.T) {
			// given
			svc, ctrl := s.newSecuredControllerWithServiceAccount("unknown-sa")
			// when/then
			test.ListClustersUnauthorized(t, svc.Context, svc, ctrl, nil)
		})

		t.Run("bad parameter", func(t *testing.T) {
			for _, username := range []string{auth.Auth} {
				t.Run(username, func(t *testing.T) {
					// given
					svc, ctrl := s.newSecuredControllerWithServiceAccount(username)
					clusterType := "foo"
					// when/then
					test.ListForAuthClientClustersBadRequest(t, svc.Context, svc, ctrl, &clusterType)
				})
			}
		})
	})
}

func (s *ClustersControllerTestSuite) TestListForAuth() {
	// given
	require.NotEmpty(s.T(), s.Configuration.GetClusters())
	// also add an extra cluster in the DB, to be returned by the endpoint, along with clusters from config file
	testsupport.CreateCluster(s.T(), s.DB)

	s.T().Run("ok", func(t *testing.T) {
		t.Run(auth.Auth, func(t *testing.T) {
			// given
			svc, ctrl := s.newSecuredControllerWithServiceAccount(auth.Auth)
			// when
			_, result := test.ListForAuthClientClustersOK(t, svc.Context, svc, ctrl, nil)
			// then
			require.NotNil(t, result)
			require.NotNil(t, result.Data)
			expected, err := s.Application.ClusterService().ListForAuth(svc.Context, nil) // also needs SA in context to list the expected clusters
			require.NoError(t, err)
			testsupport.AssertEqualFullClustersData(t, expected, result.Data)
		})
	})

	s.T().Run("failures", func(t *testing.T) {

		t.Run("unauthorized", func(t *testing.T) {
			for _, username := range []string{auth.OsoProxy, auth.Tenant, auth.JenkinsIdler, auth.JenkinsProxy} {
				t.Run(username, func(t *testing.T) {
					// given
					svc, ctrl := s.newSecuredControllerWithServiceAccount(username)
					// when/then
					test.ListForAuthClientClustersUnauthorized(t, svc.Context, svc, ctrl, nil)
				})
			}
		})

		t.Run("bad parameter", func(t *testing.T) {
			for _, username := range []string{auth.Auth} {
				t.Run(username, func(t *testing.T) {
					// given
					svc, ctrl := s.newSecuredControllerWithServiceAccount(username)
					clusterType := "foo"
					// when/then
					test.ListForAuthClientClustersBadRequest(t, svc.Context, svc, ctrl, &clusterType)
				})
			}
		})
	})
}

func createLinkIdentityClusterPayload(clusterURL, identityID string, ignoreIfExists *bool) *app.LinkIdentityToClusterData {
	return &app.LinkIdentityToClusterData{ClusterURL: clusterURL, IdentityID: identityID, IgnoreIfAlreadyExists: ignoreIfExists}
}

func (s *ClustersControllerTestSuite) TestCreate() {

	s.T().Run("ok", func(t *testing.T) {
		// given
		clusterPayload := newCreateClusterPayload()
		svc, ctrl := s.newSecuredControllerWithServiceAccount(authsupport.ToolChainOperator)
		// when
		resp := test.CreateClustersCreated(t, svc.Context, svc, ctrl, &clusterPayload)
		//then
		location := resp.Header().Get("location")
		require.NotEmpty(t, location)
	})

	s.T().Run("failures", func(t *testing.T) {

		t.Run("unauthorized", func(t *testing.T) {
			for _, username := range []string{auth.OsoProxy, auth.Tenant, auth.JenkinsIdler, auth.JenkinsProxy, "other"} {
				t.Run(username, func(t *testing.T) {
					// given
					clusterPayload := newCreateClusterPayload()
					svc, ctrl := s.newSecuredControllerWithServiceAccount(authsupport.Auth)
					// when/then
					test.CreateClustersUnauthorized(t, svc.Context, svc, ctrl, &clusterPayload)
				})
			}
		})

		t.Run("bad request", func(t *testing.T) {
			// given
			clusterPayload := newCreateClusterPayload()
			clusterPayload.Data.APIURL = " "
			svc, ctrl := s.newSecuredControllerWithServiceAccount(authsupport.ToolChainOperator)
			// when/then
			test.CreateClustersBadRequest(t, svc.Context, svc, ctrl, &clusterPayload)
		})
	})
}

func (s *ClustersControllerTestSuite) TestDelete() {

	s.T().Run("ok", func(t *testing.T) {
		// given
		c := testsupport.CreateCluster(t, s.DB)
		svc, ctrl := s.newSecuredControllerWithServiceAccount(authsupport.ToolChainOperator)
		// when
		test.DeleteClustersNoContent(t, svc.Context, svc, ctrl, c.ClusterID)
		// then
		ctx, err := authtestsupport.EmbedServiceAccountTokenInContext(context.Background(), &authtestsupport.Identity{
			Username: auth.Auth, // need another SA to load the data
			ID:       uuid.NewV4(),
		})
		require.NoError(t, err)
		_, err = s.Application.ClusterService().Load(ctx, c.ClusterID)
		testsupport.AssertError(t, err, errors.NotFoundError{}, errors.NewNotFoundError("cluster", c.ClusterID.String()).Error())
	})

	s.T().Run("failures", func(t *testing.T) {

		t.Run("unauthorized", func(t *testing.T) {
			// given
			c := testsupport.CreateCluster(t, s.DB)
			for _, username := range []string{auth.OsoProxy, auth.Tenant, auth.JenkinsIdler, auth.JenkinsProxy} {
				t.Run(username, func(t *testing.T) {
					// given
					svc, ctrl := s.newSecuredControllerWithServiceAccount(username)
					// when/then
					test.DeleteClustersUnauthorized(t, svc.Context, svc, ctrl, c.ClusterID)
				})
			}
		})

		t.Run("not found", func(t *testing.T) {
			// given
			svc, ctrl := s.newSecuredControllerWithServiceAccount(authsupport.ToolChainOperator)
			// when/then
			test.DeleteClustersNotFound(t, svc.Context, svc, ctrl, uuid.NewV4())
		})
	})
}

func (s *ClustersControllerTestSuite) TestLinkIdentityClusters() {

	s.T().Run("ok", func(t *testing.T) {

		t.Run("ignore if exists - nil", func(t *testing.T) {
			// given
			svc, ctrl := s.newSecuredControllerWithServiceAccount(auth.Auth)
			c := testsupport.CreateCluster(t, s.DB)
			payload := createLinkIdentityClusterPayload(c.URL, uuid.NewV4().String(), nil)
			// when/then
			test.LinkIdentityToClusterClustersNoContent(t, svc.Context, svc, ctrl, payload)
		})

		t.Run("ignore if exists - true", func(t *testing.T) {
			// given
			svc, ctrl := s.newSecuredControllerWithServiceAccount(auth.Auth)
			c := testsupport.CreateCluster(t, s.DB)
			ignore := true
			payload := createLinkIdentityClusterPayload(c.URL, uuid.NewV4().String(), &ignore)
			// when/then
			test.LinkIdentityToClusterClustersNoContent(t, svc.Context, svc, ctrl, payload)
		})

	})

	s.T().Run("failures", func(t *testing.T) {

		t.Run("bad request", func(t *testing.T) {

			t.Run("invalid uuid", func(t *testing.T) {
				// given
				svc, ctrl := s.newSecuredControllerWithServiceAccount(auth.Auth)
				c := testsupport.CreateCluster(t, s.DB)
				payload := createLinkIdentityClusterPayload(c.URL, "foo", nil)
				// when/then
				test.LinkIdentityToClusterClustersBadRequest(t, svc.Context, svc, ctrl, payload)
			})

			t.Run("empty space uuid", func(t *testing.T) {
				// given
				svc, ctrl := s.newSecuredControllerWithServiceAccount(auth.Auth)
				c := testsupport.CreateCluster(t, s.DB)
				payload := createLinkIdentityClusterPayload(c.URL, "  ", nil)
				// when/then
				test.LinkIdentityToClusterClustersBadRequest(t, svc.Context, svc, ctrl, payload)
			})

			t.Run("invalid cluster url", func(t *testing.T) {
				// given
				svc, ctrl := s.newSecuredControllerWithServiceAccount(auth.Auth)
				payload := createLinkIdentityClusterPayload("foo.com", uuid.NewV4().String(), nil)
				// when/then
				test.LinkIdentityToClusterClustersBadRequest(t, svc.Context, svc, ctrl, payload)
			})
		})

		t.Run("not found", func(t *testing.T) {

			t.Run("unknown cluster", func(t *testing.T) {
				// given
				svc, ctrl := s.newSecuredControllerWithServiceAccount(auth.Auth)
				payload := createLinkIdentityClusterPayload("http://foo.com", uuid.NewV4().String(), nil)
				// when/then
				test.LinkIdentityToClusterClustersNotFound(t, svc.Context, svc, ctrl, payload)
			})

		})
	})

	s.T().Run("unauthorized", func(t *testing.T) {
		t.Run("unknown token", func(t *testing.T) {
			// given
			c := testsupport.CreateCluster(t, s.DB)
			payload := createLinkIdentityClusterPayload(c.URL, uuid.NewV4().String(), nil)
			svc, ctrl := s.newSecuredControllerWithServiceAccount("foo")
			// when/then
			test.LinkIdentityToClusterClustersUnauthorized(t, svc.Context, svc, ctrl, payload)
		})
	})

	s.T().Run("internal error - ignore false", func(t *testing.T) {
		// given
		ic := testsupport.CreateIdentityCluster(t, s.DB)
		ignore := false
		payload := createLinkIdentityClusterPayload(ic.Cluster.URL, ic.IdentityID.String(), &ignore)
		svc, ctrl := s.newSecuredControllerWithServiceAccount(auth.Auth)
		// when/then
		test.LinkIdentityToClusterClustersInternalServerError(t, svc.Context, svc, ctrl, payload)
	})
}

func (s *ClustersControllerTestSuite) TestRemoveIdentityToClustersLink() {

	s.T().Run("ok", func(t *testing.T) {
		// given
		svc, ctrl := s.newSecuredControllerWithServiceAccount(auth.Auth)
		ic := testsupport.CreateIdentityCluster(t, s.DB)
		payload := createUnLinkIdentityToClusterData(ic.Cluster.URL, ic.IdentityID.String())
		// when/then
		test.RemoveIdentityToClusterLinkClustersNoContent(t, svc.Context, svc, ctrl, payload)
	})

	s.T().Run("failures", func(t *testing.T) {

		// given
		ic := testsupport.CreateIdentityCluster(t, s.DB)
		svc, ctrl := s.newSecuredControllerWithServiceAccount(auth.Auth)

		t.Run("not found", func(t *testing.T) {

			t.Run("different identity", func(t *testing.T) {
				// given
				payload := createUnLinkIdentityToClusterData(ic.Cluster.URL, uuid.NewV4().String())
				// when/then
				test.RemoveIdentityToClusterLinkClustersNotFound(t, svc.Context, svc, ctrl, payload)
			})

			t.Run("unknown cluster", func(t *testing.T) {
				// given
				payload := createUnLinkIdentityToClusterData("http://foo.com", ic.IdentityID.String())
				// when/then
				test.RemoveIdentityToClusterLinkClustersNotFound(t, svc.Context, svc, ctrl, payload)
			})
		})

		t.Run("bad request", func(t *testing.T) {

			t.Run("empty space uuid", func(t *testing.T) {
				// given
				payload := createUnLinkIdentityToClusterData(ic.Cluster.URL, "  ")
				// when/then
				test.RemoveIdentityToClusterLinkClustersBadRequest(t, svc.Context, svc, ctrl, payload)
			})

			t.Run("invalid cluster url", func(t *testing.T) {
				// given
				payload := createUnLinkIdentityToClusterData("foo.com", ic.IdentityID.String())

				// when/then
				test.RemoveIdentityToClusterLinkClustersBadRequest(t, svc.Context, svc, ctrl, payload)
			})
		})

		t.Run("unauthorized", func(t *testing.T) {

			t.Run("unknown token", func(t *testing.T) {
				// given
				svc, ctrl := s.newSecuredControllerWithServiceAccount("foo")
				payload := createUnLinkIdentityToClusterData(ic.Cluster.URL, ic.IdentityID.String())
				// when/then
				test.RemoveIdentityToClusterLinkClustersUnauthorized(t, svc.Context, svc, ctrl, payload)
			})
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
