package configuration_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/fabric8-services/fabric8-cluster/cluster"
	"github.com/fabric8-services/fabric8-cluster/cluster/repository"
	"github.com/fabric8-services/fabric8-cluster/configuration"
	"github.com/fabric8-services/fabric8-common/resource"
	testsuite "github.com/fabric8-services/fabric8-common/test/suite"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var reqLong *goa.RequestData
var reqShort *goa.RequestData

func TestConfigurationBlackbox(t *testing.T) {
	suite.Run(t, &ConfigurationBlackboxTestSuite{})
}

type ConfigurationBlackboxTestSuite struct {
	testsuite.UnitTestSuite
	config *configuration.ConfigurationData
}

func (s *ConfigurationBlackboxTestSuite) SetupTest() {
	resource.Require(s.T(), resource.UnitTest)
	config, err := configuration.GetConfigurationData()
	require.NoError(s.T(), err)
	s.config = config
}

const (
	constAuthEnvironment string = "F8_ENVIRONMENT"
	constAuthSentryDSN   string = "F8_SENTRY_DSN"
	constLocalEnv        string = "local"
)

func (s *ConfigurationBlackboxTestSuite) TestGetEnvironmentOK() {

	existingEnvironmentName := os.Getenv(constAuthEnvironment)
	existingSentryDSN := os.Getenv(constAuthSentryDSN)
	defer func() {
		os.Setenv(constAuthEnvironment, existingEnvironmentName)
		os.Setenv(constAuthSentryDSN, existingSentryDSN)
	}()

	os.Unsetenv(constAuthEnvironment)
	assert.Equal(s.T(), constLocalEnv, s.config.GetEnvironment())

	// Test cluster service URL

	// Environment not set
	saConfig, err := configuration.GetConfigurationData()
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "http://localhost", saConfig.GetClusterServiceURL())
	assert.Contains(s.T(), saConfig.DefaultConfigurationError().Error(), "environment is expected to be set to 'production' or 'prod-preview'")

	// Environment set to some unknown value
	os.Setenv(constAuthEnvironment, "somethingelse")
	saConfig, err = configuration.GetConfigurationData()
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "http://localhost", saConfig.GetClusterServiceURL())
	assert.Contains(s.T(), saConfig.DefaultConfigurationError().Error(), "environment is expected to be set to 'production' or 'prod-preview'")

	// Environment set to prod-preview
	os.Setenv(constAuthEnvironment, "prod-preview")
	saConfig, err = configuration.GetConfigurationData()
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "prod-preview", saConfig.GetEnvironment())
	assert.Equal(s.T(), "https://cluster.prod-preview.openshift.io", saConfig.GetClusterServiceURL())
	assert.NotContains(s.T(), saConfig.DefaultConfigurationError().Error(), "environment is expected to be set to 'production' or 'prod-preview'")

	// Environment set to production
	os.Setenv(constAuthEnvironment, "production")
	saConfig, err = configuration.GetConfigurationData()
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "production", saConfig.GetEnvironment())
	assert.Equal(s.T(), "https://cluster.openshift.io", saConfig.GetClusterServiceURL())
	assert.NotContains(s.T(), saConfig.DefaultConfigurationError().Error(), "environment is expected to be set to 'production' or 'prod-preview'")
}

func (s *ConfigurationBlackboxTestSuite) TestAuthServiceURL() {
	existingEnvironment := os.Getenv("F8_DEVELOPER_MODE_ENABLED")
	defer func() {
		os.Setenv("F8_DEVELOPER_MODE_ENABLED", existingEnvironment)
	}()
	os.Unsetenv("F8_DEVELOPER_MODE_ENABLED")

	checkURLValidation(s.T(), "F8_AUTH_URL", "Auth service")
}

func checkURLValidation(t *testing.T, envName, serviceName string) {
	existingEnvironment := os.Getenv(envName)
	defer func() {
		os.Setenv(envName, existingEnvironment)
	}()

	// URL not set
	os.Unsetenv(envName)
	saConfig, err := configuration.GetConfigurationData()
	require.NoError(t, err)
	assert.Contains(t, saConfig.DefaultConfigurationError().Error(), fmt.Sprintf("%s url is empty", serviceName))

	// URL is invalid
	os.Setenv(envName, "%")
	saConfig, err = configuration.GetConfigurationData()
	require.NoError(t, err)
	assert.Contains(t, saConfig.DefaultConfigurationError().Error(), fmt.Sprintf("invalid %s url: %s", serviceName, "parse %: invalid URL escape \"%\""))

	// URL is valid
	os.Setenv(envName, "https://openshift.io")
	saConfig, err = configuration.GetConfigurationData()
	require.NoError(t, err)
	assert.NotContains(t, saConfig.DefaultConfigurationError().Error(), "serviceName")
}

func (s *ConfigurationBlackboxTestSuite) TestGetSentryDSNOK() {
	constSentryDSN := "F8_SENTRY_DSN"
	existingDSN := os.Getenv(constSentryDSN)
	defer func() {
		os.Setenv(constSentryDSN, existingDSN)
	}()

	os.Unsetenv(constSentryDSN)
	assert.Equal(s.T(), "", s.config.GetSentryDSN())

	os.Setenv(constSentryDSN, "something")
	assert.Equal(s.T(), "something", s.config.GetSentryDSN())
}

func (s *ConfigurationBlackboxTestSuite) TestLoadDefaultClusterConfiguration() {
	// when
	clusters := s.config.GetClusters()
	// then
	checkClusterConfiguration(s.T(), clusters)
	cluster := s.config.GetClusterByURL("https://api.starter-us-east-2.openshift.com")
	assert.NotNil(s.T(), cluster)
	cluster = s.config.GetClusterByURL("https://api.starter-us-east-2.openshift.com")
	assert.NotNil(s.T(), cluster)
	cluster = s.config.GetClusterByURL("https://api.starter-us-east-2.openshift.com/path")
	assert.NotNil(s.T(), cluster)
	cluster = s.config.GetClusterByURL("https://api.starter-us-east-2.openshift.unknown")
	assert.Nil(s.T(), cluster)
}

func (s *ConfigurationBlackboxTestSuite) TestLoadClusterConfigurationFromFile() {
	clusterConfig, err := configuration.NewConfigurationData("", "./conf-files/oso-clusters.conf")
	require.Nil(s.T(), err)
	clusters := clusterConfig.GetClusters()
	checkClusterConfiguration(s.T(), clusters)
}

func (s *ConfigurationBlackboxTestSuite) TestClusterConfigurationWithMissingKeys() {
	_, err := configuration.NewConfigurationData("", "./conf-files/tests/oso-clusters-missing-keys.conf")
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "key name is missing")
	assert.Contains(s.T(), err.Error(), "key app-dns is missing")
	assert.Contains(s.T(), err.Error(), "key service-account-token is missing")
	assert.Contains(s.T(), err.Error(), "key service-account-username is missing")
	assert.Contains(s.T(), err.Error(), "key token-provider-id is missing")
	assert.Contains(s.T(), err.Error(), "key auth-client-id is missing")
	assert.Contains(s.T(), err.Error(), "key auth-client-secret is missing")
	assert.Contains(s.T(), err.Error(), "key auth-client-default-scope is missing")
}

func (s *ConfigurationBlackboxTestSuite) TestClusterConfigurationWithGeneratedURLs() {
	clusterConfig, err := configuration.NewConfigurationData("", "./conf-files/tests/oso-clusters-custom-urls.conf")
	require.Nil(s.T(), err)
	checkCluster(s.T(), clusterConfig.GetClusters(), repository.Cluster{
		Name:             "us-east-2",
		URL:              "https://api.starter-us-east-2.openshift.com",
		ConsoleURL:       "custom.console.url",
		MetricsURL:       "custom.metrics.url",
		LoggingURL:       "custom.logging.url",
		AppDNS:           "8a09.starter-us-east-2.openshiftapps.com",
		SAToken:          "fX0nH3d68LQ6SK5wBE6QeKJ6X8AZGVQO3dGQZZETakhmgmWAqr2KDFXE65KUwBO69aWoq",
		SAUsername:       "dsaas",
		SATokenEncrypted: false,
		TokenProviderID:  "f867ac10-5e05-4359-a0c6-b855ece59090",
		AuthClientID:     "autheast2",
		AuthClientSecret: "autheast2secret",
		AuthDefaultScope: "user:full",
		Type:             cluster.OSO,
	})
}

func (s *ConfigurationBlackboxTestSuite) TestClusterConfigurationWithEmptyArray() {
	_, err := configuration.NewConfigurationData("", "./conf-files/tests/oso-clusters-empty.conf")
	require.Error(s.T(), err)
	assert.Equal(s.T(), err.Error(), "empty cluster config file")
}

func (s *ConfigurationBlackboxTestSuite) TestClusterConfigurationFromInvalidFile() {
	_, err := configuration.NewConfigurationData("", "./conf-files/tests/oso-clusters-invalid.conf")
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "failed to load the JSON config file")

	_, err = configuration.NewConfigurationData("", "./conf-files/tests/oso-clusters-wrong-json.conf")
	require.Error(s.T(), err)
	assert.Equal(s.T(), err.Error(), "empty cluster config file")
}

func checkClusterConfiguration(t *testing.T, clusters map[string]repository.Cluster) {
	checkCluster(t, clusters, repository.Cluster{
		Name: "us-east-2",
		URL:  "https://api.starter-us-east-2.openshift.com",
		// ConsoleURL:        "https://console.starter-us-east-2.openshift.com/console/",
		// MetricsURL:        "https://metrics.starter-us-east-2.openshift.com/",
		// LoggingURL:        "https://console.starter-us-east-2.openshift.com/console/",
		AppDNS:            "8a09.starter-us-east-2.openshiftapps.com",
		SAToken:           "fX0nH3d68LQ6SK5wBE6QeKJ6X8AZGVQO3dGQZZETakhmgmWAqr2KDFXE65KUwBO69aWoq",
		SAUsername:        "dsaas",
		SATokenEncrypted:  true,
		TokenProviderID:   "f867ac10-5e05-4359-a0c6-b855ece59090",
		AuthClientID:      "autheast2",
		AuthClientSecret:  "autheast2secret",
		AuthDefaultScope:  "user:full",
		Type:              "OSO", // assigned by default
		CapacityExhausted: false,
	})
	checkCluster(t, clusters, repository.Cluster{
		Name: "us-east-2a",
		URL:  "https://api.starter-us-east-2a.openshift.com",
		// ConsoleURL:        "https://console.starter-us-east-2a.openshift.com/console/",
		// MetricsURL:        "https://metrics.starter-us-east-2a.openshift.com/",
		// LoggingURL:        "https://console.starter-us-east-2a.openshift.com/console/",
		AppDNS:            "b542.starter-us-east-2a.openshiftapps.com",
		SAToken:           "ak61T6RSAacWFruh1vZP8cyUOBtQ3Chv1rdOBddSuc9nZ2wEcs81DHXRO55NpIpVQ8uiH",
		SAUsername:        "dsaas",
		SATokenEncrypted:  true,
		TokenProviderID:   "886c7ea3-ef97-443d-b345-de94b94bb65d",
		AuthClientID:      "autheast2a",
		AuthClientSecret:  "autheast2asecret",
		AuthDefaultScope:  "user:full",
		Type:              "OSO", // assigned by default
		CapacityExhausted: false,
	})
	checkCluster(t, clusters, repository.Cluster{
		Name: "us-east-1a",
		URL:  "https://api.starter-us-east-1a.openshift.com",
		// ConsoleURL:        "https://console.starter-us-east-1a.openshift.com/console/",
		// MetricsURL:        "https://metrics.starter-us-east-1a.openshift.com/",
		// LoggingURL:        "https://console.starter-us-east-1a.openshift.com/console/",
		AppDNS:            "b542.starter-us-east-1a.openshiftapps.com",
		SAToken:           "sdfjdlfjdfkjdlfjd12324434543085djdfjd084508gfdkjdofkjg43854085dlkjdlk",
		SAUsername:        "dsaas",
		SATokenEncrypted:  false,
		TokenProviderID:   "886c7ea3-ef97-443d-b345-de94b94bb65d",
		AuthClientID:      "autheast1a",
		AuthClientSecret:  "autheast1asecret",
		AuthDefaultScope:  "user:full",
		Type:              "OSO", // assigned by default
		CapacityExhausted: true,
	})
	checkCluster(t, clusters, repository.Cluster{
		Name: "us-east-3a",
		URL:  "https://api.starter-us-east-3a.openshift.com",
		// ConsoleURL:        "https://console.starter-us-east-3a.openshift.com/console/",
		// MetricsURL:        "https://metrics.starter-us-east-3a.openshift.com/",
		// LoggingURL:        "https://console.starter-us-east-3a.openshift.com/console/",
		AppDNS:            "b542.starter-us-east-3a.openshiftapps.com",
		SAToken:           "fkdjhfdsjfgfdjlsflhjgsafgskfdsagrwgwerwshbdjasbdjbsahdbsagbdyhsbdesbh",
		SAUsername:        "dsaas",
		SATokenEncrypted:  true,
		TokenProviderID:   "1c09073a-13ad-4add-b0ff-197eaf18fc37",
		AuthClientID:      "autheast3a",
		AuthClientSecret:  "autheast3asecret",
		AuthDefaultScope:  "user:full",
		Type:              "OSD",
		CapacityExhausted: false,
	})
}

func checkCluster(t *testing.T, clusters map[string]repository.Cluster, expected repository.Cluster) {
	require.Contains(t, clusters, expected.URL)
	require.Equal(t, expected, clusters[expected.URL])
	_, err := uuid.FromString(clusters[expected.URL].TokenProviderID)
	require.Nil(t, err)
}
