package service

import (
	"context"

	"github.com/fabric8-services/fabric8-cluster/cluster/repository"
	uuid "github.com/satori/go.uuid"
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

//Services creates instances of service layer objects
type Services interface {
	ClusterService() ClusterService
}

// ClusterService the interface for the cluster service
type ClusterService interface {
	InitializeClusterWatcher() (func() error, error)
	CreateOrSaveClusterFromConfig(ctx context.Context) error
	CreateOrSaveCluster(ctx context.Context, clustr *repository.Cluster) error
	Load(ctx context.Context, clusterID uuid.UUID) (*repository.Cluster, error)
	LoadForAuth(ctx context.Context, clusterID uuid.UUID) (*repository.Cluster, error)
	FindByURL(ctx context.Context, clusterURL string) (*repository.Cluster, error)
	FindByURLForAuth(ctx context.Context, clusterURL string) (*repository.Cluster, error)
	List(ctx context.Context, clusterType *string) ([]repository.Cluster, error)
	ListForAuth(ctx context.Context, clusterType *string) ([]repository.Cluster, error)
	Delete(ctx context.Context, clusterID uuid.UUID) error
	LinkIdentityToCluster(ctx context.Context, identityID uuid.UUID, clusterURL string, ignoreError bool) error
	RemoveIdentityToClusterLink(ctx context.Context, identityID uuid.UUID, clusterURL string) error
}
