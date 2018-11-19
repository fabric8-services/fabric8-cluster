package repository_test

import (
	"context"
	"github.com/satori/go.uuid"
	"testing"

	"github.com/fabric8-services/fabric8-cluster/cluster/repository"
	"github.com/fabric8-services/fabric8-cluster/gormtestsupport"
	"github.com/fabric8-services/fabric8-cluster/test"
	"github.com/fabric8-services/fabric8-common/errors"

	"fmt"
	"github.com/fabric8-services/fabric8-cluster/configuration"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type clusterTestSuite struct {
	gormtestsupport.DBTestSuite
	repo repository.ClusterRepository
}

func TestCluster(t *testing.T) {
	suite.Run(t, &clusterTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *clusterTestSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.repo = repository.NewClusterRepository(s.DB)
}

func (s *clusterTestSuite) TestCreateAndLoadClusterOK() {
	cluster1 := test.CreateCluster(s.T(), s.DB)
	test.CreateCluster(s.T(), s.DB) // noise
	loaded, err := s.repo.Load(context.Background(), cluster1.ClusterID)
	require.NoError(s.T(), err)

	test.AssertEqualClusters(s.T(), cluster1, loaded)
}

func (s *clusterTestSuite) TestCreateAndLoadClusterByURLOK() {
	cluster1 := test.CreateCluster(s.T(), s.DB)
	test.CreateCluster(s.T(), s.DB) // noise
	loaded, err := s.repo.LoadClusterByURL(context.Background(), cluster1.URL)
	require.NoError(s.T(), err)

	test.AssertEqualClusters(s.T(), cluster1, loaded)
}

func (s *clusterTestSuite) TestCreateAndLoadClusterByURLFail() {
	test.CreateCluster(s.T(), s.DB)
	test.CreateCluster(s.T(), s.DB) // noise
	loaded, err := s.repo.LoadClusterByURL(context.Background(), uuid.NewV4().String())
	assert.Nil(s.T(), loaded)
	test.AssertError(s.T(), err, gorm.ErrRecordNotFound, "record not found")
}

func (s *clusterTestSuite) TestCreateOKInCreateOrSave() {
	cluster := test.NewCluster()
	s.repo.CreateOrSave(context.Background(), cluster)
	test.CreateCluster(s.T(), s.DB) // noise

	loaded, err := s.repo.LoadClusterByURL(context.Background(), cluster.URL)
	require.NoError(s.T(), err)

	test.AssertEqualClusters(s.T(), cluster, loaded)
}

func (s *clusterTestSuite) TestSaveOKInCreateOrSave() {
	cluster := test.NewCluster()
	test.CreateCluster(s.T(), s.DB) // noise
	s.repo.CreateOrSave(context.Background(), cluster)

	loaded, err := s.repo.LoadClusterByURL(context.Background(), cluster.URL)
	require.NoError(s.T(), err)

	test.AssertEqualClusters(s.T(), cluster, loaded)

	// update cluster details
	cluster.AppDNS = uuid.NewV4().String()
	cluster.AuthClientID = uuid.NewV4().String()
	cluster.AuthClientSecret = uuid.NewV4().String()
	cluster.AuthDefaultScope = uuid.NewV4().String()
	cluster.ConsoleURL = uuid.NewV4().String()
	cluster.LoggingURL = uuid.NewV4().String()
	cluster.MetricsURL = uuid.NewV4().String()
	cluster.Name = uuid.NewV4().String()
	cluster.SaToken = uuid.NewV4().String()
	cluster.SaUsername = uuid.NewV4().String()
	cluster.TokenProviderID = uuid.NewV4().String()
	cluster.Type = uuid.NewV4().String()

	s.repo.CreateOrSave(context.Background(), cluster)
	loaded, err = s.repo.LoadClusterByURL(context.Background(), cluster.URL)
	require.NoError(s.T(), err)

	test.AssertEqualClusters(s.T(), cluster, loaded)
}

func (s *clusterTestSuite) TestCreateOrSaveOSOClusterOK() {
	clusterConfig, err := configuration.NewConfigurationData("", "./../../configuration/conf-files/oso-clusters.conf")
	fmt.Println(clusterConfig.GetOSOClusters())
	require.Nil(s.T(), err)
	s.repo.CreateOrSaveOSOClusterFromConfig(context.Background(), clusterConfig)

	clusters, err := s.repo.Query(func(db *gorm.DB) *gorm.DB {
		return db.Where("type = ?", repository.OSO)
	})

	require.NoError(s.T(), err)
	assert.Len(s.T(), clusters, 3)

	verifyClusters(s.T(), clusters)
}

func (s *clusterTestSuite) TestDeleteOK() {
	cluster1 := test.CreateCluster(s.T(), s.DB)
	cluster2 := test.CreateCluster(s.T(), s.DB) // noise
	err := s.repo.Delete(context.Background(), cluster1.ClusterID)
	require.NoError(s.T(), err)

	_, err = s.repo.Load(context.Background(), cluster1.ClusterID)
	test.AssertError(s.T(), err, errors.NotFoundError{}, "cluster with id '%s' not found", cluster1.ClusterID)

	loaded, err := s.repo.Load(context.Background(), cluster2.ClusterID)
	require.NoError(s.T(), err)
	test.AssertEqualClusters(s.T(), cluster2, loaded)
}

func (s *clusterTestSuite) TestDeleteUnknownFails() {
	id := uuid.NewV4()
	err := s.repo.Delete(context.Background(), id)
	test.AssertError(s.T(), err, errors.NotFoundError{}, "cluster with id '%s' not found", id)
}

func (s *clusterTestSuite) TestLoadUnknownFails() {
	id := uuid.NewV4()
	_, err := s.repo.Load(context.Background(), id)
	test.AssertError(s.T(), err, errors.NotFoundError{}, "cluster with id '%s' not found", id)
}

func (s *clusterTestSuite) TestSaveOK() {
	cluster1 := test.CreateCluster(s.T(), s.DB)
	cluster2 := test.CreateCluster(s.T(), s.DB) // noise

	cluster1.AppDNS = uuid.NewV4().String()
	cluster1.AuthClientID = uuid.NewV4().String()
	cluster1.AuthClientSecret = uuid.NewV4().String()
	cluster1.AuthDefaultScope = uuid.NewV4().String()
	cluster1.ConsoleURL = uuid.NewV4().String()
	cluster1.LoggingURL = uuid.NewV4().String()
	cluster1.MetricsURL = uuid.NewV4().String()
	cluster1.Name = uuid.NewV4().String()
	cluster1.SaToken = uuid.NewV4().String()
	cluster1.SaUsername = uuid.NewV4().String()
	cluster1.TokenProviderID = uuid.NewV4().String()
	cluster1.Type = uuid.NewV4().String()
	cluster1.URL = uuid.NewV4().String()

	err := s.repo.Save(context.Background(), cluster1)
	require.NoError(s.T(), err)

	loaded1, err := s.repo.Load(context.Background(), cluster1.ClusterID)
	test.AssertEqualClusters(s.T(), cluster1, loaded1)

	loaded2, err := s.repo.Load(context.Background(), cluster2.ClusterID)
	test.AssertEqualClusters(s.T(), cluster2, loaded2)
}

func (s *clusterTestSuite) TestSaveUnknownFails() {
	id := uuid.NewV4()
	err := s.repo.Save(context.Background(), &repository.Cluster{ClusterID: id})
	test.AssertError(s.T(), err, errors.NotFoundError{}, "cluster with id '%s' not found", id)
}

func (s *clusterTestSuite) TestExists() {
	id := uuid.NewV4()
	err := s.repo.CheckExists(context.Background(), id.String())
	test.AssertError(s.T(), err, errors.NotFoundError{}, "cluster with id '%s' not found", id)

	cluster := test.CreateCluster(s.T(), s.DB)
	err = s.repo.CheckExists(context.Background(), cluster.ClusterID.String())
	require.NoError(s.T(), err)
}

func (s *clusterTestSuite) TestQueryOK() {
	cluster1 := test.CreateCluster(s.T(), s.DB)

	clusters, err := s.repo.Query(func(db *gorm.DB) *gorm.DB {
		return db.Where("cluster_id = ?", cluster1.ClusterID)
	})

	require.NoError(s.T(), err)
	require.Len(s.T(), clusters, 1)
	test.AssertEqualClusters(s.T(), cluster1, &clusters[0])
}

func verifyClusters(t *testing.T, clusters []repository.Cluster) {
	verifyCluster(t, clusters, &repository.Cluster{
		Name:             "us-east-2",
		URL:              "https://api.starter-us-east-2.openshift.com/",
		ConsoleURL:       "https://console.starter-us-east-2.openshift.com/console/",
		MetricsURL:       "https://metrics.starter-us-east-2.openshift.com/",
		LoggingURL:       "https://console.starter-us-east-2.openshift.com/console/",
		AppDNS:           "8a09.starter-us-east-2.openshiftapps.com",
		SaToken:          "fX0nH3d68LQ6SK5wBE6QeKJ6X8AZGVQO3dGQZZETakhmgmWAqr2KDFXE65KUwBO69aWoq",
		SaUsername:       "dsaas",
		TokenProviderID:  "f867ac10-5e05-4359-a0c6-b855ece59090",
		AuthClientID:     "autheast2",
		AuthClientSecret: "autheast2secret",
		AuthDefaultScope: "user:full",
		Type:             "OSO",
		//CapacityExhausted:      false,
	})
	verifyCluster(t, clusters, &repository.Cluster{
		Name:             "us-east-2a",
		URL:              "https://api.starter-us-east-2a.openshift.com/",
		ConsoleURL:       "https://console.starter-us-east-2a.openshift.com/console/",
		MetricsURL:       "https://metrics.starter-us-east-2a.openshift.com/",
		LoggingURL:       "https://console.starter-us-east-2a.openshift.com/console/",
		AppDNS:           "b542.starter-us-east-2a.openshiftapps.com",
		SaToken:          "ak61T6RSAacWFruh1vZP8cyUOBtQ3Chv1rdOBddSuc9nZ2wEcs81DHXRO55NpIpVQ8uiH",
		SaUsername:       "dsaas",
		TokenProviderID:  "886c7ea3-ef97-443d-b345-de94b94bb65d",
		AuthClientID:     "autheast2a",
		AuthClientSecret: "autheast2asecret",
		AuthDefaultScope: "user:full",
		Type:             "OSO",
		//CapacityExhausted:      false,
	})
	verifyCluster(t, clusters, &repository.Cluster{
		Name:             "us-east-1a",
		URL:              "https://api.starter-us-east-1a.openshift.com/",
		ConsoleURL:       "https://console.starter-us-east-1a.openshift.com/console/",
		MetricsURL:       "https://metrics.starter-us-east-1a.openshift.com/",
		LoggingURL:       "https://console.starter-us-east-1a.openshift.com/console/",
		AppDNS:           "b542.starter-us-east-1a.openshiftapps.com",
		SaToken:          "sdfjdlfjdfkjdlfjd12324434543085djdfjd084508gfdkjdofkjg43854085dlkjdlk",
		SaUsername:       "dsaas",
		TokenProviderID:  "886c7ea3-ef97-443d-b345-de94b94bb65d",
		AuthClientID:     "autheast1a",
		AuthClientSecret: "autheast1asecret",
		AuthDefaultScope: "user:full",
		Type:             "OSO",
		//CapacityExhausted:      true,
	})
}

func verifyCluster(t *testing.T, clusters []repository.Cluster, expected *repository.Cluster) {
	cluster := getCluster(clusters, expected.URL)
	test.AssertEqualClusterDetails(t, expected, cluster)
}

func getCluster(clusters []repository.Cluster, url string) *repository.Cluster {
	for _, c := range clusters {
		fmt.Println(c.URL, url)
		if c.URL == url {
			return &c
		}
	}
	return nil
}
