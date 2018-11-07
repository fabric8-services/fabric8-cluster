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
	GetOSOClusters() map[string]configuration.OSOCluster
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
	for _, clusterConfig := range c.config.GetOSOClusters() {
		cluster := &app.ClusterData{
			Name:              clusterConfig.Name,
			APIURL:            httpsupport.AddTrailingSlashToURL(clusterConfig.APIURL),
			ConsoleURL:        httpsupport.AddTrailingSlashToURL(clusterConfig.ConsoleURL),
			MetricsURL:        httpsupport.AddTrailingSlashToURL(clusterConfig.MetricsURL),
			LoggingURL:        httpsupport.AddTrailingSlashToURL(clusterConfig.LoggingURL),
			AppDNS:            clusterConfig.AppDNS,
			CapacityExhausted: clusterConfig.CapacityExhausted,
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
	for _, clusterConfig := range c.config.GetOSOClusters() {
		cluster := &app.FullClusterData{
			Name:              clusterConfig.Name,
			APIURL:            httpsupport.AddTrailingSlashToURL(clusterConfig.APIURL),
			ConsoleURL:        httpsupport.AddTrailingSlashToURL(clusterConfig.ConsoleURL),
			MetricsURL:        httpsupport.AddTrailingSlashToURL(clusterConfig.MetricsURL),
			LoggingURL:        httpsupport.AddTrailingSlashToURL(clusterConfig.LoggingURL),
			AppDNS:            clusterConfig.AppDNS,
			CapacityExhausted: clusterConfig.CapacityExhausted,

			AuthClientDefaultScope: clusterConfig.AuthClientDefaultScope,
			AuthClientID:           clusterConfig.AuthClientID,
			AuthClientSecret:       clusterConfig.AuthClientSecret,
			ServiceAccountToken:    clusterConfig.ServiceAccountToken,
			ServiceAccountUsername: clusterConfig.ServiceAccountUsername,
			TokenProviderID:        clusterConfig.TokenProviderID,
		}
		data = append(data, cluster)
	}
	clusters := app.FullClusterList{
		Data: data,
	}
	return ctx.OK(&clusters)
}
