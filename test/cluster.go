package test

import (
	"context"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/fabric8-services/fabric8-cluster/cluster/repository"

	"github.com/fabric8-services/fabric8-cluster/app"
	"github.com/fabric8-services/fabric8-cluster/configuration"
	"github.com/fabric8-services/fabric8-common/httpsupport"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func CreateCluster(t *testing.T, db *gorm.DB) *repository.Cluster {

	cluster := NewCluster()
	repo := repository.NewClusterRepository(db)

	err := repo.Create(context.Background(), cluster)
	require.NoError(t, err)

	cls, err := repo.Load(context.Background(), cluster.ClusterID)
	require.NoError(t, err)

	AssertEqualCluster(t, cluster, cls)
	return cluster
}

// NewCluster returns a new cluster with random values for all fields
func NewCluster() *repository.Cluster {
	return &repository.Cluster{
		AppDNS:            uuid.NewV4().String(),
		AuthClientID:      uuid.NewV4().String(),
		AuthClientSecret:  uuid.NewV4().String(),
		AuthDefaultScope:  uuid.NewV4().String(),
		ConsoleURL:        uuid.NewV4().String(),
		LoggingURL:        uuid.NewV4().String(),
		MetricsURL:        uuid.NewV4().String(),
		Name:              uuid.NewV4().String(),
		SAToken:           uuid.NewV4().String(),
		SAUsername:        uuid.NewV4().String(),
		SATokenEncrypted:  true,
		TokenProviderID:   uuid.NewV4().String(),
		Type:              uuid.NewV4().String(),
		URL:               "http://" + uuid.NewV4().String() + "/",
		CapacityExhausted: false,
	}
}

// NormalizeCluster a function to normalize one or more field in a given cluster
type NormalizeCluster func(repository.Cluster) repository.Cluster

// AddTrailingSlashes a normalize function which converts all URL by adding a trailing
// slash if needed
var AddTrailingSlashes = func(source repository.Cluster) repository.Cluster {
	source.URL = httpsupport.AddTrailingSlashToURL(source.URL)
	source.ConsoleURL = httpsupport.AddTrailingSlashToURL(source.ConsoleURL)
	source.LoggingURL = httpsupport.AddTrailingSlashToURL(source.LoggingURL)
	source.MetricsURL = httpsupport.AddTrailingSlashToURL(source.MetricsURL)
	return source
}

// Normalize returns a new cluster based on the source option, with normalization functions
// applied
func Normalize(source repository.Cluster, changes ...NormalizeCluster) repository.Cluster {
	result := source
	for _, normalize := range changes {
		result = normalize(result)
	}
	return result
}

// AssertClusters verifies that the `actual` cluster belongs to the `expected`
func AssertClusters(t *testing.T, expected []repository.Cluster, actual *repository.Cluster) {
	for _, e := range expected {
		if e.ClusterID == actual.ClusterID {
			AssertEqualCluster(t, &e, actual)
			return
		}
	}
	// no match found
	assert.Fail(t, "cluster with ID '%s' couldn't be found in %v", actual.ClusterID, expected)
}

// AssertEqualClusters verifies that all the `actual` and `expected` clusters are have the same values
// including sensitive details if `expectSensitiveInfo` is `true`
func AssertEqualClusters(t *testing.T, expected, actual []repository.Cluster) {
	require.Len(t, actual, len(expected))
	for _, a := range actual {
		e := FilterClusterByURL(a.URL, expected)
		AssertEqualCluster(t, e, &a)
	}
}

// AssertEqualCluster verifies that the `actual` and `expected` clusters are have the same values
// including sensitive details if `expectSensitiveInfo` is `true`
func AssertEqualCluster(t *testing.T, expected, actual *repository.Cluster) {
	assert.Equal(t, expected.ClusterID, actual.ClusterID)
	AssertEqualClusterDetails(t, expected, actual)
}

// AssertEqualClusterDetails verifies that the `actual` and `expected` clusters are have the same values
// including sensitive details if `expectSensitiveInfo` is `true`
func AssertEqualClusterDetails(t *testing.T, expected, actual *repository.Cluster) {
	require.NotNil(t, expected)
	require.NotNil(t, actual)
	assert.Equal(t, expected.URL, actual.URL)
	assert.Equal(t, expected.AppDNS, actual.AppDNS)
	assert.Equal(t, expected.Type, actual.Type)
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.MetricsURL, actual.MetricsURL)
	assert.Equal(t, expected.LoggingURL, actual.LoggingURL)
	assert.Equal(t, expected.ConsoleURL, actual.ConsoleURL)
	assert.Equal(t, expected.CapacityExhausted, actual.CapacityExhausted)
	assert.Equal(t, expected.AuthDefaultScope, actual.AuthDefaultScope)
	assert.Equal(t, expected.AuthClientID, actual.AuthClientID)
	assert.Equal(t, expected.AuthClientSecret, actual.AuthClientSecret)
	assert.Equal(t, expected.TokenProviderID, actual.TokenProviderID)
	assert.Equal(t, expected.SAUsername, actual.SAUsername)
	assert.Equal(t, expected.SAToken, actual.SAToken)
	assert.Equal(t, expected.SATokenEncrypted, actual.SATokenEncrypted)
}

// AssertEqualClustersData verifies that data for all actual clusters match the expected ones
func AssertEqualClustersData(t *testing.T, expected []repository.Cluster, actual []*app.ClusterData) {
	require.Len(t, actual, len(expected))
	for _, c := range actual {
		require.NotNil(t, c)
		e := FilterClusterByURL(c.APIURL, expected)
		require.NotNil(t, e, "cluster with url %s could not found", c.APIURL)
		AssertEqualClusterData(t, e, c)
	}
}

// AssertEqualClusterData verifies that data for actual cluster match the expected one
func AssertEqualClusterData(t *testing.T, expected *repository.Cluster, actual *app.ClusterData) {
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, httpsupport.AddTrailingSlashToURL(expected.URL), actual.APIURL)
	assert.Equal(t, httpsupport.AddTrailingSlashToURL(expected.ConsoleURL), actual.ConsoleURL)
	assert.Equal(t, httpsupport.AddTrailingSlashToURL(expected.MetricsURL), actual.MetricsURL)
	assert.Equal(t, httpsupport.AddTrailingSlashToURL(expected.LoggingURL), actual.LoggingURL)
	assert.Equal(t, expected.AppDNS, actual.AppDNS)
	assert.Equal(t, expected.Type, actual.Type)
	assert.Equal(t, expected.CapacityExhausted, actual.CapacityExhausted)
}

// AssertEqualFullClustersData verifies that data for all actual clusters match the expected ones
func AssertEqualFullClustersData(t *testing.T, expected []repository.Cluster, actual []*app.FullClusterData) {
	require.Len(t, actual, len(expected))
	for _, c := range actual {
		require.NotNil(t, c)
		e := FilterClusterByURL(c.APIURL, expected)
		require.NotNil(t, e, "cluster with url %s could not found", c.APIURL)
		AssertEqualFullClusterData(t, e, c)
	}
}

// AssertEqualFullClusterData verifies that data for actual cluster match the expected one
func AssertEqualFullClusterData(t *testing.T, expected *repository.Cluster, actual *app.FullClusterData) {
	t.Logf("verifying cluster '%s': %v", actual.Name, spew.Sdump(actual))
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, httpsupport.AddTrailingSlashToURL(expected.URL), actual.APIURL)
	assert.Equal(t, httpsupport.AddTrailingSlashToURL(expected.ConsoleURL), actual.ConsoleURL)
	assert.Equal(t, httpsupport.AddTrailingSlashToURL(expected.MetricsURL), actual.MetricsURL)
	assert.Equal(t, httpsupport.AddTrailingSlashToURL(expected.LoggingURL), actual.LoggingURL)
	assert.Equal(t, expected.AppDNS, actual.AppDNS)
	assert.Equal(t, expected.Type, actual.Type)
	assert.Equal(t, expected.CapacityExhausted, actual.CapacityExhausted)
	// sensitive info
	assert.Equal(t, expected.AuthClientID, actual.AuthClientID)
	assert.Equal(t, expected.AuthDefaultScope, actual.AuthClientDefaultScope)
	assert.Equal(t, expected.AuthClientSecret, actual.AuthClientSecret)
	assert.Equal(t, expected.TokenProviderID, actual.TokenProviderID)
	require.NotNil(t, actual.SaTokenEncrypted)
	assert.Equal(t, expected.SATokenEncrypted, *actual.SaTokenEncrypted)
	assert.Equal(t, expected.SAUsername, actual.ServiceAccountUsername)
	assert.Equal(t, expected.SAToken, actual.ServiceAccountToken)
}

func FilterClusterByURL(url string, clusters []repository.Cluster) *repository.Cluster {
	for _, cluster := range clusters {
		if cluster.URL == url {
			return &cluster
		}
	}
	return nil
}

func CreateIdentityCluster(t *testing.T, db *gorm.DB, cluster *repository.Cluster, identityID *uuid.UUID) *repository.IdentityCluster {
	if cluster == nil {
		cluster = CreateCluster(t, db)
	}
	if identityID == nil {
		id := uuid.NewV4()
		identityID = &id
	}
	idCluster := &repository.IdentityCluster{
		ClusterID:  cluster.ClusterID,
		IdentityID: *identityID,
	}

	repo := repository.NewIdentityClusterRepository(db)
	err := repo.Create(context.Background(), idCluster)
	require.NoError(t, err)

	loaded, err := repo.Load(context.Background(), idCluster.IdentityID, idCluster.ClusterID)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	AssertEqualCluster(t, cluster, &loaded.Cluster)
	AssertEqualIdentityClusters(t, idCluster, loaded)

	return loaded
}

func AssertEqualIdentityClusters(t *testing.T, expected, actual *repository.IdentityCluster) {
	require.NotNil(t, expected)
	require.NotNil(t, actual)
	assert.Equal(t, expected.IdentityID, actual.IdentityID)
	assert.Equal(t, expected.ClusterID, actual.ClusterID)
}

func ClusterFromConfigurationCluster(configCluster configuration.Cluster) *repository.Cluster {
	return &repository.Cluster{
		Name:              configCluster.Name,
		URL:               httpsupport.AddTrailingSlashToURL(configCluster.APIURL),
		ConsoleURL:        httpsupport.AddTrailingSlashToURL(configCluster.ConsoleURL),
		MetricsURL:        httpsupport.AddTrailingSlashToURL(configCluster.MetricsURL),
		LoggingURL:        httpsupport.AddTrailingSlashToURL(configCluster.LoggingURL),
		AppDNS:            configCluster.AppDNS,
		CapacityExhausted: configCluster.CapacityExhausted,
		Type:              configCluster.Type,

		SAToken:          configCluster.ServiceAccountToken,
		SAUsername:       configCluster.ServiceAccountUsername,
		SATokenEncrypted: *configCluster.ServiceAccountTokenEncrypted,
		TokenProviderID:  configCluster.TokenProviderID,
		AuthClientID:     configCluster.AuthClientID,
		AuthClientSecret: configCluster.AuthClientSecret,
		AuthDefaultScope: configCluster.AuthClientDefaultScope,
	}
}
