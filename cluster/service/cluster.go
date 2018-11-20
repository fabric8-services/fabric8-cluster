package service

import (
	"context"
	"github.com/fabric8-services/fabric8-cluster/application/service"
	"github.com/fabric8-services/fabric8-cluster/application/service/base"
	servicectx "github.com/fabric8-services/fabric8-cluster/application/service/context"
	"github.com/fabric8-services/fabric8-cluster/cluster/repository"
	"github.com/fabric8-services/fabric8-cluster/configuration"
	"github.com/fabric8-services/fabric8-common/httpsupport"
)

const (
	OSD = "OSD"
	OCP = "OCP"
	OSO = "OSO"
)

type clusterService struct {
	base.BaseService
	config ClusterConfig
}

type ClusterConfig interface {
	GetOSOClusters() map[string]configuration.OSOCluster
}

// NewClusterService creates a new cluster service
func NewClusterService(context servicectx.ServiceContext, config ClusterConfig) service.ClusterService {
	return &clusterService{
		BaseService: base.NewBaseService(context),
		config:      config,
	}
}

func (c clusterService) CreateOrSaveOSOClusterFromConfig(ctx context.Context) error {

	for _, clusterConfig := range c.config.GetOSOClusters() {
		cluster := &repository.Cluster{
			Name:       clusterConfig.Name,
			URL:        httpsupport.AddTrailingSlashToURL(clusterConfig.APIURL),
			ConsoleURL: httpsupport.AddTrailingSlashToURL(clusterConfig.ConsoleURL),
			MetricsURL: httpsupport.AddTrailingSlashToURL(clusterConfig.MetricsURL),
			LoggingURL: httpsupport.AddTrailingSlashToURL(clusterConfig.LoggingURL),
			AppDNS:     clusterConfig.AppDNS,
			//CapacityExhausted: clusterConfig.CapacityExhausted,

			SaToken:          clusterConfig.ServiceAccountToken,
			SaUsername:       clusterConfig.ServiceAccountUsername,
			TokenProviderID:  clusterConfig.TokenProviderID,
			AuthClientID:     clusterConfig.AuthClientID,
			AuthClientSecret: clusterConfig.AuthClientSecret,
			AuthDefaultScope: clusterConfig.AuthClientDefaultScope,
			Type:             OSO,
		}

		err := c.ExecuteInTransaction(func() error {
			return c.Repositories().Clusters().CreateOrSave(ctx, cluster)
		})

		if err != nil {
			return err
		}
	}
	return nil
}
