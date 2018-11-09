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
	"github.com/satori/go.uuid"
)

type IdentityCluster struct {
	gormsupport.Lifecycle
	// The associated Identity ID
	IdentityID uuid.UUID `sql:"type:uuid" gorm:"primary_key;column:identity_id"`
	// The associated cluster
	Cluster Cluster `gorm:"ForeignKey:ClusterID"`
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

// IdentityClusterRepository represents the storage interface.
type IdentityClusterRepository interface {
	LoadByIdentity(ctx context.Context, identityID uuid.UUID) (*IdentityCluster, error)
	Create(ctx context.Context, u *IdentityCluster) error
	Delete(ctx context.Context, identityID, clusterID uuid.UUID) error
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

// Load returns a single Cluster as a Database Model
func (m *GormIdentityClusterRepository) LoadByIdentity(ctx context.Context, identityID uuid.UUID) (*IdentityCluster, error) {
	defer goa.MeasureSince([]string{"goa", "db", "identity_cluster", "load"}, time.Now())
	var native IdentityCluster
	err := m.db.Table(m.TableName()).Preload("ClusterID").Where("identity_id = ?", identityID).Find(&native).Error
	if err == gorm.ErrRecordNotFound {
		return nil, errors.NewNotFoundError("identity_cluster", identityID.String())
	}
	return &native, errs.WithStack(err)
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
		}, "unable to create the cluster")
		return errs.WithStack(err)
	}
	log.Debug(ctx, map[string]interface{}{
		"cluster_id":  c.ClusterID.String(),
		"identity_id": c.IdentityID.String(),
	}, "Identity Cluster created!")
	return nil
}

// Delete removes a single record.
func (m *GormIdentityClusterRepository) Delete(ctx context.Context, identityID, clusterID uuid.UUID) error {
	defer goa.MeasureSince([]string{"goa", "db", "identity_cluster", "delete"}, time.Now())

	c := IdentityCluster{IdentityID: identityID, ClusterID: clusterID}

	result := m.db.Delete(&c)

	if result.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"cluster_id":  c.ClusterID.String(),
			"identity_id": c.IdentityID.String(),
			"err":         result.Error,
		}, "unable to delete the cluster")
		return errs.WithStack(result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundErrorFromString(fmt.Sprintf("nothing to delete: identity cluster not found (clusterID:\"%s\", identityID:\"%s\")", clusterID, identityID.String()))
	}

	log.Debug(ctx, map[string]interface{}{
		"cluster_id":  c.ClusterID.String(),
		"identity_id": c.IdentityID.String(),
	}, "Cluster deleted!")

	return nil
}
