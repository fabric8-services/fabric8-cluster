package test

import (
	"context"
	"testing"

	"github.com/fabric8-services/fabric8-cluster/cluster/repository"

	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func CreateCluster(t *testing.T, db *gorm.DB) *repository.Cluster {

	cluster := &repository.Cluster{
		AppDNS:           uuid.NewV4().String(),
		AuthClientID:     uuid.NewV4().String(),
		AuthClientSecret: uuid.NewV4().String(),
		AuthDefaultScope: uuid.NewV4().String(),
		ConsoleURL:       uuid.NewV4().String(),
		LoggingURL:       uuid.NewV4().String(),
		MetricsURL:       uuid.NewV4().String(),
		Name:             uuid.NewV4().String(),
		SaToken:          uuid.NewV4().String(),
		SaUsername:       uuid.NewV4().String(),
		TokenProviderID:  uuid.NewV4().String(),
		Type:             uuid.NewV4().String(),
		URL:              uuid.NewV4().String(),
	}
	repo := repository.NewClusterRepository(db)

	err := repo.Create(context.Background(), cluster)
	require.NoError(t, err)

	cls, err := repo.Load(context.Background(), cluster.ClusterID)
	require.NoError(t, err)

	AssertEqualClusters(t, cluster, cls)
	return cluster
}

func AssertEqualClusters(t *testing.T, expected, actual *repository.Cluster) {
	require.NotNil(t, expected)
	require.NotNil(t, actual)
	assert.Equal(t, expected.URL, actual.URL)
	assert.Equal(t, expected.Type, actual.Type)
	assert.Equal(t, expected.TokenProviderID, actual.TokenProviderID)
	assert.Equal(t, expected.SaUsername, actual.SaUsername)
	assert.Equal(t, expected.SaToken, actual.SaToken)
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.MetricsURL, actual.MetricsURL)
	assert.Equal(t, expected.LoggingURL, actual.LoggingURL)
	assert.Equal(t, expected.ConsoleURL, actual.ConsoleURL)
	assert.Equal(t, expected.AuthDefaultScope, actual.AuthDefaultScope)
	assert.Equal(t, expected.AppDNS, actual.AppDNS)
	assert.Equal(t, expected.AuthClientID, actual.AuthClientID)
	assert.Equal(t, expected.ClusterID, actual.ClusterID)
	assert.Equal(t, expected.AuthClientSecret, actual.AuthClientSecret)
}
