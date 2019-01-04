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

	AssertEqualClusters(t, cluster, cls, true)
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

func AssertEqualClusters(t *testing.T, expected, actual *repository.Cluster, compareSensitiveInfo bool) {
	AssertEqualClusterDetails(t, expected, actual, compareSensitiveInfo)
	assert.Equal(t, expected.ClusterID, actual.ClusterID)
}

func AssertEqualClusterDetails(t *testing.T, expected, actual *repository.Cluster, compareSensitiveInfo bool) {
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
	if compareSensitiveInfo {
		assert.Equal(t, expected.AuthDefaultScope, actual.AuthDefaultScope)
		assert.Equal(t, expected.AuthClientID, actual.AuthClientID)
		assert.Equal(t, expected.AuthClientSecret, actual.AuthClientSecret)
		assert.Equal(t, expected.TokenProviderID, actual.TokenProviderID)
		assert.Equal(t, expected.SAUsername, actual.SAUsername)
		assert.Equal(t, expected.SAToken, actual.SAToken)
		assert.Equal(t, expected.SATokenEncrypted, actual.SATokenEncrypted)
	}
}

func AssertEqualClusterData(t *testing.T, clusters []repository.Cluster, clusterList []*app.ClusterData) {
	require.Len(t, clusterList, len(clusters))

	for _, c := range clusterList {
		require.NotNil(t, c)
		apiURL := c.APIURL
		require.NotNil(t, apiURL)
		cluster := FilterClusterByURL(apiURL, clusters)
		require.NotNil(t, cluster, "cluster with url %s could not found", apiURL)
		assert.Equal(t, cluster.Name, c.Name)
		assert.Equal(t, cluster.URL, apiURL)
		assert.Equal(t, cluster.ConsoleURL, c.ConsoleURL)
		assert.Equal(t, cluster.MetricsURL, c.MetricsURL)
		assert.Equal(t, cluster.LoggingURL, c.LoggingURL)
		assert.Equal(t, cluster.AppDNS, c.AppDNS)
		assert.Equal(t, cluster.Type, c.Type)
		assert.Equal(t, cluster.CapacityExhausted, c.CapacityExhausted)
	}
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
	AssertEqualClusters(t, cluster, &loaded.Cluster, true)
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
