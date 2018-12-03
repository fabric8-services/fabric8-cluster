package controller

import (
	"github.com/fabric8-services/fabric8-cluster/app"
	"github.com/fabric8-services/fabric8-cluster/application"
	"github.com/fabric8-services/fabric8-common/auth"
	"github.com/fabric8-services/fabric8-common/errors"
	"github.com/fabric8-services/fabric8-common/log"
	"github.com/goadesign/goa"
)

// IdentityClustersController implements the identityClusters resource.
type IdentityClustersController struct {
	*goa.Controller
	app application.Application
}

// NewIdentityClustersController creates a identityClusters controller.
func NewIdentityClustersController(service *goa.Service, app application.Application) *IdentityClustersController {
	return &IdentityClustersController{Controller: service.NewController("IdentityClustersController"), app: app}
}

// Create populates Identity Cluster relationship
func (c *IdentityClustersController) Create(ctx *app.CreateIdentityClustersContext) error {
	if !auth.IsSpecificServiceAccount(ctx, auth.Auth) {
		log.Error(ctx, nil, "the account is not authorized to create identity cluster relationship")
		return app.JSONErrorResponse(ctx, errors.NewUnauthorizedError("account not authorized to create identity cluster relationship"))
	}

	if err := c.app.ClusterService().LinkIdentityToCluster(ctx, ctx.Payload.Attributes.IdentityID, ctx.Payload.Attributes.ClusterURL); err != nil {
		return app.JSONErrorResponse(ctx, err)
	}

	return ctx.NoContent()
}
