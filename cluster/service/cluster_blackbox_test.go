package service_test

import (
	"context"
	"github.com/fabric8-services/fabric8-cluster/cluster/repository"
	clustersvc "github.com/fabric8-services/fabric8-cluster/cluster/service"
	"github.com/fabric8-services/fabric8-cluster/gormtestsupport"
	"github.com/fabric8-services/fabric8-cluster/test"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
)

func TestCluster(t *testing.T) {
	suite.Run(t, &ClusterServiceTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

type ClusterServiceTestSuite struct {
	gormtestsupport.DBTestSuite
}

func (s *ClusterServiceTestSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
}

func (s *ClusterServiceTestSuite) TestCreateOrSaveOSOClusterOK() {
	err := s.Application.ClusterService().CreateOrSaveOSOClusterFromConfig(context.Background())
	require.NoError(s.T(), err)

	clusters, err := s.Application.Clusters().Query(func(db *gorm.DB) *gorm.DB {
		return db.Where("type = ?", clustersvc.OSO)
	})
	require.NoError(s.T(), err)
	assert.Len(s.T(), clusters, 3)

	verifyClusters(s.T(), clusters)
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
		if c.URL == url {
			return &c
		}
	}
	return nil
}
