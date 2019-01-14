package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

// createCluster represents a single cluster object
var createCluster = JSONSingle(
	"CreateCluster",
	"Holds the data to create a cluster",
	createClusterData,
	nil)

var createClusterData = a.Type("createClusterData", func() {
	a.Attribute("name", d.String, "Cluster name")
	a.Attribute("api-url", d.String, "API URL")
	a.Attribute("console-url", d.String, "Web console URL")
	a.Attribute("metrics-url", d.String, "Metrics URL")
	a.Attribute("logging-url", d.String, "Logging URL")
	a.Attribute("app-dns", d.String, "User application domain name in the cluster")
	a.Attribute("type", d.String, "Cluster type. Such as OSD, OSO, OCP, etc")
	a.Attribute("capacity-exhausted", d.Boolean, "Cluster is full if set to 'true'")
	a.Attribute("service-account-token", d.String, "Decrypted cluster wide token")
	a.Attribute("service-account-username", d.String, "Username of the cluster wide user")
	a.Attribute("token-provider-id", d.String, "Token provider ID")
	a.Attribute("auth-client-id", d.String, "OAuth client ID")
	a.Attribute("auth-client-secret", d.String, "OAuth client secret")
	a.Attribute("auth-client-default-scope", d.String, "OAuth client default scope")

	a.Required("name", "api-url", // other URLs are optional, they can be derived from the `api-url` if not explicitely provided
		"app-dns", "type",
		"service-account-token", "service-account-username", "auth-client-id", "auth-client-secret",
		"auth-client-default-scope")
})

// clusterList represents an array of cluster objects
var clusterList = JSONList(
	"Cluster",
	"Holds the response to a cluster list request",
	clusterData,
	nil,
	nil)

var clusterData = a.Type("ClusterData", func() {
	a.Attribute("name", d.String, "Cluster name")
	a.Attribute("api-url", d.String, "API URL")
	a.Attribute("console-url", d.String, "Web console URL")
	a.Attribute("metrics-url", d.String, "Metrics URL")
	a.Attribute("logging-url", d.String, "Logging URL")
	a.Attribute("app-dns", d.String, "User application domain name in the cluster")
	a.Attribute("type", d.String, "Cluster type. Such as OSD, OSO, OCP, etc")
	a.Attribute("capacity-exhausted", d.Boolean, "Cluster is full if set to 'true'")
	a.Required("name", "console-url", "metrics-url", "api-url", "logging-url", "app-dns", "type", "capacity-exhausted")
})

var fullClusterList = JSONList(
	"FullCluster",
	"Holds the response to a full cluster list request",
	fullClusterData,
	nil,
	nil)

var fullClusterData = a.Type("FullClusterData", func() {
	a.Attribute("name", d.String, "Cluster name")
	a.Attribute("api-url", d.String, "API URL")
	a.Attribute("console-url", d.String, "Web console URL")
	a.Attribute("metrics-url", d.String, "Metrics URL")
	a.Attribute("logging-url", d.String, "Logging URL")
	a.Attribute("app-dns", d.String, "User application domain name in the cluster")
	a.Attribute("type", d.String, "Cluster type. Such as OSD, OSO, OCP, etc")
	a.Attribute("capacity-exhausted", d.Boolean, "Cluster is full if set to 'true'")

	a.Attribute("service-account-token", d.String, "Decrypted cluster wide token")
	a.Attribute("service-account-username", d.String, "Username of the cluster wide user")
	a.Attribute("sa-token-encrypted", d.Boolean, "encrypted Service Account Token set to 'true'")
	a.Attribute("token-provider-id", d.String, "Token provider ID")
	a.Attribute("auth-client-id", d.String, "OAuth client ID")
	a.Attribute("auth-client-secret", d.String, "OAuth client secret")
	a.Attribute("auth-client-default-scope", d.String, "OAuth client default scope")

	a.Required("name", "console-url", "metrics-url", "api-url", "logging-url", "app-dns", "type", "capacity-exhausted",
		"service-account-token", "service-account-username", "token-provider-id", "auth-client-id", "auth-client-secret",
		"auth-client-default-scope")
})

// singleCluster represents a single cluster object
var showSingleCluster = JSONSingle(
	"Cluster",
	"Holds the response to a cluster request",
	clusterData,
	nil)

// singleCluster represents a single cluster object, including auth data
var showSingleFullCluster = JSONSingle(
	"FullCluster",
	"Holds the response to a cluster request",
	fullClusterData,
	nil)

var _ = a.Resource("clusters", func() {
	a.BasePath("/clusters")

	a.Action("list", func() {
		a.Security("jwt")
		a.Routing(
			a.GET("/"),
		)
		a.Description("Get all cluster configurations")
		a.Response(d.OK, clusterList)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})

	a.Action("listForAuthClient", func() {
		a.Security("jwt")
		a.Routing(
			a.GET("/auth"),
		)
		a.Description("Get all cluster configurations (including Auth information)")
		a.Response(d.OK, fullClusterList)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})

	a.Action("show", func() {
		a.Security("jwt")
		a.Routing(
			a.GET("/:clusterID"),
		)
		a.Params(func() {
			a.Param("clusterID", d.UUID, "the ID of the cluster to show")
			a.Required("clusterID")
		})
		a.Description("Get single cluster configuration")
		a.Response(d.OK, showSingleCluster)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("showForAuthClient", func() {
		a.Security("jwt")
		a.Routing(
			a.GET("/:clusterID/auth"),
		)
		a.Params(func() {
			a.Param("clusterID", d.UUID, "the ID of the cluster to show")
			a.Required("clusterID")
		})
		a.Description("Get single cluster configuration (including Auth information)")
		a.Response(d.OK, showSingleFullCluster)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("create", func() {
		a.Security("jwt")
		a.Routing(
			a.POST("/"),
		)
		a.Payload(createCluster)
		a.Description("Add a cluster configuration")
		a.Response(d.Created)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("delete", func() {
		a.Security("jwt")
		a.Routing(
			a.DELETE("/:clusterID"),
		)
		a.Params(func() {
			a.Param("clusterID", d.UUID, "the ID of the cluster to delete")
			a.Required("clusterID")
		})
		a.Description("Delete a cluster configuration")
		a.Response(d.NoContent)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("linkIdentityToCluster", func() {
		a.Security("jwt")
		a.Routing(
			a.POST("/identities"),
		)
		a.Payload(linkIdentityToClusterData)
		a.Description("create a identitycluster using a service account")
		a.Response(d.NoContent)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.BadRequest, JSONAPIErrors)
	})

	a.Action("removeIdentityToClusterLink", func() {
		a.Security("jwt")
		a.Routing(
			a.DELETE("/identities"),
		)
		a.Payload(unLinkIdentityToClusterdata)
		a.Description("Remove a identity cluster relation using a service account")
		a.Response(d.NoContent)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.BadRequest, JSONAPIErrors)
	})
})

// linkIdentityToClusterData represents the data of an identified IdentityCluster object to create
var linkIdentityToClusterData = a.Type("linkIdentityToClusterData", func() {
	a.Attribute("identity-id", d.String, "The id of corresponding Identity")
	a.Attribute("cluster-url", d.String, "Cluster URL")
	a.Attribute("ignore-if-already-exists", d.Boolean, "Ignore creation error if this identity already exists. By default 'True'")

	a.Required("cluster-url", "identity-id")
})

// unLinkIdentityToClusterdata represents data of an identified IdentityCluster object to remove
var unLinkIdentityToClusterdata = a.Type("unLinkIdentityToClusterdata", func() {
	a.Attribute("identity-id", d.String, "The id of corresponding Identity")
	a.Attribute("cluster-url", d.String, "Cluster URL")

	a.Required("cluster-url", "identity-id")
})
