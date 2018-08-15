package controller

import (
	"github.com/fabric8-services/fabric8-cluster/app"
	"github.com/fabric8-services/fabric8-cluster/configuration"
	"github.com/fabric8-services/fabric8-cluster/jsonapi"
	"github.com/fabric8-services/fabric8-cluster/rest"
	"github.com/fabric8-services/fabric8-cluster/token"
	"github.com/fabric8-services/fabric8-common/errors"
	"github.com/fabric8-services/fabric8-common/log"

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

// Show runs the list of available OSO clusters.
func (c *ClustersController) Show(ctx *app.ShowClustersContext) error {
	if !token.IsSpecificServiceAccount(ctx, token.OsoProxy, token.Tenant, token.JenkinsIdler, token.JenkinsProxy) {
		log.Error(ctx, nil, "unauthorized access to cluster info")
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError("unauthorized access to cluster info"))
	}
	var data []*app.ClusterData
	for _, clusterConfig := range c.config.GetOSOClusters() {
		cluster := &app.ClusterData{
			Name:              clusterConfig.Name,
			APIURL:            rest.AddTrailingSlashToURL(clusterConfig.APIURL),
			ConsoleURL:        rest.AddTrailingSlashToURL(clusterConfig.ConsoleURL),
			MetricsURL:        rest.AddTrailingSlashToURL(clusterConfig.MetricsURL),
			LoggingURL:        rest.AddTrailingSlashToURL(clusterConfig.LoggingURL),
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
