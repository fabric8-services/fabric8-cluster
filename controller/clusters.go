package controller

import (
	"github.com/fabric8-services/fabric8-cluster/app"
	"github.com/fabric8-services/fabric8-cluster/application"
	cluster "github.com/fabric8-services/fabric8-cluster/cluster/repository"
	"github.com/fabric8-services/fabric8-cluster/configuration"
	"github.com/fabric8-services/fabric8-common/auth"
	"github.com/fabric8-services/fabric8-common/errors"
	"github.com/fabric8-services/fabric8-common/httpsupport"
	"github.com/fabric8-services/fabric8-common/log"

	"github.com/goadesign/goa"
)

type clusterConfiguration interface {
	GetClusters() map[string]configuration.Cluster
}

// ClustersController implements the clusters resource.
type ClustersController struct {
	*goa.Controller
	config clusterConfiguration
	app    application.Application
}

// NewClustersController creates a clusters controller.
func NewClustersController(service *goa.Service, config clusterConfiguration, app application.Application) *ClustersController {
	return &ClustersController{
		Controller: service.NewController("ClustersController"),
		config:     config,
		app:        app,
	}
}

// Show returns the list of available OSO clusters.
func (c *ClustersController) Show(ctx *app.ShowClustersContext) error {
	if !auth.IsSpecificServiceAccount(ctx, auth.OsoProxy, auth.Tenant, auth.JenkinsIdler, auth.JenkinsProxy, auth.Auth) {
		log.Error(ctx, nil, "unauthorized access to cluster info")
		return app.JSONErrorResponse(ctx, errors.NewUnauthorizedError("unauthorized access to cluster info"))
	}
	var data []*app.ClusterData
	for _, configCluster := range c.config.GetClusters() {
		clusterData := &app.ClusterData{
			Name:              configCluster.Name,
			APIURL:            httpsupport.AddTrailingSlashToURL(configCluster.APIURL),
			ConsoleURL:        httpsupport.AddTrailingSlashToURL(configCluster.ConsoleURL),
			MetricsURL:        httpsupport.AddTrailingSlashToURL(configCluster.MetricsURL),
			LoggingURL:        httpsupport.AddTrailingSlashToURL(configCluster.LoggingURL),
			AppDNS:            configCluster.AppDNS,
			Type:              configCluster.Type,
			CapacityExhausted: configCluster.CapacityExhausted,
		}
		data = append(data, clusterData)
	}
	clusters := app.ClusterList{
		Data: data,
	}
	return ctx.OK(&clusters)
}

// ShowAuthClient returns the list of available OSO clusters with full configuration including Auth client data.
// To be used by Auth service only
func (c *ClustersController) ShowAuthClient(ctx *app.ShowAuthClientClustersContext) error {
	if !auth.IsSpecificServiceAccount(ctx, auth.Auth) {
		log.Error(ctx, nil, "unauthorized access to cluster info")
		return app.JSONErrorResponse(ctx, errors.NewUnauthorizedError("unauthorized access to cluster info"))
	}
	var data []*app.FullClusterData
	for _, configCluster := range c.config.GetClusters() {
		cluster := &app.FullClusterData{
			Name:                   configCluster.Name,
			APIURL:                 httpsupport.AddTrailingSlashToURL(configCluster.APIURL),
			ConsoleURL:             httpsupport.AddTrailingSlashToURL(configCluster.ConsoleURL),
			MetricsURL:             httpsupport.AddTrailingSlashToURL(configCluster.MetricsURL),
			LoggingURL:             httpsupport.AddTrailingSlashToURL(configCluster.LoggingURL),
			AppDNS:                 configCluster.AppDNS,
			Type:                   configCluster.Type,
			CapacityExhausted:      configCluster.CapacityExhausted,
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

// Create creates a new cluster configuration for later use
func (c *ClustersController) Create(ctx *app.CreateClustersContext) error {
	// check that the token belongs to a user
	if !auth.IsSpecificServiceAccount(ctx, auth.ToolChainOperator) {
		log.Error(ctx, nil, "unauthorized access to cluster info")
		return app.JSONErrorResponse(ctx, errors.NewUnauthorizedError("unauthorized access to cluster info"))
	}
	clustr := cluster.Cluster{
		Name:             ctx.Payload.Data.Name,
		Type:             ctx.Payload.Data.Type,
		AppDNS:           ctx.Payload.Data.AppDNS,
		SAToken:          ctx.Payload.Data.ServiceAccountToken,
		SAUsername:       ctx.Payload.Data.ServiceAccountUsername,
		TokenProviderID:  ctx.Payload.Data.TokenProviderID,
		AuthClientID:     ctx.Payload.Data.TokenProviderID,
		AuthClientSecret: ctx.Payload.Data.AuthClientSecret,
		AuthDefaultScope: ctx.Payload.Data.AuthClientDefaultScope,
	}
	if ctx.Payload.Data.ConsoleURL != nil {
		clustr.ConsoleURL = *ctx.Payload.Data.ConsoleURL
	}

	if ctx.Payload.Data.LoggingURL != nil {
		clustr.LoggingURL = *ctx.Payload.Data.LoggingURL
	}
	if ctx.Payload.Data.MetricsURL != nil {
		clustr.MetricsURL = *ctx.Payload.Data.MetricsURL
	}
	if ctx.Payload.Data.CapacityExhausted != nil {
		clustr.CapacityExhausted = *ctx.Payload.Data.CapacityExhausted
	}
	clusterSvc := c.app.ClusterService()
	err := clusterSvc.CreateOrSaveCluster(ctx, &clustr)
	if err != nil {
		return app.JSONErrorResponse(ctx, err)
	}
	return ctx.Created() // TODO: include a `Location` response if we want to "show" a single cluster at a time (eg: `/api/clusters/:clusterID`)
}
