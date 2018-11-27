package controller

import (
	"github.com/fabric8-services/fabric8-cluster/app"
	"github.com/fabric8-services/fabric8-common/auth"
	"github.com/goadesign/goa"

	"github.com/fabric8-services/fabric8-cluster/application"
	"github.com/fabric8-services/fabric8-common/log"
)

// UserController implements the user resource.
type UserController struct {
	*goa.Controller
	app application.Application
}

// NewUserController creates a user controller.
func NewUserController(service *goa.Service, app application.Application) *UserController {
	return &UserController{Controller: service.NewController("UserController"), app: app}
}

// Clusters runs the clusters action.
func (c *UserController) Clusters(ctx *app.ClustersUserContext) error {
	identityID, err := auth.LocateIdentity(ctx)
	if err != nil {
		return app.JSONErrorResponse(ctx, err)
	}

	clusters, err := c.app.IdentityClusters().ListClustersForIdentity(ctx, identityID)

	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"identity_id": identityID,
			"err":         err,
		}, "failed to list clusters for identity %s", identityID)
		return app.JSONErrorResponse(ctx, err)
	}
	data := make([]*app.ClusterData, 0)
	for _, cluster := range clusters {
		clusterData := &app.ClusterData{
			Name:              cluster.Name,
			APIURL:            cluster.URL,
			ConsoleURL:        cluster.ConsoleURL,
			MetricsURL:        cluster.MetricsURL,
			LoggingURL:        cluster.LoggingURL,
			AppDNS:            cluster.AppDNS,
			Type:              cluster.Type,
			CapacityExhausted: cluster.CapacityExhausted,
		}
		data = append(data, clusterData)
	}

	return ctx.OK(&app.ClusterList{Data: data})
}
