package repository

import (
	"github.com/fabric8-services/fabric8-cluster/cluster/repository"
)

//Repositories stands for a particular implementation of the business logic of our application
type Repositories interface {
	Clusters() repository.ClusterRepository
	IdentityClusters() repository.IdentityClusterRepository
}
