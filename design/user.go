package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var _ = a.Resource("user", func() {
	a.BasePath("/user")

	a.Action("clusters", func() {
		a.Security("jwt")
		a.Routing(
			a.GET("/clusters"),
		)
		a.Description("Get clusters available to user")
		a.Response(d.OK, clusterList)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
})
