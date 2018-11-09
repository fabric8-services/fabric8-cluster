package repository

import (
	"context"
	"time"

	"github.com/fabric8-services/fabric8-cluster/application/repository/base"
	"github.com/fabric8-services/fabric8-common/errors"
	"github.com/fabric8-services/fabric8-common/gormsupport"
	"github.com/fabric8-services/fabric8-common/log"

	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

type Cluster struct {
	gormsupport.HardDeleteLifecycle

	// This is the primary key value
	ClusterID uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key;column:cluster_id"`
	// The name of the cluster
	Name string
	// API URL of the cluster
	URL string
	// Console URL of the cluster
	ConsoleURL string
	// Metrics URL of the cluster
	MetricsURL string
	// Logging URL of the cluster
	LoggingURL string
	// Application host name used by the cluster
	AppDNS string
	// Encrypted Service Account token
	SaToken string
	// Service Account name
	SaUsername string
	// Token Provider ID
	TokenProviderID string
	// OAuthClient ID used to link users account
	AuthClientID string
	// OAuthClient secret used to link users account
	AuthClientSecret string
	// OAuthClient default scope used to link users account
	AuthDefaultScope string
	// Cluster type. Such as OSD, OSO, OCP, etc
	Type string
}

// GormClusterRepository is the implementation of the storage interface for Cluster.
type GormClusterRepository struct {
	db *gorm.DB
}

// NewClusterRepository creates a new storage type.
func NewClusterRepository(db *gorm.DB) ClusterRepository {
	return &GormClusterRepository{db: db}
}

// ClusterRepository represents the storage interface.
type ClusterRepository interface {
	base.Exister
	Load(ctx context.Context, ID uuid.UUID) (*Cluster, error)
	Create(ctx context.Context, u *Cluster) error
	Save(ctx context.Context, u *Cluster) error
	Delete(ctx context.Context, ID uuid.UUID) error
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (m *GormClusterRepository) TableName() string {
	return "cluster"
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (m Cluster) TableName() string {
	return "cluster"
}

// CheckExists returns nil if the given ID exists otherwise returns an error
func (m *GormClusterRepository) CheckExists(ctx context.Context, id string) error {
	defer goa.MeasureSince([]string{"goa", "db", "cluster", "exists"}, time.Now())
	return base.CheckHardDeletableExists(ctx, m.db, m.TableName(), "cluster_id", id)
}

// CRUD Functions

// Load returns a single Cluster as a Database Model
func (m *GormClusterRepository) Load(ctx context.Context, id uuid.UUID) (*Cluster, error) {
	defer goa.MeasureSince([]string{"goa", "db", "cluster", "load"}, time.Now())
	var native Cluster
	err := m.db.Table(m.TableName()).Where("cluster_id = ?", id).Find(&native).Error
	if err == gorm.ErrRecordNotFound {
		return nil, errors.NewNotFoundError("cluster", id.String())
	}
	return &native, errs.WithStack(err)
}

// Create creates a new record.
func (m *GormClusterRepository) Create(ctx context.Context, c *Cluster) error {
	defer goa.MeasureSince([]string{"goa", "db", "cluster", "create"}, time.Now())
	if c.ClusterID == uuid.Nil {
		c.ClusterID = uuid.NewV4()
	}
	err := m.db.Create(c).Error
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"cluster_id": c.ClusterID.String(),
			"err":        err,
		}, "unable to create the cluster")
		return errs.WithStack(err)
	}
	log.Debug(ctx, map[string]interface{}{
		"cluster_id": c.ClusterID.String(),
	}, "Cluster created!")
	return nil
}

// Save modifies a single record
func (m *GormClusterRepository) Save(ctx context.Context, c *Cluster) error {
	defer goa.MeasureSince([]string{"goa", "db", "cluster", "save"}, time.Now())

	obj, err := m.Load(ctx, c.ClusterID)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"cluster_id": c.ClusterID.String(),
			"err":        err,
		}, "unable to update cluster")
		return errs.WithStack(err)
	}
	err = m.db.Model(obj).Updates(c).Error
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"cluster_id": c.ClusterID.String(),
			"err":        err,
		}, "unable to update cluster")
		return errs.WithStack(err)
	}

	log.Debug(ctx, map[string]interface{}{
		"cluster_id": c.ClusterID.String(),
	}, "Cluster saved!")
	return nil
}

// Delete removes a single record. This is a hard delete!
func (m *GormClusterRepository) Delete(ctx context.Context, id uuid.UUID) error {
	defer goa.MeasureSince([]string{"goa", "db", "cluster", "delete"}, time.Now())

	obj := Cluster{ClusterID: id}

	result := m.db.Delete(&obj)

	if result.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"cluster_id": id.String(),
			"err":        result.Error,
		}, "unable to delete the cluster")
		return errs.WithStack(result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("cluster", id.String())
	}

	log.Debug(ctx, map[string]interface{}{
		"cluster_id": id.String(),
	}, "Cluster deleted!")

	return nil
}
