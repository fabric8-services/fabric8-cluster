package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/fabric8-services/fabric8-common/errors"
	"github.com/fabric8-services/fabric8-common/gormsupport"
	"github.com/fabric8-services/fabric8-common/log"

	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// IdentityClusterRepository represents the storage interface.
type IdentityClusterRepository interface {
	Load(ctx context.Context, identityID, clusterID uuid.UUID) (*IdentityCluster, error)
	ListClustersForIdentity(ctx context.Context, identityID uuid.UUID) ([]Cluster, error)
	Create(ctx context.Context, u *IdentityCluster) error
	Delete(ctx context.Context, identityID uuid.UUID, clusterURL string) error
}

// IdentityCluster a type that associates an Identity to a Cluster
type IdentityCluster struct {
	gormsupport.Lifecycle
	// The associated Identity ID
	IdentityID uuid.UUID `sql:"type:uuid" gorm:"primary_key;column:identity_id"`
	// The associated cluster
	Cluster Cluster `gorm:"ForeignKey:ClusterID;association_foreignkey:ClusterID"`
	// The foreign key value for ClusterID
	ClusterID uuid.UUID ` sql:"type:uuid" gorm:"primary_key;column:cluster_id"`
}

// GormIdentityClusterRepository is the implementation of the storage interface for IdentityCluster.
type GormIdentityClusterRepository struct {
	db *gorm.DB
}

// NewIdentityClusterRepository creates a new storage type.
func NewIdentityClusterRepository(db *gorm.DB) IdentityClusterRepository {
	return &GormIdentityClusterRepository{db: db}
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (m *GormIdentityClusterRepository) TableName() string {
	return "identity_cluster"
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (m IdentityCluster) TableName() string {
	return "identity_cluster"
}

// CRUD Functions

// Load returns a single Identity Cluster as a Database Model
func (m *GormIdentityClusterRepository) Load(ctx context.Context, identityID, clusterID uuid.UUID) (*IdentityCluster, error) {
	defer goa.MeasureSince([]string{"goa", "db", "identity_cluster", "load"}, time.Now())
	var native IdentityCluster
	err := m.db.Table(m.TableName()).Preload("Cluster").Where("identity_id = ? and cluster_id = ?", identityID, clusterID).Find(&native).Error
	if err == gorm.ErrRecordNotFound {
		return nil, errors.NewNotFoundErrorFromString(fmt.Sprintf("identity_cluster with identity ID %s and cluster ID %s not found", identityID, clusterID))
	}
	return &native, errs.WithStack(err)
}

// ListClustersForIdentity returns the list of all cluster for the identity
func (m *GormIdentityClusterRepository) ListClustersForIdentity(ctx context.Context, identityID uuid.UUID) ([]Cluster, error) {
	defer goa.MeasureSince([]string{"goa", "db", "identity_cluster", "list_clusters_for_identity"}, time.Now())
	var rows []IdentityCluster
	err := m.db.Table(m.TableName()).Preload("Cluster").Where("identity_id = ?", identityID).Find(&rows).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errs.WithStack(err)
	}
	clusters := make([]Cluster, 0, len(rows))
	for _, idCluster := range rows {
		clusters = append(clusters, idCluster.Cluster)
	}
	return clusters, nil
}

// Create creates a new record.
func (m *GormIdentityClusterRepository) Create(ctx context.Context, c *IdentityCluster) error {
	defer goa.MeasureSince([]string{"goa", "db", "identity_cluster", "create"}, time.Now())
	err := m.db.Create(c).Error
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"cluster_id":  c.ClusterID.String(),
			"identity_id": c.IdentityID.String(),
			"err":         err,
		}, "unable to create the identity cluster")
		return errs.WithStack(err)
	}
	log.Debug(ctx, map[string]interface{}{
		"cluster_id":  c.ClusterID.String(),
		"identity_id": c.IdentityID.String(),
	}, "Identity cluster created!")
	return nil
}

// Delete removes the identity/cluster relationship identified by the given `identityID` and `clusterURL`
func (m *GormIdentityClusterRepository) Delete(ctx context.Context, identityID uuid.UUID, clusterURL string) error {
	defer goa.MeasureSince([]string{"goa", "db", "identity_cluster", "delete"}, time.Now())

	result := m.db.Exec(fmt.Sprintf(`delete from %[1]s where identity_id = ? 
		and cluster_id = (select cluster_id from %[2]s where url = ?)`,
		IdentityCluster{}.TableName(), Cluster{}.TableName()), identityID.String(), clusterURL)

	if result.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"cluster_url": clusterURL,
			"identity_id": identityID.String(),
			"err":         result.Error,
		}, "unable to delete the identity cluster")
		return errs.WithStack(result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundErrorFromString(fmt.Sprintf("nothing to delete: identity cluster not found (identityID:\"%s\", clusterURL:\"%s\")", identityID.String(), clusterURL))
	}

	log.Debug(ctx, map[string]interface{}{
		"cluster_url": clusterURL,
		"identity_id": identityID.String(),
	}, "Identity cluster deleted!")

	return nil
}
