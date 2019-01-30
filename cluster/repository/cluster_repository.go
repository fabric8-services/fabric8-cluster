package repository

import (
	"context"
	"strings"
	"time"

	"github.com/fabric8-services/fabric8-cluster/cluster"
	"github.com/fabric8-services/fabric8-common/httpsupport"

	"github.com/fabric8-services/fabric8-cluster/application/repository/base"
	"github.com/fabric8-services/fabric8-common/errors"
	"github.com/fabric8-services/fabric8-common/gormsupport"
	"github.com/fabric8-services/fabric8-common/log"

	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"fmt"
)

// Cluster the struct that holds the cluster info
type Cluster struct {
	gormsupport.LifecycleHardDelete
	// This is the primary key value
	ClusterID uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key;column:cluster_id"`
	// The name of the cluster
	Name string `mapstructure:"name"`
	// API URL of the cluster
	URL string `sql:"unique_index" mapstructure:"api-url"`
	// Console URL of the cluster
	ConsoleURL string `mapstructure:"console-url" optional:"true"` // Optional in config file
	// Metrics URL of the cluster
	MetricsURL string `mapstructure:"metrics-url" optional:"true"` // Optional in config file
	// Logging URL of the cluster
	LoggingURL string `mapstructure:"logging-url" optional:"true"` // Optional in config file
	// Application host name used by the cluster
	AppDNS string `mapstructure:"app-dns"`
	// Service Account token (encrypted or not, depending on the state of the sibling SATokenEncrypted field)
	SAToken string `mapstructure:"service-account-token"`
	// Service Account username
	SAUsername string `mapstructure:"service-account-username"`
	// SA Token encrypted
	SATokenEncrypted bool `mapstructure:"service-account-token-encrypted" optional:"true" default:"true"` // Optional in config file
	// Token Provider ID
	TokenProviderID string `mapstructure:"token-provider-id"`
	// OAuthClient ID used to link users account
	AuthClientID string `mapstructure:"auth-client-id"`
	// OAuthClient secret used to link users account
	AuthClientSecret string `mapstructure:"auth-client-secret"`
	// OAuthClient default scope used to link users account
	AuthDefaultScope string `mapstructure:"auth-client-default-scope"`
	// Cluster type. Such as OSD, OSO, OCP, etc
	Type string `mapstructure:"type" optional:"true" default:"OSO"` // Optional in config file
	// cluster capacity exhausted by default false
	CapacityExhausted bool `mapstructure:"capacity-exhausted" optional:"true"` // Optional in config file
}

// Normalize fills the `console`, `metrics` and `logging` URL if there were missing,
// and appends a trailing slash if needed.
func (c *Cluster) Normalize() error {
	// ensure that cluster URL ends with a slash
	c.URL = httpsupport.AddTrailingSlashToURL(c.URL)

	var err error
	// fill missing values and ensures that all URLs have a trailing slash
	// console URL
	if strings.TrimSpace(c.ConsoleURL) == "" {
		c.ConsoleURL, err = ConvertAPIURL(c.URL, "console", "console")
		if err != nil {
			return err
		}
	}
	c.ConsoleURL = httpsupport.AddTrailingSlashToURL(c.ConsoleURL)
	// metrics URL
	if strings.TrimSpace(c.MetricsURL) == "" {
		c.MetricsURL, err = ConvertAPIURL(c.URL, "metrics", "")
		if err != nil {
			return err
		}
	}
	c.MetricsURL = httpsupport.AddTrailingSlashToURL(c.MetricsURL)
	// logging URL
	if strings.TrimSpace(c.LoggingURL) == "" {
		// This is not a typo; the logging host is the same as the console host in current k8s
		c.LoggingURL, err = ConvertAPIURL(c.URL, "console", "console")
		if err != nil {
			return err
		}
	}
	c.LoggingURL = httpsupport.AddTrailingSlashToURL(c.LoggingURL)
	// ensure that AppDNS URL ends with a slash
	c.AppDNS = httpsupport.AddTrailingSlashToURL(c.AppDNS)
	// apply default type of cluster
	if c.Type == "" {
		c.Type = cluster.OSO
	}
	return nil
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
	CreateOrSave(ctx context.Context, u *Cluster) error
	Delete(ctx context.Context, ID uuid.UUID) error
	Query(funcs ...func(*gorm.DB) *gorm.DB) ([]Cluster, error)
	FindByURL(ctx context.Context, url string) (*Cluster, error)
	List(ctx context.Context) ([]Cluster, error)
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (m *GormClusterRepository) TableName() string {
	return "cluster"
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (c Cluster) TableName() string {
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

// LoadByURL returns a single Cluster filtered using 'url'
func (m *GormClusterRepository) FindByURL(ctx context.Context, url string) (*Cluster, error) {
	defer goa.MeasureSince([]string{"goa", "db", "cluster", "loadClusterByURL"}, time.Now())
	var native Cluster
	// make sure that the URL to use during the search also has a trailing slash (see the Cluster.Normalize() method)
	err := m.db.Table(m.TableName()).Where("url = ?", httpsupport.AddTrailingSlashToURL(url)).Find(&native).Error
	if err == gorm.ErrRecordNotFound {
		return nil, errors.NewNotFoundErrorFromString(fmt.Sprintf("cluster with url '%s' not found", url))
	}
	return &native, errs.WithStack(err)
}

// Create creates a new record.
func (m *GormClusterRepository) Create(ctx context.Context, c *Cluster) error {
	defer goa.MeasureSince([]string{"goa", "db", "cluster", "create"}, time.Now())
	if c.ClusterID == uuid.Nil {
		c.ClusterID = uuid.NewV4()
	}
	err := c.Normalize()
	if err != nil {
		return errs.WithStack(err)
	}
	err = m.db.Create(c).Error
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

	existing, err := m.Load(ctx, c.ClusterID)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"cluster_id": c.ClusterID.String(),
			"err":        err,
		}, "unable to update cluster")
		return errs.WithStack(err)
	}
	return m.update(ctx, existing, c)
}

// update updates the existing cluster record with the given "new" one
func (m *GormClusterRepository) update(ctx context.Context, existing, c *Cluster) error {
	err := c.Normalize()
	if err != nil {
		return errs.WithStack(err)
	}
	err = m.db.Model(existing).Updates(c).Error
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"cluster_id": c.ClusterID.String(),
			"err":        err,
		}, "unable to update cluster")
		return errs.WithStack(err)
	}

	log.Debug(ctx, map[string]interface{}{
		"cluster_id": c.ClusterID.String(),
	}, "cluster saved")
	return nil
}

// CreateOrSave creates cluster or saves cluster if any cluster found using url
func (m *GormClusterRepository) CreateOrSave(ctx context.Context, c *Cluster) error {
	existing, err := m.FindByURL(ctx, c.URL)
	if err != nil {
		if ok, _ := errors.IsNotFoundError(err); ok {
			return m.Create(ctx, c)
		}
		log.Error(ctx, map[string]interface{}{
			"cluster_url": c.URL,
			"err":         err,
		}, "unable to load cluster")
		return errs.WithStack(err)
	}
	return m.update(ctx, existing, c)
}

// Delete removes a single record. This is a hard delete!
// Also, remove all identity/cluster relationship associated with this cluster to remove.
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

// Query exposes an open ended Query model
func (m *GormClusterRepository) Query(funcs ...func(*gorm.DB) *gorm.DB) ([]Cluster, error) {
	defer goa.MeasureSince([]string{"goa", "db", "cluster", "query"}, time.Now())
	var objs []Cluster

	err := m.db.Scopes(funcs...).Table(m.TableName()).Find(&objs).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errs.WithStack(err)
	}
	log.Debug(nil, map[string]interface{}{}, "cluster query done successfully!")

	return objs, nil
}

// List lists ALL clusters
func (m *GormClusterRepository) List(ctx context.Context) ([]Cluster, error) {
	return m.Query()

}
