package controller

import (
	"github.com/fabric8-services/fabric8-cluster/app"
	"github.com/fabric8-services/fabric8-cluster/configuration"
	"github.com/fabric8-services/fabric8-common/errors"
	"github.com/fabric8-services/fabric8-common/httpsupport"
	"github.com/fabric8-services/fabric8-common/log"
	"github.com/fabric8-services/fabric8-common/token"

	"github.com/goadesign/goa"
)

type clusterConfiguration interface {
	GetClusters() map[string]configuration.Cluster
}

// ClustersController implements the clusters resource.
type ClustersController struct {
	*goa.Controller
	config clusterConfiguration
}

// NewClustersController creates a clusters controller.
func NewClustersController(service *goa.Service, config clusterConfiguration) *ClustersController {
	return &ClustersController{
		Controller: service.NewController("ClustersController"),
		config:     config,
	}
}

// Show returns the list of available OSO clusters.
func (c *ClustersController) Show(ctx *app.ShowClustersContext) error {
	if !token.IsSpecificServiceAccount(ctx, token.OsoProxy, token.Tenant, token.JenkinsIdler, token.JenkinsProxy, token.Auth) {
		log.Error(ctx, nil, "unauthorized access to cluster info")
		return app.JSONErrorResponse(ctx, errors.NewUnauthorizedError("unauthorized access to cluster info"))
	}
	var data []*app.ClusterData
	for _, configCluster := range c.config.GetClusters() {
		cluster := &app.ClusterData{
			Name:              configCluster.Name,
			APIURL:            httpsupport.AddTrailingSlashToURL(configCluster.APIURL),
			ConsoleURL:        httpsupport.AddTrailingSlashToURL(configCluster.ConsoleURL),
			MetricsURL:        httpsupport.AddTrailingSlashToURL(configCluster.MetricsURL),
			LoggingURL:        httpsupport.AddTrailingSlashToURL(configCluster.LoggingURL),
			AppDNS:            configCluster.AppDNS,
			Type:              configCluster.Type,
			CapacityExhausted: configCluster.CapacityExhausted,
		}
		data = append(data, cluster)
	}
	clusters := app.ClusterList{
		Data: data,
	}
	return ctx.OK(&clusters)
}

// ShowAuthClient returns the list of available OSO clusters with full configuration including Auth client data.
// To be used by Auth service only
func (c *ClustersController) ShowAuthClient(ctx *app.ShowAuthClientClustersContext) error {
	if !token.IsSpecificServiceAccount(ctx, token.Auth) {
		log.Error(ctx, nil, "unauthorized access to cluster info")
		return app.JSONErrorResponse(ctx, errors.NewUnauthorizedError("unauthorized access to cluster info"))
	}
	var data []*app.FullClusterData
	for _, configCluster := range c.config.GetClusters() {
		cluster := &app.FullClusterData{
			Name:              configCluster.Name,
			APIURL:            httpsupport.AddTrailingSlashToURL(configCluster.APIURL),
			ConsoleURL:        httpsupport.AddTrailingSlashToURL(configCluster.ConsoleURL),
			MetricsURL:        httpsupport.AddTrailingSlashToURL(configCluster.MetricsURL),
			LoggingURL:        httpsupport.AddTrailingSlashToURL(configCluster.LoggingURL),
			AppDNS:            configCluster.AppDNS,
			Type:              configCluster.Type,
			CapacityExhausted: configCluster.CapacityExhausted,

			AuthClientDefaultScope: configCluster.AuthClientDefaultScope,
			AuthClientID:           configCluster.AuthClientID,
			AuthClientSecret:       configCluster.AuthClientSecret,
			ServiceAccountToken:    configCluster.ServiceAccountToken,
			ServiceAccountUsername: configCluster.ServiceAccountUsername,
			TokenProviderID:        configCluster.TokenProviderID,
		}
		data = append(data, cluster)
	}
	clusters := app.FullClusterList{
		Data: data,
	}
	return ctx.OK(&clusters)
}
