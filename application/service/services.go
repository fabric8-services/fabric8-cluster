package service

import (
	"context"

	"github.com/fabric8-services/fabric8-cluster/cluster/repository"
)

/*
Steps for adding a new Service:
1. Add a new service interface to application/service/services.go
2. Create an implementation of the service interface
3. Add a new method to service.Services interface in application/service/services.go for accessing the service interface
   defined in step 1
4. Add a new method to application/service/factory/service_factory.go which implements the service access method
   from step #3 and uses the service constructor from step 2
5. Add a new method to gormapplication/application.go which implements the service access method from step #3
   and use the factory method from the step #4
*/

// ClusterService the interface for the cluster service
type ClusterService interface {
	CreateOrSaveClusterFromConfig(ctx context.Context) error
	CreateOrSaveCluster(ctx context.Context, clustr *repository.Cluster) error
	InitializeClusterWatcher() (func() error, error)
}

//Services creates instances of service layer objects
type Services interface {
	ClusterService() ClusterService
}
