package test

import (
	"context"
	"fmt"
	"testing"

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

	AssertEqualClusters(t, cluster, cls)
	return cluster
}

func NewCluster() *repository.Cluster {
	random := uuid.NewV4()
	return &repository.Cluster{
		ClusterID:         random,
		AppDNS:            random.String(),
		AuthClientID:      random.String(),
		AuthClientSecret:  random.String(),
		AuthDefaultScope:  random.String(),
		ConsoleURL:        random.String(),
		LoggingURL:        random.String(),
		MetricsURL:        random.String(),
		Name:              random.String(),
		SAToken:           random.String(),
		SAUsername:        random.String(),
		SATokenEncrypted:  true,
		TokenProviderID:   random.String(),
		Type:              random.String(),
		URL:               fmt.Sprintf("http://%s/", random.String()),
		CapacityExhausted: false,
	}
}

// AssertClusters verifies that the `actual` cluster belongs to the `expected` (and compares all fields)
func AssertClusters(t *testing.T, expected []repository.Cluster, actual *repository.Cluster) {
	for _, e := range expected {
		if e.ClusterID == actual.ClusterID {
			AssertEqualClusters(t, &e, actual)
			return
		}
	}
	// no match found
	assert.Fail(t, "cluster with ID '%s' couldn't be found in %v", actual.ClusterID, expected)
}

func AssertEqualClusters(t *testing.T, expected, actual *repository.Cluster) {
	AssertEqualClusterDetails(t, expected, actual)
	assert.Equal(t, expected.ClusterID, actual.ClusterID)
}

func AssertEqualClusterDetails(t *testing.T, expected, actual *repository.Cluster) {
	require.NotNil(t, expected)
	require.NotNil(t, actual)
	assert.Equal(t, expected.URL, actual.URL)
	assert.Equal(t, expected.Type, actual.Type)
	assert.Equal(t, expected.TokenProviderID, actual.TokenProviderID)
	assert.Equal(t, expected.SAUsername, actual.SAUsername)
	assert.Equal(t, expected.SAToken, actual.SAToken)
	assert.Equal(t, expected.SATokenEncrypted, actual.SATokenEncrypted)
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.MetricsURL, actual.MetricsURL)
	assert.Equal(t, expected.LoggingURL, actual.LoggingURL)
	assert.Equal(t, expected.ConsoleURL, actual.ConsoleURL)
	assert.Equal(t, expected.AuthDefaultScope, actual.AuthDefaultScope)
	assert.Equal(t, expected.AppDNS, actual.AppDNS)
	assert.Equal(t, expected.AuthClientID, actual.AuthClientID)
	assert.Equal(t, expected.AuthClientSecret, actual.AuthClientSecret)
	assert.Equal(t, expected.CapacityExhausted, actual.CapacityExhausted)
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
	AssertEqualClusters(t, cluster, &loaded.Cluster)
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
