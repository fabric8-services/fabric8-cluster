package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

// clusterList represents an array of cluster objects
var clusterList = JSONList(
	"Cluster",
	"Holds the response to a cluster list request",
	clusterData,
	nil,
	nil)

var fullClusterList = JSONList(
	"FullCluster",
	"Holds the response to a full cluster list request",
	fullClusterData,
	nil,
	nil)

var clusterData = a.Type("ClusterData", func() {
	a.Attribute("name", d.String, "Cluster name")
	a.Attribute("api-url", d.String, "API URL")
	a.Attribute("console-url", d.String, "Web console URL")
	a.Attribute("metrics-url", d.String, "Metrics URL")
	a.Attribute("logging-url", d.String, "Logging URL")
	a.Attribute("app-dns", d.String, "User application domain name in the cluster")
	a.Attribute("capacity-exhausted", d.Boolean, "Cluster is full if set to 'true'")
	a.Required("name", "console-url", "metrics-url", "api-url", "logging-url", "app-dns", "capacity-exhausted")
})

var fullClusterData = a.Type("FullClusterData", func() {
	a.Attribute("name", d.String, "Cluster name")
	a.Attribute("api-url", d.String, "API URL")
	a.Attribute("console-url", d.String, "Web console URL")
	a.Attribute("metrics-url", d.String, "Metrics URL")
	a.Attribute("logging-url", d.String, "Logging URL")
	a.Attribute("app-dns", d.String, "User application domain name in the cluster")
	a.Attribute("capacity-exhausted", d.Boolean, "Cluster is full if set to 'true'")

	a.Attribute("service-account-token", d.String, "Decrypted cluster wide token")
	a.Attribute("service-account-username", d.String, "Username of the cluster wide user")
	a.Attribute("token-provider-id", d.String, "Token provider ID")
	a.Attribute("auth-client-id", d.String, "OAuth client ID")
	a.Attribute("auth-client-secret", d.String, "OAuth client secret")
	a.Attribute("auth-client-default-scope", d.String, "OAuth client default scope")

	a.Required("name", "console-url", "metrics-url", "api-url", "logging-url", "app-dns", "capacity-exhausted",
		"service-account-token", "service-account-username", "token-provider-id", "auth-client-id", "auth-client-secret",
		"auth-client-default-scope")
})

var _ = a.Resource("clusters", func() {
	a.BasePath("/clusters")

	a.Action("show", func() {
		a.Security("jwt")
		a.Routing(
			a.GET("/"),
		)
		a.Description("Get clusters configuration")
		a.Response(d.OK, clusterList)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})

	a.Action("showAuthClient", func() {
		a.Security("jwt")
		a.Routing(
			a.GET("/auth"),
		)
		a.Description("Get full cluster configuration including Auth information")
		a.Response(d.OK, fullClusterList)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
})
