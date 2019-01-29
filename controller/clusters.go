package controller

import (
	"github.com/fabric8-services/fabric8-cluster/app"
	"github.com/fabric8-services/fabric8-cluster/application"
	cluster "github.com/fabric8-services/fabric8-cluster/cluster/repository"
	"github.com/fabric8-services/fabric8-common/errors"
	"github.com/fabric8-services/fabric8-common/httpsupport"
	"github.com/fabric8-services/fabric8-common/log"

	"fmt"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// ClustersController implements the clusters resource.
type ClustersController struct {
	*goa.Controller
	app application.Application
}

// NewClustersController creates a clusters controller.
func NewClustersController(service *goa.Service, app application.Application) *ClustersController {
	return &ClustersController{
		Controller: service.NewController("ClustersController"),
		app:        app,
	}
}

// List returns the list of available clusters.
func (c *ClustersController) List(ctx *app.ListClustersContext) error {
	clusters, err := c.app.ClusterService().List(ctx)
	if err != nil {
		return app.JSONErrorResponse(ctx, err)
	}
	var data []*app.ClusterData
	for _, clustr := range clusters {
		data = append(data, convertToClusterData(clustr))
	}
	return ctx.OK(&app.ClusterList{
		Data: data,
	})
}

// ListForAuthClient returns the list of available clusters with full configuration including Auth client data.
// To be used by Auth service only
func (c *ClustersController) ListForAuthClient(ctx *app.ListForAuthClientClustersContext) error {
	clusters, err := c.app.ClusterService().ListForAuth(ctx)
	if err != nil {
		return app.JSONErrorResponse(ctx, err)
	}
	var data []*app.FullClusterData
	for _, clustr := range clusters {
		data = append(data, convertToFullClusterData(clustr))
	}
	return ctx.OK(&app.FullClusterList{
		Data: data,
	})
}

// Show returns the single clusters.
func (c *ClustersController) Show(ctx *app.ShowClustersContext) error {
	// authorization is checked at the service level for more consistency accross the codebase.
	clustr, err := c.app.ClusterService().Load(ctx, ctx.ClusterID)
	if err != nil {
		return app.JSONErrorResponse(ctx, err)
	}
	return ctx.OK(&app.ClusterSingle{
		Data: convertToClusterData(*clustr),
	})
}

// ShowForAuthClient returns the cluster with full configuration including Auth client data.
// To be used by Auth service only
func (c *ClustersController) ShowForAuthClient(ctx *app.ShowForAuthClientClustersContext) error {
	// authorization is checked at the service level for more consistency accross the codebase.
	clustr, err := c.app.ClusterService().LoadForAuth(ctx, ctx.ClusterID)
	if err != nil {
		return app.JSONErrorResponse(ctx, err)
	}
	return ctx.OK(&app.FullClusterSingle{
		Data: convertToFullClusterData(*clustr),
	})
}

// Create creates a new cluster configuration for later use
func (c *ClustersController) Create(ctx *app.CreateClustersContext) error {
	clustr := cluster.Cluster{
		Name:             ctx.Payload.Data.Name,
		Type:             ctx.Payload.Data.Type,
		URL:              ctx.Payload.Data.APIURL,
		AppDNS:           ctx.Payload.Data.AppDNS,
		SAToken:          ctx.Payload.Data.ServiceAccountToken,
		SAUsername:       ctx.Payload.Data.ServiceAccountUsername,
		AuthClientID:     ctx.Payload.Data.AuthClientID,
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
	if ctx.Payload.Data.TokenProviderID != nil {
		clustr.TokenProviderID = *ctx.Payload.Data.TokenProviderID
	}
	clusterSvc := c.app.ClusterService()
	err := clusterSvc.CreateOrSaveCluster(ctx, &clustr)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"error": err,
		}, "error while creating new cluster configuration")
		return app.JSONErrorResponse(ctx, err)
	}
	ctx.ResponseData.Header().Set("Location", app.ClustersHref(clustr.ClusterID.String()))
	return ctx.Created()
}

// Delete deletes the cluster identified by the `clusterID` param
func (c *ClustersController) Delete(ctx *app.DeleteClustersContext) error {
	err := c.app.ClusterService().Delete(ctx, ctx.ClusterID)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"error": err,
		}, "error while deleting a cluster configuration")
		return app.JSONErrorResponse(ctx, err)
	}
	return ctx.NoContent()
}

// LinkIdentityToCluster populates Identity Cluster relationship
func (c *ClustersController) LinkIdentityToCluster(ctx *app.LinkIdentityToClusterClustersContext) error {
	identityID, err := uuid.FromString(ctx.Payload.IdentityID)
	if err != nil {
		return app.JSONErrorResponse(ctx, errors.NewBadParameterErrorFromString(fmt.Sprintf("identity-id %s is not a valid UUID", identityID)))
	}

	// ignoreIfAlreadyExisted by default true
	ignore := true
	if ignoreIfExists := ctx.Payload.IgnoreIfAlreadyExists; ignoreIfExists != nil {
		ignore = *ignoreIfExists
	}
	if err := c.app.ClusterService().LinkIdentityToCluster(ctx, identityID, ctx.Payload.ClusterURL, ignore); err != nil {
		log.Error(ctx, map[string]interface{}{
			"error": err,
		}, "error while linking identity-id %s to cluster with url '%s'", identityID, ctx.Payload.ClusterURL)
		return app.JSONErrorResponse(ctx, err)
	}
	return ctx.NoContent()
}

// RemoveIdentityToClusterLink removes Identity Cluster relationship
func (c *ClustersController) RemoveIdentityToClusterLink(ctx *app.RemoveIdentityToClusterLinkClustersContext) error {
	identityID, err := uuid.FromString(ctx.Payload.IdentityID)
	if err != nil {
		return app.JSONErrorResponse(ctx, errors.NewBadParameterErrorFromString(fmt.Sprintf("identity-id %s is not a valid UUID", identityID)))
	}
	if err := c.app.ClusterService().RemoveIdentityToClusterLink(ctx, identityID, ctx.Payload.ClusterURL); err != nil {
		log.Error(ctx, map[string]interface{}{
			"error": err,
		}, "error while removing link of identity-id %s to cluster with url '%s'", identityID, ctx.Payload.ClusterURL)
		return app.JSONErrorResponse(ctx, err)
	}

	return ctx.NoContent()
}

func convertToClusterData(clustr cluster.Cluster) *app.ClusterData {
	return &app.ClusterData{
		Name:              clustr.Name,
		APIURL:            httpsupport.AddTrailingSlashToURL(clustr.URL),
		ConsoleURL:        httpsupport.AddTrailingSlashToURL(clustr.ConsoleURL),
		MetricsURL:        httpsupport.AddTrailingSlashToURL(clustr.MetricsURL),
		LoggingURL:        httpsupport.AddTrailingSlashToURL(clustr.LoggingURL),
		AppDNS:            clustr.AppDNS,
		Type:              clustr.Type,
		CapacityExhausted: clustr.CapacityExhausted,
	}
}

func convertToFullClusterData(clustr cluster.Cluster) *app.FullClusterData {
	encrypted := clustr.SATokenEncrypted
	return &app.FullClusterData{
		Name:                   clustr.Name,
		APIURL:                 httpsupport.AddTrailingSlashToURL(clustr.URL),
		ConsoleURL:             httpsupport.AddTrailingSlashToURL(clustr.ConsoleURL),
		MetricsURL:             httpsupport.AddTrailingSlashToURL(clustr.MetricsURL),
		LoggingURL:             httpsupport.AddTrailingSlashToURL(clustr.LoggingURL),
		AppDNS:                 clustr.AppDNS,
		Type:                   clustr.Type,
		CapacityExhausted:      clustr.CapacityExhausted,
		AuthClientDefaultScope: clustr.AuthDefaultScope,
		AuthClientID:           clustr.AuthClientID,
		AuthClientSecret:       clustr.AuthClientSecret,
		SaTokenEncrypted:       &encrypted,
		ServiceAccountToken:    clustr.SAToken,
		ServiceAccountUsername: clustr.SAUsername,
		TokenProviderID:        clustr.TokenProviderID,
	}
}
