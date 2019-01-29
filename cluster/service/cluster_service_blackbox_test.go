package service_test

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-common/auth"

	"github.com/fabric8-services/fabric8-cluster/gormapplication"

	"github.com/fabric8-services/fabric8-cluster/cluster"
	"github.com/fabric8-services/fabric8-cluster/cluster/repository"
	"github.com/fabric8-services/fabric8-cluster/configuration"
	"github.com/fabric8-services/fabric8-cluster/gormtestsupport"
	"github.com/fabric8-services/fabric8-cluster/test"
	"github.com/fabric8-services/fabric8-common/errors"
	testsupport "github.com/fabric8-services/fabric8-common/test"
	authtestsupport "github.com/fabric8-services/fabric8-common/test/auth"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestClusterService(t *testing.T) {
	suite.Run(t, &ClusterServiceTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

type ClusterServiceTestSuite struct {
	gormtestsupport.DBTestSuite
}

func (s *ClusterServiceTestSuite) TestCreateOrSaveClusterFromConfigOK() {
	// when
	err := s.Application.ClusterService().CreateOrSaveClusterFromConfig(context.Background())
	// then
	require.NoError(s.T(), err)
	// lookup OSO clusters
	osoClusters, err := s.Application.Clusters().Query(func(db *gorm.DB) *gorm.DB {
		return db.Where("type = ?", cluster.OSO)
	})
	require.NoError(s.T(), err)
	assert.Len(s.T(), osoClusters, 3)
	// lookup OSD cluster
	osdClusters, err := s.Application.Clusters().Query(func(db *gorm.DB) *gorm.DB {
		return db.Where("type = ?", cluster.OSD)
	})
	require.NoError(s.T(), err)
	assert.Len(s.T(), osdClusters, 1)
	// verify all records
	verifyClusters(s.T(), s.Configuration.GetClusters(), append(osoClusters, osdClusters...), true)
}

func (s *ClusterServiceTestSuite) TestCreateOrSaveCluster() {

	s.T().Run("create", func(t *testing.T) {

		t.Run("valid with missing URLs", func(t *testing.T) {
			// given
			c := newTestCluster()
			name := c.Name
			c.ConsoleURL = " "
			c.LoggingURL = " "
			c.MetricsURL = " "
			c.TokenProviderID = " "
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			// then
			require.NoError(t, err)
			assert.NotNil(t, c.ClusterID)
			assert.Equal(t, name, c.Name)
			assert.Equal(t, cluster.OCP, c.Type)
			assert.Equal(t, fmt.Sprintf("https://cluster.%s/", name), c.AppDNS)
			assert.Equal(t, fmt.Sprintf("https://api.cluster.%s/", name), c.URL)
			assert.Equal(t, true, c.CapacityExhausted)
			assert.Equal(t, "ServiceAccountToken", c.SAToken)
			assert.Equal(t, "ServiceAccountUsername", c.SAUsername)
			assert.Equal(t, "AuthClientID", c.AuthClientID)
			assert.Equal(t, "AuthClientSecret", c.AuthClientSecret)
			assert.Equal(t, "AuthClientDefaultScope", c.AuthDefaultScope)
			// optional fields: generated values with a trailing slash
			assert.Equal(t, fmt.Sprintf("https://console.cluster.%s/console/", name), c.ConsoleURL)
			assert.Equal(t, fmt.Sprintf("https://metrics.cluster.%s/", name), c.MetricsURL)
			assert.Equal(t, fmt.Sprintf("https://console.cluster.%s/console/", name), c.LoggingURL)
			assert.Equal(t, c.ClusterID.String(), c.TokenProviderID)
		})

		t.Run("valid with all URLs", func(t *testing.T) {
			// given
			c := newTestCluster()
			name := c.Name
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			// then
			require.NoError(t, err)
			assert.NotNil(t, c.ClusterID)
			assert.Equal(t, name, c.Name)
			assert.Equal(t, cluster.OCP, c.Type)
			assert.Equal(t, fmt.Sprintf("https://cluster.%s/", name), c.AppDNS)
			assert.Equal(t, fmt.Sprintf("https://api.cluster.%s/", name), c.URL)
			assert.Equal(t, true, c.CapacityExhausted)
			assert.Equal(t, "ServiceAccountToken", c.SAToken)
			assert.Equal(t, "ServiceAccountUsername", c.SAUsername)
			assert.Equal(t, "AuthClientID", c.AuthClientID)
			assert.Equal(t, "AuthClientSecret", c.AuthClientSecret)
			assert.Equal(t, "AuthClientDefaultScope", c.AuthDefaultScope)
			// optional fields: keep provided values, but with a trailing slash
			assert.Equal(t, fmt.Sprintf("https://console.cluster.%s/", name), c.ConsoleURL)
			assert.Equal(t, fmt.Sprintf("https://metrics.cluster.%s/", name), c.MetricsURL)
			assert.Equal(t, fmt.Sprintf("https://logging.cluster.%s/", name), c.LoggingURL)
			assert.Equal(t, "TokenProviderID", c.TokenProviderID)
		})
	})

	s.T().Run("save existing", func(t *testing.T) {

		t.Run("with updated TokenProviderID", func(t *testing.T) {
			// given an existing cluster
			c := newTestCluster()
			require.Equal(t, uuid.Nil, c.ClusterID)
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			require.NoError(t, err)
			t.Logf("created cluster ID: %v", c.ClusterID)
			require.NotEqual(t, uuid.Nil, c.ClusterID)
			// when updating with an updated TokenProviderID value
			reloaded, err := s.Application.Clusters().LoadByURL(context.Background(), c.URL)
			require.NoError(t, err)
			reloaded.TokenProviderID = "UpdatedTokenProviderID"
			err = s.Application.ClusterService().CreateOrSaveCluster(context.Background(), reloaded)
			// then
			require.NoError(t, err)
			// read again from DB
			updated, err := s.Application.Clusters().LoadByURL(context.Background(), reloaded.URL)
			require.NoError(t, err)
			assert.Equal(t, c.ClusterID, updated.ClusterID)
			assert.Equal(t, "UpdatedTokenProviderID", updated.TokenProviderID)
		})

		t.Run("with missing TokenProviderID", func(t *testing.T) {
			// given an existing cluster
			c := newTestCluster()
			require.Equal(t, uuid.Nil, c.ClusterID)
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			require.NoError(t, err)
			t.Logf("created cluster ID: %v", c.ClusterID)
			require.NotEqual(t, uuid.Nil, c.ClusterID)
			// when updating without any TokenProviderID value
			reloaded, err := s.Application.Clusters().LoadByURL(context.Background(), c.URL)
			require.NoError(t, err)
			reloaded.TokenProviderID = ""
			err = s.Application.ClusterService().CreateOrSaveCluster(context.Background(), reloaded)
			// then
			require.NoError(t, err)
			// read again from DB
			updated, err := s.Application.Clusters().LoadByURL(context.Background(), reloaded.URL)
			require.NoError(t, err)
			assert.Equal(t, c.ClusterID, updated.ClusterID)
			// expect TokenProviderID to be equal to old value
			assert.Equal(t, c.TokenProviderID, updated.TokenProviderID)
		})

		t.Run("without trailing slashed in updated URLs", func(t *testing.T) {
			// given an existing cluster
			c := newTestCluster()
			require.Equal(t, uuid.Nil, c.ClusterID)
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			require.NoError(t, err)
			t.Logf("created cluster ID: %v", c.ClusterID)
			require.NotEqual(t, uuid.Nil, c.ClusterID)
			// when updating with an updated TokenProviderID value
			reloaded, err := s.Application.Clusters().LoadByURL(context.Background(), c.URL)
			require.NoError(t, err)
			reloaded.ConsoleURL = "https://console.cluster.com/console"
			reloaded.MetricsURL = "https://metrics.cluster.com"
			reloaded.LoggingURL = "https://console.cluster.com/console"
			err = s.Application.ClusterService().CreateOrSaveCluster(context.Background(), reloaded)
			// then
			require.NoError(t, err)
			// read again from DB
			updated, err := s.Application.Clusters().LoadByURL(context.Background(), reloaded.URL)
			require.NoError(t, err)
			assert.Equal(t, c.ClusterID, updated.ClusterID)
			assert.Equal(t, "https://console.cluster.com/console/", updated.ConsoleURL)
			assert.Equal(t, "https://metrics.cluster.com/", updated.MetricsURL)
			assert.Equal(t, "https://console.cluster.com/console/", updated.LoggingURL)
		})

		t.Run("with empty updated URLs", func(t *testing.T) {
			// given an existing cluster
			c := newTestCluster()
			require.Equal(t, uuid.Nil, c.ClusterID)
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			require.NoError(t, err)
			t.Logf("created cluster ID: %v", c.ClusterID)
			require.NotEqual(t, uuid.Nil, c.ClusterID)
			// when updating with an updated TokenProviderID value
			reloaded, err := s.Application.Clusters().LoadByURL(context.Background(), c.URL)
			require.NoError(t, err)
			reloaded.ConsoleURL = ""
			reloaded.MetricsURL = ""
			reloaded.LoggingURL = ""
			err = s.Application.ClusterService().CreateOrSaveCluster(context.Background(), reloaded)
			// then
			require.NoError(t, err)
			// read again from DB
			updated, err := s.Application.Clusters().LoadByURL(context.Background(), reloaded.URL)
			require.NoError(t, err)
			// console, metrics and logging URLs should be set based on the cluster URL itself (including a trailing slash)
			assert.Equal(t, c.ClusterID, updated.ClusterID)
			consoleURL, err := repository.ConvertAPIURL(c.URL, "console", "console/")
			require.NoError(t, err)
			assert.Equal(t, consoleURL, updated.ConsoleURL)
			metricsURL, err := repository.ConvertAPIURL(c.URL, "metrics", "/")
			require.NoError(t, err)
			assert.Equal(t, metricsURL, updated.MetricsURL)
			loggingURL, err := repository.ConvertAPIURL(c.URL, "console", "console/")
			require.NoError(t, err)
			assert.Equal(t, loggingURL, updated.LoggingURL)
		})

	})

	s.T().Run("invalid", func(t *testing.T) {

		t.Run("empty name", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.Name = " "
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			// then
			testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': empty field '%s' is not allowed", c.Name, "name"))
		})

		t.Run("empty service-account-token", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.SAToken = " "
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			// then
			testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': empty field '%s' is not allowed", c.Name, "service-account-token"))
		})

		t.Run("empty service-account-username", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.SAUsername = " "
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			// then
			testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': empty field '%s' is not allowed", c.Name, "service-account-username"))
		})

		t.Run("auth-client-id", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.AuthClientID = " "
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			// then
			testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': empty field '%s' is not allowed", c.Name, "auth-client-id"))
		})

		t.Run("token-provider-id", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.AuthClientSecret = " "
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			// then
			testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': empty field '%s' is not allowed", c.Name, "auth-client-secret"))
		})

		t.Run("auth-client-default-scope", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.AuthDefaultScope = " "
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			// then
			testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': empty field '%s' is not allowed", c.Name, "auth-client-default-scope"))
		})

		t.Run("invalid API URL", func(t *testing.T) {

			t.Run("empty", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.URL = " "
				// when
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
				// then
				testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': '%s' URL '%s' is invalid: missing scheme or host", c.Name, "API", c.URL))
			})

			t.Run("missing scheme", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.URL = "api.cluster.com"
				// when
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
				// then
				testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': '%s' URL '%s' is invalid: missing scheme or host", c.Name, "API", c.URL))
			})

			t.Run("missing host", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.URL = "https://"
				// when
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
				// then
				testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': '%s' URL '%s' is invalid: missing scheme or host", c.Name, "API", c.URL))
			})
		})

		t.Run("invalid console URL", func(t *testing.T) {

			t.Run("missing scheme", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.ConsoleURL = "console.cluster-foo.com"
				// when
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
				// then
				testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': '%s' URL '%s' is invalid: missing scheme or host", c.Name, "console", c.ConsoleURL))
			})

			t.Run("missing host", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.ConsoleURL = "https://"
				// when
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
				// then
				testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': '%s' URL '%s' is invalid: missing scheme or host", c.Name, "console", c.ConsoleURL))
			})

		})

		t.Run("invalid logging URL", func(t *testing.T) {

			t.Run("missing scheme", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.LoggingURL = "logging.cluster-foo.com"
				// when
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
				// then
				testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': '%s' URL '%s' is invalid: missing scheme or host", c.Name, "logging", c.LoggingURL))
			})

			t.Run("missing host", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.LoggingURL = "https://"
				// when
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
				// then
				testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': '%s' URL '%s' is invalid: missing scheme or host", c.Name, "logging", c.LoggingURL))
			})

		})

		t.Run("invalid metrics URL", func(t *testing.T) {

			t.Run("missing scheme", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.MetricsURL = "metrics.cluster-foo.com"
				// when
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
				// then
				testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': '%s' URL '%s' is invalid: missing scheme or host", c.Name, "metrics", c.MetricsURL))
			})

			t.Run("missing host", func(t *testing.T) {
				// given
				c := newTestCluster()
				c.MetricsURL = "https://"
				// when
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
				// then
				testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': '%s' URL '%s' is invalid: missing scheme or host", c.Name, "metrics", c.MetricsURL))
			})

		})

		t.Run("invalid type", func(t *testing.T) {
			// given
			c := newTestCluster()
			c.Type = "FOO"
			// when
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			// then
			testsupport.AssertError(t, err, errors.BadParameterError{}, fmt.Sprintf("failed to create or save cluster named '%s': invalid type of cluster: '%s' (expected 'OSO', 'OCP' or 'OSD')", c.Name, c.Type))
		})
	})
}

func (s *ClusterServiceTestSuite) TestLoad() {

	s.T().Run("ok", func(t *testing.T) {

		for _, saName := range []string{"fabric8-oso-proxy", "fabric8-tenant", "fabric8-jenkins-idler", "fabric8-jenkins-proxy", "fabric8-auth"} {
			t.Run(saName, func(t *testing.T) {
				// given
				c := newTestCluster()
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
				require.NoError(t, err)
				sa := &authtestsupport.Identity{
					Username: saName,
					ID:       uuid.NewV4(),
				}
				ctx, err := authtestsupport.EmbedServiceAccountTokenInContext(context.Background(), sa)
				require.NoError(t, err)
				// when
				result, err := s.Application.ClusterService().Load(ctx, c.ClusterID)
				// then
				require.NoError(t, err)
				require.NotNil(t, result)
				test.AssertEqualCluster(t, *c, *result, false)
			})
		}
	})

	s.T().Run("failures", func(t *testing.T) {

		t.Run("unauthorized", func(t *testing.T) {
			// given
			c := newTestCluster()
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			require.NoError(t, err)
			sa := &authtestsupport.Identity{
				Username: "foo",
				ID:       uuid.NewV4(),
			}
			ctx, err := authtestsupport.EmbedServiceAccountTokenInContext(context.Background(), sa)
			require.NoError(t, err)
			// when
			_, err = s.Application.ClusterService().Load(ctx, c.ClusterID)
			// then
			testsupport.AssertError(t, err, errors.UnauthorizedError{}, "unauthorized access to cluster info")
		})

		t.Run("not found", func(t *testing.T) {
			// given
			c := newTestCluster()
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			require.NoError(t, err)
			sa := &authtestsupport.Identity{
				Username: "fabric8-auth",
				ID:       uuid.NewV4(),
			}
			ctx, err := authtestsupport.EmbedServiceAccountTokenInContext(context.Background(), sa)
			require.NoError(t, err)
			id := uuid.NewV4()
			// when
			_, err = s.Application.ClusterService().Load(ctx, id)
			// then
			require.Error(t, err)
			testsupport.AssertError(t, err, errors.NotFoundError{}, errors.NewNotFoundError("cluster", id.String()).Error())
		})
	})
}

func (s *ClusterServiceTestSuite) TestLoadForAuth() {

	s.T().Run("ok", func(t *testing.T) {
		for _, saName := range []string{"fabric8-auth"} {
			t.Run(saName, func(t *testing.T) {
				// given
				c := newTestCluster()
				err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
				require.NoError(t, err)
				sa := &authtestsupport.Identity{
					Username: saName,
					ID:       uuid.NewV4(),
				}
				ctx, err := authtestsupport.EmbedServiceAccountTokenInContext(context.Background(), sa)
				require.NoError(t, err)
				// when
				result, err := s.Application.ClusterService().LoadForAuth(ctx, c.ClusterID)
				// then
				require.NoError(t, err)
				require.NotNil(t, result)
				test.AssertEqualCluster(t, *c, *result, true)
			})
		}
	})

	s.T().Run("failures", func(t *testing.T) {

		t.Run("unauthorized", func(t *testing.T) {
			for _, saName := range []string{"fabric8-oso-proxy", "fabric8-tenant", "fabric8-jenkins-idler", "fabric8-jenkins-proxy", "other"} {
				t.Run(saName, func(t *testing.T) {
					// given
					c := newTestCluster()
					err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
					require.NoError(t, err)
					sa := &authtestsupport.Identity{
						Username: "foo",
						ID:       uuid.NewV4(),
					}
					ctx, err := authtestsupport.EmbedServiceAccountTokenInContext(context.Background(), sa)
					require.NoError(t, err)
					// when
					_, err = s.Application.ClusterService().Load(ctx, c.ClusterID)
					// then
					testsupport.AssertError(t, err, errors.UnauthorizedError{}, "unauthorized access to cluster info")
				})
			}
		})

		t.Run("not found", func(t *testing.T) {
			// given
			c := newTestCluster()
			err := s.Application.ClusterService().CreateOrSaveCluster(context.Background(), c)
			require.NoError(t, err)
			sa := &authtestsupport.Identity{
				Username: "fabric8-auth", // use a SA that is not unauthorized
				ID:       uuid.NewV4(),
			}
			ctx, err := authtestsupport.EmbedServiceAccountTokenInContext(context.Background(), sa)
			require.NoError(t, err)
			id := uuid.NewV4()
			// when
			_, err = s.Application.ClusterService().Load(ctx, id)
			// then
			require.Error(t, err)
			testsupport.AssertError(t, err, errors.NotFoundError{}, errors.NewNotFoundError("cluster", id.String()).Error())
		})
	})
}

func (s *ClusterServiceTestSuite) TestList() {
	// given
	test.CreateCluster(s.T(), s.DB) // add extra cluster
	err := s.Application.ClusterService().CreateOrSaveClusterFromConfig(context.Background())
	require.NoError(s.T(), err)

	s.T().Run("ok", func(t *testing.T) {
		for _, saName := range []string{"fabric8-oso-proxy", "fabric8-tenant", "fabric8-jenkins-idler", "fabric8-jenkins-proxy", "fabric8-auth"} {
			t.Run(saName, func(t *testing.T) {
				// given
				sa := &authtestsupport.Identity{
					Username: saName,
					ID:       uuid.NewV4(),
				}
				ctx, err := authtestsupport.EmbedServiceAccountTokenInContext(context.Background(), sa)
				require.NoError(t, err)
				// when
				result, err := s.Application.ClusterService().List(ctx)
				// then
				require.NoError(t, err)
				expected, err := repository.NewClusterRepository(s.DB).List(context.Background())
				require.NoError(t, err)
				test.AssertEqualClusters(t, expected, result, false)
			})
		}
	})

	s.T().Run("failures", func(t *testing.T) {
		t.Run("unauthorized", func(t *testing.T) {
			// given
			err := s.Application.ClusterService().CreateOrSaveClusterFromConfig(context.Background())
			require.NoError(t, err)
			sa := &authtestsupport.Identity{
				Username: "foo",
				ID:       uuid.NewV4(),
			}
			ctx, err := authtestsupport.EmbedServiceAccountTokenInContext(context.Background(), sa)
			require.NoError(t, err)
			// when
			_, err = s.Application.ClusterService().List(ctx)
			// then
			require.Error(t, err)
			testsupport.AssertError(t, err, errors.UnauthorizedError{}, "unauthorized access to clusters info")
		})
	})
}
func (s *ClusterServiceTestSuite) TestListForAuth() {
	// given
	test.CreateCluster(s.T(), s.DB) // add extra cluster
	err := s.Application.ClusterService().CreateOrSaveClusterFromConfig(context.Background())
	require.NoError(s.T(), err)

	s.T().Run("ok", func(t *testing.T) {
		for _, saName := range []string{"fabric8-auth"} {
			t.Run(saName, func(t *testing.T) {
				// given
				sa := &authtestsupport.Identity{
					Username: saName,
					ID:       uuid.NewV4(),
				}
				ctx, err := authtestsupport.EmbedServiceAccountTokenInContext(context.Background(), sa)
				require.NoError(t, err)
				// when
				result, err := s.Application.ClusterService().ListForAuth(ctx)
				// then
				require.NoError(t, err)
				expected, err := repository.NewClusterRepository(s.DB).List(context.Background())
				require.NoError(t, err)
				test.AssertEqualClusters(t, expected, result, true)
			})
		}
	})

	s.T().Run("failures", func(t *testing.T) {

		t.Run("unauthorized", func(t *testing.T) {
			for _, saName := range []string{"fabric8-oso-proxy", "fabric8-tenant", "fabric8-jenkins-idler", "fabric8-jenkins-proxy", "other"} {
				t.Run(saName, func(t *testing.T) {
					// given
					err := s.Application.ClusterService().CreateOrSaveClusterFromConfig(context.Background())
					require.NoError(t, err)
					sa := &authtestsupport.Identity{
						Username: saName,
						ID:       uuid.NewV4(),
					}
					ctx, err := authtestsupport.EmbedServiceAccountTokenInContext(context.Background(), sa)
					require.NoError(t, err)
					// when
					_, err = s.Application.ClusterService().ListForAuth(ctx)
					// then
					require.Error(t, err)
					testsupport.AssertError(t, err, errors.UnauthorizedError{}, "unauthorized access to clusters info")
				})
			}
		})
	})
}

func (s *ClusterServiceTestSuite) TestDelete() {

	s.T().Run("ok", func(t *testing.T) {
		// given
		c := test.CreateCluster(t, s.DB)
		idCuster1 := test.CreateIdentityCluster(t, s.DB, test.WithCluster(c))
		idCuster2 := test.CreateIdentityCluster(t, s.DB, test.WithCluster(c))
		// noise
		noiseIdCuster1 := test.CreateIdentityCluster(t, s.DB)
		noiseIdCuster2 := test.CreateIdentityCluster(t, s.DB)
		// auth
		sa := &authtestsupport.Identity{
			Username: auth.ToolChainOperator,
			ID:       uuid.NewV4(),
		}
		ctx, err := authtestsupport.EmbedServiceAccountTokenInContext(context.Background(), sa)
		require.NoError(t, err)
		// when
		err = s.Application.ClusterService().Delete(ctx, c.ClusterID)
		// then
		require.NoError(t, err)
		// check that cluster and links to identities were removed
		r, err := repository.NewClusterRepository(s.DB).Load(ctx, c.ClusterID)
		testsupport.AssertError(t, err, errors.NotFoundError{}, errors.NewNotFoundError("cluster", c.ClusterID.String()).Error())
		require.Nil(t, r)
		_, err = repository.NewIdentityClusterRepository(s.DB).Load(ctx, idCuster1.IdentityID, idCuster1.ClusterID)
		testsupport.AssertError(t, err, errors.NotFoundError{}, fmt.Sprintf("identity_cluster with identity ID %s and cluster ID %s not found", idCuster1.IdentityID, idCuster1.ClusterID.String()))
		_, err = repository.NewIdentityClusterRepository(s.DB).Load(ctx, idCuster2.IdentityID, idCuster2.ClusterID)
		testsupport.AssertError(t, err, errors.NotFoundError{}, fmt.Sprintf("identity_cluster with identity ID %s and cluster ID %s not found", idCuster2.IdentityID, idCuster2.ClusterID.String()))
		// also check that other cluster/identities (noise) still exist
		_, err = repository.NewIdentityClusterRepository(s.DB).Load(ctx, noiseIdCuster1.IdentityID, noiseIdCuster1.ClusterID)
		require.NoError(t, err)
		_, err = repository.NewIdentityClusterRepository(s.DB).Load(ctx, noiseIdCuster2.IdentityID, noiseIdCuster2.ClusterID)
		require.NoError(t, err)
	})

	s.T().Run("failures", func(t *testing.T) {

		t.Run("unauthorized", func(t *testing.T) {
			// given
			c := test.CreateCluster(t, s.DB)

			for _, saName := range []string{"fabric8-auth", "fabric8-oso-proxy", "fabric8-tenant", "fabric8-jenkins-idler", "fabric8-jenkins-proxy", "other"} {
				t.Run(saName, func(t *testing.T) {
					// given
					sa := &authtestsupport.Identity{
						Username: saName,
						ID:       uuid.NewV4(),
					}
					ctx, err := authtestsupport.EmbedServiceAccountTokenInContext(context.Background(), sa)
					require.NoError(t, err)
					// when
					err = s.Application.ClusterService().Delete(ctx, c.ClusterID)
					// then
					testsupport.AssertError(t, err, errors.UnauthorizedError{}, "unauthorized access to delete a cluster configuration")
				})
			}
		})

		t.Run("not found", func(t *testing.T) {
			// given
			sa := &authtestsupport.Identity{
				Username: auth.ToolChainOperator,
				ID:       uuid.NewV4(),
			}
			ctx, err := authtestsupport.EmbedServiceAccountTokenInContext(context.Background(), sa)
			require.NoError(t, err)
			// when
			clusterID := uuid.NewV4()
			err = s.Application.ClusterService().Delete(ctx, clusterID)
			// then
			testsupport.AssertError(t, err, errors.NotFoundError{}, errors.NewNotFoundError("cluster", clusterID.String()).Error())
		})

	})

}

func newTestCluster() *repository.Cluster {
	name := uuid.NewV4().String()
	return &repository.Cluster{
		Name:              name,
		Type:              cluster.OCP,
		URL:               fmt.Sprintf("https://api.cluster.%s", name),
		AppDNS:            fmt.Sprintf("https://cluster.%s", name),
		ConsoleURL:        fmt.Sprintf("https://console.cluster.%s", name),
		LoggingURL:        fmt.Sprintf("https://logging.cluster.%s", name),
		MetricsURL:        fmt.Sprintf("https://metrics.cluster.%s", name),
		CapacityExhausted: true,
		SAToken:           "ServiceAccountToken",
		SAUsername:        "ServiceAccountUsername",
		SATokenEncrypted:  true,
		TokenProviderID:   "TokenProviderID",
		AuthClientID:      "AuthClientID",
		AuthClientSecret:  "AuthClientSecret",
		AuthDefaultScope:  "AuthClientDefaultScope",
	}
}

func (s *ClusterServiceTestSuite) TestClusterConfigurationWatcher() {
	t := s.T()
	// Create a temp file with content from ./conf-files/oso-clusters-custom.conf
	tmpFileName := createTempClusterConfigFile(t)
	defer os.Remove(tmpFileName)

	// Load configuration from the temp file
	config, err := configuration.NewConfigurationData("", tmpFileName)
	require.NoError(t, err)
	c := config.GetClusterByURL("https://api.starter-us-east-2a.openshift.com")
	require.NotNil(t, c)

	original := c.CapacityExhausted

	// initialize application with new config
	application := gormapplication.NewGormDB(s.DB, config)
	// Start watching
	haltWatcher, err := application.ClusterService().InitializeClusterWatcher()
	require.NoError(t, err)
	defer func() {
		haltWatcher()
	}()

	// Update the config file
	updateClusterConfigFile(t, tmpFileName, "./configuration/conf-files/tests/oso-clusters-capacity-updated.conf")
	// Check if it has been updated
	waitForConfigUpdate(t, config, !original)

	// Update the config file back to the original
	updateClusterConfigFile(t, tmpFileName, "./configuration/conf-files/oso-clusters.conf")
	// Check if it has been updated
	waitForConfigUpdate(t, config, original)

	// Update the config file to some invalid data
	updateClusterConfigFile(t, tmpFileName, "./configuration/conf-files/tests/oso-clusters-invalid.conf")
	// The configuration should not change
	waitForConfigUpdate(t, config, original)
	updateClusterConfigFile(t, tmpFileName, "./configuration/conf-files/tests/oso-clusters-empty.conf")
	// The configuration should not change
	waitForConfigUpdate(t, config, original)

	// Fix the file
	updateClusterConfigFile(t, tmpFileName, "./configuration/conf-files/tests/oso-clusters-capacity-updated.conf")
	// Now configuration should be updated
	waitForConfigUpdate(t, config, !original)
}

func (s *ClusterServiceTestSuite) TestClusterConfigurationWatcherNoErrorForDefaultConfig() {
	s.Application = gormapplication.NewGormDB(s.DB, s.Configuration)
	haltWatcher, err := s.Application.ClusterService().InitializeClusterWatcher()
	require.NoError(s.T(), err)
	defer func() {
		haltWatcher()
	}()
}

func (s *ClusterServiceTestSuite) TestLinkIdentityToCluster() {

	s.T().Run("ok", func(t *testing.T) {

		t.Run("ignore if exists", func(t *testing.T) {
			// given
			identityCluster := test.CreateIdentityCluster(t, s.DB)

			// when
			err := s.Application.ClusterService().LinkIdentityToCluster(s.Ctx, identityCluster.IdentityID, identityCluster.Cluster.URL, true)
			require.NoError(t, err)

			// then
			loaded1, err := s.Application.IdentityClusters().Load(s.Ctx, identityCluster.IdentityID, identityCluster.Cluster.ClusterID)
			require.NoError(t, err)
			test.AssertEqualCluster(t, identityCluster.Cluster, loaded1.Cluster, true)
			test.AssertEqualIdentityClusters(t, identityCluster, *loaded1)

			clusters, err := s.Application.IdentityClusters().ListClustersForIdentity(s.Ctx, identityCluster.IdentityID)
			require.NoError(t, err)
			assert.Len(t, clusters, 1)
			test.AssertEqualCluster(t, identityCluster.Cluster, clusters[0], true)
		})

		t.Run("do not ignore if exists", func(t *testing.T) {
			// given
			identityCluster := test.CreateIdentityCluster(t, s.DB)

			// when
			err := s.Application.ClusterService().LinkIdentityToCluster(s.Ctx, identityCluster.IdentityID, identityCluster.Cluster.URL, false)
			testsupport.AssertError(t, err, errors.InternalError{}, "failed to link identity '%s' with cluster '%s': pq: duplicate key value violates unique constraint \"identity_cluster_pkey\"", identityCluster.IdentityID, identityCluster.Cluster.ClusterID)

			// then
			loaded1, err := s.Application.IdentityClusters().Load(s.Ctx, identityCluster.IdentityID, identityCluster.Cluster.ClusterID)
			require.NoError(t, err)
			test.AssertEqualCluster(t, identityCluster.Cluster, loaded1.Cluster, true)
			test.AssertEqualIdentityClusters(t, identityCluster, *loaded1)

			clusters, err := s.Application.IdentityClusters().ListClustersForIdentity(s.Ctx, identityCluster.IdentityID)
			require.NoError(t, err)

			assert.Len(t, clusters, 1)
			test.AssertEqualCluster(t, identityCluster.Cluster, clusters[0], true)
		})

		t.Run("link multiple clusters to single identity", func(t *testing.T) {
			// given
			identityID := uuid.NewV4()
			c1 := test.CreateCluster(t, s.DB)
			identityCluster1 := test.CreateIdentityCluster(t, s.DB, test.WithCluster(c1), test.WithIdentityID(identityID))
			c2 := test.CreateCluster(t, s.DB)
			identityCluster2 := repository.IdentityCluster{
				IdentityID: identityID,
				ClusterID:  c2.ClusterID,
			}
			// when
			err := s.Application.ClusterService().LinkIdentityToCluster(s.Ctx, identityID, c2.URL, true)
			require.NoError(t, err)
			// then
			loaded1, err := s.Application.IdentityClusters().Load(s.Ctx, identityID, c1.ClusterID)
			require.NoError(t, err)
			test.AssertEqualCluster(t, c1, loaded1.Cluster, true)
			test.AssertEqualIdentityClusters(t, identityCluster1, *loaded1)
			loaded2, err := s.Application.IdentityClusters().Load(s.Ctx, identityID, c2.ClusterID)
			require.NoError(t, err)
			test.AssertEqualCluster(t, c2, loaded2.Cluster, true)
			test.AssertEqualIdentityClusters(t, identityCluster2, *loaded2)
		})
	})

	s.T().Run("failures", func(t *testing.T) {

		t.Run("random cluster url", func(t *testing.T) {
			// given
			clusterURL := "http://random.url"
			// when
			err := s.Application.ClusterService().LinkIdentityToCluster(s.Ctx, uuid.NewV4(), clusterURL, true)
			// then
			test.AssertError(t, err, errors.NotFoundError{}, "cluster with url '%s' not found", clusterURL)
		})
	})
}

func (s *ClusterServiceTestSuite) TestRemoveIdentityToClusterLink() {

	s.T().Run("ok", func(t *testing.T) {

		t.Run("unlink completely", func(t *testing.T) {
			// given
			identityCluster := test.CreateIdentityCluster(t, s.DB)
			// when
			err := s.Application.ClusterService().RemoveIdentityToClusterLink(s.Ctx, identityCluster.IdentityID, identityCluster.Cluster.URL)
			// then
			require.NoError(t, err)
			_, err = s.Application.IdentityClusters().Load(s.Ctx, identityCluster.IdentityID, identityCluster.Cluster.ClusterID)
			test.AssertError(t, err, errors.NotFoundError{}, fmt.Sprintf("identity_cluster with identity ID %s and cluster ID %s not found", identityCluster.IdentityID, identityCluster.Cluster.ClusterID))
			clusters, err := s.Application.IdentityClusters().ListClustersForIdentity(s.Ctx, identityCluster.IdentityID)
			require.NoError(t, err)
			assert.Empty(t, clusters)
		})

		t.Run("unlink single cluster", func(t *testing.T) {
			// given
			identityID := uuid.NewV4()
			identityCluster1 := test.CreateIdentityCluster(t, s.DB, test.WithIdentityID(identityID))
			identityCluster2 := test.CreateIdentityCluster(t, s.DB, test.WithIdentityID(identityID))

			// when
			err := s.Application.ClusterService().RemoveIdentityToClusterLink(s.Ctx, identityID, identityCluster2.Cluster.URL)
			require.NoError(t, err)

			// then
			clusters, err := s.Application.IdentityClusters().ListClustersForIdentity(s.Ctx, identityID)
			require.NoError(t, err)
			require.Len(t, clusters, 1)
			test.AssertEqualCluster(t, identityCluster1.Cluster, clusters[0], true)

			loaded1, err := s.Application.IdentityClusters().Load(s.Ctx, identityID, identityCluster1.Cluster.ClusterID)
			require.NoError(t, err)
			test.AssertEqualCluster(t, identityCluster1.Cluster, loaded1.Cluster, true)
			test.AssertEqualIdentityClusters(t, identityCluster1, *loaded1)

			_, err = s.Application.IdentityClusters().Load(s.Ctx, identityID, identityCluster2.Cluster.ClusterID)
			test.AssertError(t, err, errors.NotFoundError{}, fmt.Sprintf("identity_cluster with identity ID %s and cluster ID %s not found", identityID, identityCluster2.Cluster.ClusterID))
		})
	})

	s.T().Run("failures", func(t *testing.T) {

		// given
		identityCluster := test.CreateIdentityCluster(t, s.DB)

		t.Run("random cluster url", func(t *testing.T) {
			// given
			clusterURL := "http://random.url"
			// when
			err := s.Application.ClusterService().RemoveIdentityToClusterLink(s.Ctx, identityCluster.IdentityID, clusterURL)
			// then
			test.AssertError(t, err, errors.NotFoundError{}, fmt.Sprintf(`nothing to delete: identity cluster not found (cluster with URL '%s' not found)`, clusterURL))
		})

		t.Run("random identity id", func(t *testing.T) {
			// given
			identityID := uuid.NewV4()
			// when
			err := s.Application.ClusterService().RemoveIdentityToClusterLink(s.Ctx, identityID, identityCluster.Cluster.URL)
			// then
			test.AssertError(t, err, errors.NotFoundError{}, fmt.Sprintf(`nothing to delete: identity cluster not found (identity-id:'%s', cluster-url:'%s')`, identityID.String(), identityCluster.Cluster.URL))
		})
	})
}

func createTempClusterConfigFile(t *testing.T) string {
	to, err := ioutil.TempFile("", "oso-clusters.conf")
	require.NoError(t, err)
	defer to.Close()

	from, err := os.Open("./../../configuration/conf-files/oso-clusters.conf")
	require.NoError(t, err)
	defer from.Close()

	_, err = io.Copy(to, from)
	require.NoError(t, err)
	return to.Name()
}

func updateClusterConfigFile(t *testing.T, to, from string) {
	fromFile, err := os.Open("./../../" + from)
	require.NoError(t, err)
	defer fromFile.Close()

	toFile, err := os.OpenFile(to, os.O_RDWR|os.O_CREATE, 0666)
	require.NoError(t, err)
	defer toFile.Close()

	_, err = io.Copy(toFile, fromFile)
	require.NoError(t, err)
}

func waitForConfigUpdate(t *testing.T, config *configuration.ConfigurationData, expected bool) {
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		c := config.GetClusterByURL("https://api.starter-us-east-2a.openshift.com")
		require.NotNil(t, c)

		// verify that cluster type set to OSO in case of not present in config
		require.Equal(t, cluster.OSO, c.Type)
		if expected == c.CapacityExhausted {
			return
		}
	}
	require.Fail(t, "cluster config has not been reloaded within 3s")
}

func verifyClusters(t *testing.T, expectedClusters map[string]repository.Cluster, actualClusters []repository.Cluster, compareSensitiveInfo bool) {
	for _, expectedCluster := range expectedClusters {
		err := expectedCluster.Normalize()
		require.NoError(t, err)
		verifyCluster(t, expectedCluster, actualClusters, compareSensitiveInfo)
	}
}

func verifyCluster(t *testing.T, expectedCluster repository.Cluster, actualClusters []repository.Cluster, compareSensitiveInfo bool) {
	actualCluster, err := test.FilterClusterByURL(expectedCluster.URL, actualClusters)
	require.NoError(t, err)
	test.AssertEqualClusterDetails(t, expectedCluster, actualCluster, compareSensitiveInfo)
}
