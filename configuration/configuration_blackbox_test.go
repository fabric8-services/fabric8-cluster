package configuration_test

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/fabric8-services/fabric8-cluster/configuration"
	"github.com/fabric8-services/fabric8-common/resource"

	"github.com/fabric8-services/fabric8-cluster/cluster"
	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var reqLong *goa.RequestData
var reqShort *goa.RequestData
var config *configuration.ConfigurationData

func TestMain(m *testing.M) {
	resetConfiguration()

	reqLong = &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}
	reqShort = &goa.RequestData{
		Request: &http.Request{Host: "api.domain.org"},
	}
	os.Exit(m.Run())
}

func resetConfiguration() {
	var err error

	// calling NewConfigurationData("") is same as GetConfigurationData()
	config, err = configuration.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("failed to setup the configuration: %s", err.Error()))
	}
}

func TestGetEnvironmentOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	constAuthEnvironment := "F8_ENVIRONMENT"
	constAuthSentryDSN := "F8_SENTRY_DSN"
	constLocalEnv := "local"

	existingEnvironmentName := os.Getenv(constAuthEnvironment)
	existingSentryDSN := os.Getenv(constAuthSentryDSN)
	defer func() {
		os.Setenv(constAuthEnvironment, existingEnvironmentName)
		os.Setenv(constAuthSentryDSN, existingSentryDSN)
		resetConfiguration()
	}()

	os.Unsetenv(constAuthEnvironment)
	assert.Equal(t, constLocalEnv, config.GetEnvironment())

	// Test cluster service URL

	// Environment not set
	saConfig, err := configuration.GetConfigurationData()
	require.NoError(t, err)
	assert.Equal(t, "http://localhost", saConfig.GetClusterServiceURL())
	assert.Contains(t, saConfig.DefaultConfigurationError().Error(), "environment is expected to be set to 'production' or 'prod-preview'")

	// Environment set to some unknown value
	os.Setenv(constAuthEnvironment, "somethingelse")
	saConfig, err = configuration.GetConfigurationData()
	require.NoError(t, err)
	assert.Equal(t, "http://localhost", saConfig.GetClusterServiceURL())
	assert.Contains(t, saConfig.DefaultConfigurationError().Error(), "environment is expected to be set to 'production' or 'prod-preview'")

	// Environment set to prod-preview
	os.Setenv(constAuthEnvironment, "prod-preview")
	saConfig, err = configuration.GetConfigurationData()
	require.NoError(t, err)
	assert.Equal(t, "prod-preview", saConfig.GetEnvironment())
	assert.Equal(t, "https://cluster.prod-preview.openshift.io", saConfig.GetClusterServiceURL())
	assert.NotContains(t, saConfig.DefaultConfigurationError().Error(), "environment is expected to be set to 'production' or 'prod-preview'")

	// Environment set to production
	os.Setenv(constAuthEnvironment, "production")
	saConfig, err = configuration.GetConfigurationData()
	require.NoError(t, err)
	assert.Equal(t, "production", saConfig.GetEnvironment())
	assert.Equal(t, "https://cluster.openshift.io", saConfig.GetClusterServiceURL())
	assert.NotContains(t, saConfig.DefaultConfigurationError().Error(), "environment is expected to be set to 'production' or 'prod-preview'")
}

func TestAuthServiceURL(t *testing.T) {
	existingEnvironment := os.Getenv("F8_DEVELOPER_MODE_ENABLED")
	defer func() {
		os.Setenv("F8_DEVELOPER_MODE_ENABLED", existingEnvironment)
		resetConfiguration()
	}()
	os.Unsetenv("F8_DEVELOPER_MODE_ENABLED")

	checkURLValidation(t, "F8_AUTH_URL", "Auth service")
}

func checkURLValidation(t *testing.T, envName, serviceName string) {
	resource.Require(t, resource.UnitTest)

	existingEnvironment := os.Getenv(envName)
	defer func() {
		os.Setenv(envName, existingEnvironment)
		resetConfiguration()
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

func TestGetSentryDSNOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	constSentryDSN := "F8_SENTRY_DSN"
	existingDSN := os.Getenv(constSentryDSN)
	defer func() {
		os.Setenv(constSentryDSN, existingDSN)
		resetConfiguration()
	}()

	os.Unsetenv(constSentryDSN)
	assert.Equal(t, "", config.GetSentryDSN())

	os.Setenv(constSentryDSN, "something")
	assert.Equal(t, "something", config.GetSentryDSN())
}

func TestLoadDefaultClusterConfiguration(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	clusters := config.GetClusters()
	checkClusterConfiguration(t, clusters)

	cluster := config.GetClusterByURL("https://api.starter-us-east-2.openshift.com/")
	assert.NotNil(t, cluster)
	cluster = config.GetClusterByURL("https://api.starter-us-east-2.openshift.com/")
	assert.NotNil(t, cluster)
	cluster = config.GetClusterByURL("https://api.starter-us-east-2.openshift.com/path/")
	assert.NotNil(t, cluster)
	cluster = config.GetClusterByURL("https://api.starter-us-east-2.openshift.unknown")
	assert.Nil(t, cluster)
}

func TestLoadClusterConfigurationFromFile(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	clusterConfig, err := configuration.NewConfigurationData("", "./conf-files/oso-clusters.conf")
	require.Nil(t, err)
	clusters := clusterConfig.GetClusters()
	checkClusterConfiguration(t, clusters)
}

func TestClusterConfigurationWithMissingKeys(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	_, err := configuration.NewConfigurationData("", "./conf-files/tests/oso-clusters-missing-keys.conf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "key name is missing")
	assert.Contains(t, err.Error(), "key app-dns is missing")
	assert.Contains(t, err.Error(), "key service-account-token is missing")
	assert.Contains(t, err.Error(), "key service-account-username is missing")
	assert.Contains(t, err.Error(), "key token-provider-id is missing")
	assert.Contains(t, err.Error(), "key auth-client-id is missing")
	assert.Contains(t, err.Error(), "key auth-client-secret is missing")
	assert.Contains(t, err.Error(), "key auth-client-default-scope is missing")
}

func TestClusterConfigurationWithGeneratedURLs(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	clusterConfig, err := configuration.NewConfigurationData("", "./conf-files/tests/oso-clusters-custom-urls.conf")
	require.Nil(t, err)
	checkCluster(t, clusterConfig.GetClusters(), configuration.Cluster{
		Name:                         "us-east-2",
		APIURL:                       "https://api.starter-us-east-2.openshift.com/",
		ConsoleURL:                   "custom.console.url/",
		MetricsURL:                   "custom.metrics.url/",
		LoggingURL:                   "custom.logging.url/",
		AppDNS:                       "8a09.starter-us-east-2.openshiftapps.com",
		ServiceAccountToken:          "fX0nH3d68LQ6SK5wBE6QeKJ6X8AZGVQO3dGQZZETakhmgmWAqr2KDFXE65KUwBO69aWoq",
		ServiceAccountUsername:       "dsaas",
		ServiceAccountTokenEncrypted: configuration.PointerToBool(false),
		TokenProviderID:              "f867ac10-5e05-4359-a0c6-b855ece59090",
		AuthClientID:                 "autheast2",
		AuthClientSecret:             "autheast2secret",
		AuthClientDefaultScope:       "user:full",
		Type:                         cluster.OSO,
	})
}

func TestClusterConfigurationWithEmptyArray(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	_, err := configuration.NewConfigurationData("", "./conf-files/tests/oso-clusters-empty.conf")
	require.Error(t, err)
	assert.Equal(t, err.Error(), "empty cluster config file")
}

func TestClusterConfigurationFromInvalidFile(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	_, err := configuration.NewConfigurationData("", "./conf-files/tests/oso-clusters-invalid.conf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load the JSON config file")

	_, err = configuration.NewConfigurationData("", "./conf-files/tests/oso-clusters-wrong-json.conf")
	require.Error(t, err)
	assert.Equal(t, err.Error(), "empty cluster config file")
}

func checkClusterConfiguration(t *testing.T, clusters map[string]configuration.Cluster) {
	checkCluster(t, clusters, configuration.Cluster{
		Name:                         "us-east-2",
		APIURL:                       "https://api.starter-us-east-2.openshift.com/",
		ConsoleURL:                   "https://console.starter-us-east-2.openshift.com/console/",
		MetricsURL:                   "https://metrics.starter-us-east-2.openshift.com/",
		LoggingURL:                   "https://console.starter-us-east-2.openshift.com/console/",
		AppDNS:                       "8a09.starter-us-east-2.openshiftapps.com",
		ServiceAccountToken:          "fX0nH3d68LQ6SK5wBE6QeKJ6X8AZGVQO3dGQZZETakhmgmWAqr2KDFXE65KUwBO69aWoq",
		ServiceAccountUsername:       "dsaas",
		ServiceAccountTokenEncrypted: configuration.PointerToBool(true),
		TokenProviderID:              "f867ac10-5e05-4359-a0c6-b855ece59090",
		AuthClientID:                 "autheast2",
		AuthClientSecret:             "autheast2secret",
		AuthClientDefaultScope:       "user:full",
		Type:                         "OSO",
		CapacityExhausted:            false,
	})
	checkCluster(t, clusters, configuration.Cluster{
		Name:                         "us-east-2a",
		APIURL:                       "https://api.starter-us-east-2a.openshift.com/",
		ConsoleURL:                   "https://console.starter-us-east-2a.openshift.com/console/",
		MetricsURL:                   "https://metrics.starter-us-east-2a.openshift.com/",
		LoggingURL:                   "https://console.starter-us-east-2a.openshift.com/console/",
		AppDNS:                       "b542.starter-us-east-2a.openshiftapps.com",
		ServiceAccountToken:          "ak61T6RSAacWFruh1vZP8cyUOBtQ3Chv1rdOBddSuc9nZ2wEcs81DHXRO55NpIpVQ8uiH",
		ServiceAccountUsername:       "dsaas",
		ServiceAccountTokenEncrypted: configuration.PointerToBool(true),
		TokenProviderID:              "886c7ea3-ef97-443d-b345-de94b94bb65d",
		AuthClientID:                 "autheast2a",
		AuthClientSecret:             "autheast2asecret",
		AuthClientDefaultScope:       "user:full",
		Type:                         "OSO",
		CapacityExhausted:            false,
	})
	checkCluster(t, clusters, configuration.Cluster{
		Name:                         "us-east-1a",
		APIURL:                       "https://api.starter-us-east-1a.openshift.com/",
		ConsoleURL:                   "https://console.starter-us-east-1a.openshift.com/console/",
		MetricsURL:                   "https://metrics.starter-us-east-1a.openshift.com/",
		LoggingURL:                   "https://console.starter-us-east-1a.openshift.com/console/",
		AppDNS:                       "b542.starter-us-east-1a.openshiftapps.com",
		ServiceAccountToken:          "sdfjdlfjdfkjdlfjd12324434543085djdfjd084508gfdkjdofkjg43854085dlkjdlk",
		ServiceAccountUsername:       "dsaas",
		ServiceAccountTokenEncrypted: configuration.PointerToBool(false),
		TokenProviderID:              "886c7ea3-ef97-443d-b345-de94b94bb65d",
		AuthClientID:                 "autheast1a",
		AuthClientSecret:             "autheast1asecret",
		AuthClientDefaultScope:       "user:full",
		Type:                         "OSO",
		CapacityExhausted:            true,
	})
	checkCluster(t, clusters, configuration.Cluster{
		Name:                         "us-east-3a",
		APIURL:                       "https://api.starter-us-east-3a.openshift.com/",
		ConsoleURL:                   "https://console.starter-us-east-3a.openshift.com/console/",
		MetricsURL:                   "https://metrics.starter-us-east-3a.openshift.com/",
		LoggingURL:                   "https://console.starter-us-east-3a.openshift.com/console/",
		AppDNS:                       "b542.starter-us-east-3a.openshiftapps.com",
		ServiceAccountToken:          "fkdjhfdsjfgfdjlsflhjgsafgskfdsagrwgwerwshbdjasbdjbsahdbsagbdyhsbdesbh",
		ServiceAccountUsername:       "dsaas",
		ServiceAccountTokenEncrypted: configuration.PointerToBool(true),
		TokenProviderID:              "1c09073a-13ad-4add-b0ff-197eaf18fc37",
		AuthClientID:                 "autheast3a",
		AuthClientSecret:             "autheast3asecret",
		AuthClientDefaultScope:       "user:full",
		Type:                         "OSD",
		CapacityExhausted:            false,
	})
}

func checkCluster(t *testing.T, clusters map[string]configuration.Cluster, expected configuration.Cluster) {
	require.Contains(t, clusters, expected.APIURL)
	require.Equal(t, expected, clusters[expected.APIURL])
	_, err := uuid.FromString(clusters[expected.APIURL].TokenProviderID)
	require.Nil(t, err)
}
