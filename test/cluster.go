package test

import (
	"context"
	"testing"

	"github.com/pkg/errors"

	"github.com/fabric8-services/fabric8-cluster/cluster/repository"

	"github.com/fabric8-services/fabric8-cluster/app"
	"github.com/fabric8-services/fabric8-common/httpsupport"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CreateCluster returns a new cluster after saves it in the DB
func CreateCluster(t *testing.T, db *gorm.DB) repository.Cluster {
	cluster := NewCluster()
	repo := repository.NewClusterRepository(db)
	err := repo.Create(context.Background(), &cluster)
	require.NoError(t, err)
	// verify that the cluster exists in the DB
	cls, err := repo.Load(context.Background(), cluster.ClusterID)
	require.NoError(t, err)
	AssertEqualCluster(t, cluster, *cls, true)
	return cluster
}

// NewCluster returns a new cluster with random values for all fields
func NewCluster() repository.Cluster {
	return repository.Cluster{
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

// AssertClusters verifies that the `actual` cluster belongs to the `expected`,
// and compares all fields including sensitive details if `expectSensitiveInfo` is `true`
func AssertClusters(t *testing.T, expected []repository.Cluster, actual repository.Cluster, expectSensitiveInfo bool) {
	for _, e := range expected {
		if e.ClusterID == actual.ClusterID {
			AssertEqualCluster(t, e, actual, expectSensitiveInfo)
			return
		}
	}
	// no match found
	assert.Fail(t, "cluster with ID '%s' couldn't be found in %v", actual.ClusterID, expected)
}

// AssertEqualClusters verifies that all the `actual` and `expected` clusters are have the same values
// including sensitive details if `expectSensitiveInfo` is `true`
func AssertEqualClusters(t *testing.T, expected, actual []repository.Cluster, expectSensitiveInfo bool) {
	require.Len(t, actual, len(expected))
	for _, a := range actual {
		e, err := FilterClusterByURL(a.URL, expected)
		require.NoError(t, err)
		AssertEqualCluster(t, e, a, expectSensitiveInfo)
	}
}

// AssertEqualCluster verifies that the `actual` and `expected` clusters are have the same values
// including sensitive details if `expectSensitiveInfo` is `true`
func AssertEqualCluster(t *testing.T, expected, actual repository.Cluster, expectSensitiveInfo bool) {
	assert.Equal(t, expected.ClusterID, actual.ClusterID)
	AssertEqualClusterDetails(t, expected, actual, expectSensitiveInfo)
}

// AssertEqualClusterDetails verifies that the `actual` and `expected` clusters are have the same values
// including sensitive details if `expectSensitiveInfo` is `true`
func AssertEqualClusterDetails(t *testing.T, expected, actual repository.Cluster, expectSensitiveInfo bool) {
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
	if expectSensitiveInfo {
		assert.Equal(t, expected.AuthDefaultScope, actual.AuthDefaultScope)
		assert.Equal(t, expected.AuthClientID, actual.AuthClientID)
		assert.Equal(t, expected.AuthClientSecret, actual.AuthClientSecret)
		assert.Equal(t, expected.TokenProviderID, actual.TokenProviderID)
		assert.Equal(t, expected.SAUsername, actual.SAUsername)
		assert.Equal(t, expected.SAToken, actual.SAToken)
		assert.Equal(t, expected.SATokenEncrypted, actual.SATokenEncrypted)
	} else {
		assert.Equal(t, "", actual.AuthDefaultScope)
		assert.Equal(t, "", actual.AuthClientID)
		assert.Equal(t, "", actual.AuthClientSecret)
		assert.Equal(t, "", actual.TokenProviderID)
		assert.Equal(t, "", actual.SAUsername)
		assert.Equal(t, "", actual.SAToken)
		assert.Equal(t, false, actual.SATokenEncrypted)
	}
}

// AssertEqualClustersData verifies that data for all actual clusters match the expected ones
func AssertEqualClustersData(t *testing.T, expected []repository.Cluster, actual []*app.ClusterData) {
	require.Len(t, actual, len(expected))
	for _, a := range actual {
		require.NotNil(t, a)
		expected, err := FilterClusterByURL(a.APIURL, expected)
		require.NoError(t, err)
		AssertEqualClusterData(t, expected, *a)
	}
}

// AssertEqualClusterData verifies that data for actual cluster match the expected one
func AssertEqualClusterData(t *testing.T, expected repository.Cluster, actual app.ClusterData) {
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
	for _, a := range actual {
		require.NotNil(t, a)
		expected, err := FilterClusterByURL(a.APIURL, expected)
		require.NoError(t, err)
		AssertEqualFullClusterData(t, expected, *a)
	}
}

// AssertEqualFullClusterData verifies that data for actual cluster match the expected one
func AssertEqualFullClusterData(t *testing.T, expected repository.Cluster, actual app.FullClusterData) {
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

// FilterClusterByURL returns the cluster that has the given URL or an error if none was found
func FilterClusterByURL(url string, clusters []repository.Cluster) (repository.Cluster, error) {
	for _, cluster := range clusters {
		if cluster.URL == url {
			return cluster, nil
		}
	}
	return repository.Cluster{}, errors.Errorf("unable to find cluster with url '%s'", url)
}

// CreateIdentityClusterOption an option to configure the identity/cluster link to create
type CreateIdentityClusterOption func(*repository.IdentityCluster)

// WithCluster an option to specify the cluster to use when linking to an identity
func WithCluster(c repository.Cluster) CreateIdentityClusterOption {
	return func(ic *repository.IdentityCluster) {
		ic.Cluster = c
		ic.ClusterID = c.ClusterID
	}
}

// WithIdentityID an option to specify the identity to use when linking to a cluster
func WithIdentityID(identityID uuid.UUID) CreateIdentityClusterOption {
	return func(ic *repository.IdentityCluster) {
		ic.IdentityID = identityID
	}
}

// CreateIdentityCluster returns a new IdentityCluster after saving it in the DB.
// if no identity or cluster was provided in the options, a random UUID is used for the identity ID and
// a new cluster is created on the fly
func CreateIdentityCluster(t *testing.T, db *gorm.DB, options ...CreateIdentityClusterOption) repository.IdentityCluster {
	idCluster := repository.IdentityCluster{}
	for _, option := range options {
		option(&idCluster)
	}
	if idCluster.Cluster.ClusterID == uuid.Nil {
		c := CreateCluster(t, db)
		idCluster.Cluster = c
		idCluster.ClusterID = c.ClusterID
	}
	if idCluster.IdentityID == uuid.Nil {
		idCluster.IdentityID = uuid.NewV4()
	}
	repo := repository.NewIdentityClusterRepository(db)
	err := repo.Create(context.Background(), &idCluster)
	require.NoError(t, err)
	// verify
	loaded, err := repo.Load(context.Background(), idCluster.IdentityID, idCluster.ClusterID)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	AssertEqualCluster(t, idCluster.Cluster, loaded.Cluster, true)
	AssertEqualIdentityClusters(t, idCluster, *loaded)
	return *loaded
}

// AssertEqualIdentityClusters verifies that the identity/cluster links are equal
func AssertEqualIdentityClusters(t *testing.T, expected, actual repository.IdentityCluster) {
	assert.Equal(t, expected.IdentityID, actual.IdentityID)
	assert.Equal(t, expected.ClusterID, actual.ClusterID)
}

// // ClusterFromConfigurationCluster converts a "configuration" cluster to a "model" cluster
// func ClusterFromConfigurationCluster(c repository.Cluster) repository.Cluster {
// 	return repository.Cluster{
// 		Name:              c.Name,
// 		URL:               httpsupport.AddTrailingSlashToURL(c.URL),
// 		ConsoleURL:        httpsupport.AddTrailingSlashToURL(c.ConsoleURL),
// 		MetricsURL:        httpsupport.AddTrailingSlashToURL(c.MetricsURL),
// 		LoggingURL:        httpsupport.AddTrailingSlashToURL(c.LoggingURL),
// 		AppDNS:            c.AppDNS,
// 		CapacityExhausted: c.CapacityExhausted,
// 		Type:              c.Type,

// 		SAToken:          c.SAToken,
// 		SAUsername:       c.SAUsername,
// 		SATokenEncrypted: c.SATokenEncrypted,
// 		TokenProviderID:  c.TokenProviderID,
// 		AuthClientID:     c.AuthClientID,
// 		AuthClientSecret: c.AuthClientSecret,
// 		AuthDefaultScope: c.AuthDefaultScope,
// 	}
// }
