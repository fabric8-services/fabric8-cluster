package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var _ = a.Resource("identityClusters", func() {
	a.BasePath("/identityclusters")

	a.Action("create", func() {
		a.Security("jwt")
		a.Routing(
			a.POST(""),
		)
		a.Payload(createIdentityClusterData)
		a.Description("create a identitycluster using a service account")
		a.Response(d.NoContent)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.BadRequest, JSONAPIErrors)
	})
})

// createIdentityClusterData represents the data of an identified IdentityCluster object to create
var createIdentityClusterData = a.Type("createIdentityClusterData", func() {
	a.Attribute("type", d.String, "type of the identity cluster")
	a.Attribute("attributes", createIdentityClusterAttributes, "Attributes of the identity cluster")
	a.Attribute("links", genericLinks)
	a.Required("type", "attributes")
})


var createIdentityClusterAttributes = a.Type("createIdentityClusterAttributes", func() {
	a.Attribute("cluster-url", d.String, "Cluster URL")
	a.Attribute("identity-id", d.String, "The id of corresponding Identity")

	a.Required("cluster-url", "identity-id")
})
