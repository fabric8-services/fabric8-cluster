package test

import (
	"context"
	"testing"

	"github.com/fabric8-services/fabric8-cluster/cluster/repository"

	"github.com/fabric8-services/fabric8-cluster/cluster/service"
	"github.com/fabric8-services/fabric8-cluster/configuration"
	"github.com/fabric8-services/fabric8-common/httpsupport"
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
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
	return &repository.Cluster{
		AppDNS:            uuid.NewV4().String(),
		AuthClientID:      uuid.NewV4().String(),
		AuthClientSecret:  uuid.NewV4().String(),
		AuthDefaultScope:  uuid.NewV4().String(),
		ConsoleURL:        uuid.NewV4().String(),
		LoggingURL:        uuid.NewV4().String(),
		MetricsURL:        uuid.NewV4().String(),
		Name:              uuid.NewV4().String(),
		SaToken:           uuid.NewV4().String(),
		SaUsername:        uuid.NewV4().String(),
		TokenProviderID:   uuid.NewV4().String(),
		Type:              uuid.NewV4().String(),
		URL:               uuid.NewV4().String(),
		CapacityExhausted: false,
	}
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
	assert.Equal(t, expected.SaUsername, actual.SaUsername)
	assert.Equal(t, expected.SaToken, actual.SaToken)
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

func GetClusterFromOSOCluster(osoCluster configuration.OSOCluster) *repository.Cluster {
	return &repository.Cluster{
		Name:              osoCluster.Name,
		URL:               httpsupport.AddTrailingSlashToURL(osoCluster.APIURL),
		ConsoleURL:        httpsupport.AddTrailingSlashToURL(osoCluster.ConsoleURL),
		MetricsURL:        httpsupport.AddTrailingSlashToURL(osoCluster.MetricsURL),
		LoggingURL:        httpsupport.AddTrailingSlashToURL(osoCluster.LoggingURL),
		AppDNS:            osoCluster.AppDNS,
		CapacityExhausted: osoCluster.CapacityExhausted,

		SaToken:          osoCluster.ServiceAccountToken,
		SaUsername:       osoCluster.ServiceAccountUsername,
		TokenProviderID:  osoCluster.TokenProviderID,
		AuthClientID:     osoCluster.AuthClientID,
		AuthClientSecret: osoCluster.AuthClientSecret,
		AuthDefaultScope: osoCluster.AuthClientDefaultScope,
		Type:             service.OSO,
	}
}
