package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/fabric8-services/fabric8-common/auth"

	"github.com/fabric8-services/fabric8-cluster/application/service"
	"github.com/fabric8-services/fabric8-cluster/application/service/base"
	servicectx "github.com/fabric8-services/fabric8-cluster/application/service/context"
	"github.com/fabric8-services/fabric8-cluster/cluster"
	"github.com/fabric8-services/fabric8-cluster/cluster/repository"
	"github.com/fabric8-services/fabric8-cluster/configuration"
	"github.com/fabric8-services/fabric8-common/errors"
	"github.com/fabric8-services/fabric8-common/log"

	"github.com/fsnotify/fsnotify"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

type clusterService struct {
	base.BaseService
	loader ConfigLoader
}

// ConfigLoader to interface for the config watcher/loader
type ConfigLoader interface {
	ReloadClusterConfig() error
	GetClusterConfigurationFilePath() string
	GetClusters() map[string]repository.Cluster
}

// NewClusterService creates a new cluster service with the default implementation
func NewClusterService(context servicectx.ServiceContext, loader ConfigLoader) service.ClusterService {
	return &clusterService{
		BaseService: base.NewBaseService(context),
		loader:      loader,
	}
}

// CreateOrSaveClusterFromConfig creates clusters or save updated cluster info from config
func (s clusterService) CreateOrSaveClusterFromConfig(ctx context.Context) error {
	log.Warn(ctx, map[string]interface{}{}, "creating/updating clusters from config file")
	for _, configCluster := range s.loader.GetClusters() {
		var err error
		rc := &repository.Cluster{
			Name:              configCluster.Name,
			URL:               configCluster.URL,
			ConsoleURL:        configCluster.ConsoleURL,
			MetricsURL:        configCluster.MetricsURL,
			LoggingURL:        configCluster.LoggingURL,
			AppDNS:            configCluster.AppDNS,
			CapacityExhausted: configCluster.CapacityExhausted,
			Type:              configCluster.Type,
			SAToken:           configCluster.SAToken,
			SAUsername:        configCluster.SAUsername,
			SATokenEncrypted:  configCluster.SATokenEncrypted,
			TokenProviderID:   configCluster.TokenProviderID,
			AuthClientID:      configCluster.AuthClientID,
			AuthClientSecret:  configCluster.AuthClientSecret,
			AuthDefaultScope:  configCluster.AuthDefaultScope,
		}
		err = s.ExecuteInTransaction(func() error {
			return s.Repositories().Clusters().CreateOrSave(ctx, rc)
		})
		if err != nil {
			return err
		}
	}
	log.Warn(ctx, map[string]interface{}{}, "creating/updating clusters from config file has been completed/done")
	return nil
}

// CreateOrSaveCluster creates clusters or save updated cluster info
func (s clusterService) CreateOrSaveCluster(ctx context.Context, clustr *repository.Cluster) error {
	// check that the token belongs to a user
	if !auth.IsSpecificServiceAccount(ctx, auth.ToolChainOperator) {
		log.Error(ctx, nil, "unauthorized access to cluster info")
		return errors.NewUnauthorizedError("unauthorized access to cluster info")
	}
	err := s.validate(ctx, clustr)
	if err != nil {
		return errs.Wrapf(err, "failed to create or save cluster named '%s'", clustr.Name)
	}
	return s.ExecuteInTransaction(func() error {
		return s.Repositories().Clusters().CreateOrSave(ctx, clustr)
	})
}

// Load loads the cluster given its ID, but without the sentitive info (token, etc.)
// This method is allowed for the following service accounts:
// - Auth
// - OSO Proxy
// - Tenant
// - Jenkins Idler
// - Jenkins Proxy
// returns a NotFoundError error if no cluster with the given ID exists, or an "error with stack" if something wrong happend
func (s clusterService) Load(ctx context.Context, clusterID uuid.UUID) (*repository.Cluster, error) {
	if !auth.IsSpecificServiceAccount(ctx, auth.OsoProxy, auth.Tenant, auth.JenkinsIdler, auth.JenkinsProxy, auth.Auth) {
		return nil, errors.NewUnauthorizedError("unauthorized access to cluster info")
	}
	result, err := s.Repositories().Clusters().Load(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	// hide all sensitive info from the cluster record to return
	result.AuthDefaultScope = ""
	result.AuthClientID = ""
	result.AuthClientSecret = ""
	result.SAToken = ""
	result.SAUsername = ""
	result.TokenProviderID = ""
	result.SATokenEncrypted = false
	return result, nil
}

// LoadForAuth loads the cluster given its ID, including the sentitive info (token, etc)
// This method is allowed for the 'Auth' service account only
// returns a NotFoundError error if no cluster with the given ID exists, or an "error with stack" if something wrong happend
func (s clusterService) LoadForAuth(ctx context.Context, clusterID uuid.UUID) (*repository.Cluster, error) {
	if !auth.IsSpecificServiceAccount(ctx, auth.Auth) {
		return nil, errors.NewUnauthorizedError("unauthorized access to cluster info")
	}
	return s.Repositories().Clusters().Load(ctx, clusterID)
}

// FindByURL loads the cluster given its URL, but without the sentitive info (token, etc.)
// This method is allowed for the following service accounts:
// - Auth
// - OSO Proxy
// - Tenant
// - Jenkins Idler
// - Jenkins Proxy
// returns a NotFoundError error if no cluster with the given ID exists, or an "error with stack" if something wrong happend
func (s clusterService) FindByURL(ctx context.Context, clusterURL string) (*repository.Cluster, error) {
	// check the `clusterURL` paraneter to make sure it's a valid URL
	err := validateURL(clusterURL)
	if err != nil {
		return nil, errors.NewBadParameterError("cluster-url", clusterURL)
	}
	// check the user token
	if !auth.IsSpecificServiceAccount(ctx, auth.OsoProxy, auth.Tenant, auth.JenkinsIdler, auth.JenkinsProxy, auth.Auth) {
		return nil, errors.NewUnauthorizedError("unauthorized access to cluster info")
	}
	result, err := s.Repositories().Clusters().FindByURL(ctx, clusterURL)
	if err != nil {
		return nil, err
	}
	// hide all sensitive info from the cluster record to return
	result.AuthDefaultScope = ""
	result.AuthClientID = ""
	result.AuthClientSecret = ""
	result.SAToken = ""
	result.SAUsername = ""
	result.TokenProviderID = ""
	result.SATokenEncrypted = false
	return result, nil
}

// FindByURLForAuth loads the cluster given its URL, including all sentitive info (token, etc.)
// This method is allowed for the 'auth' service account only.
// returns a NotFoundError error if no cluster with the given ID exists, or an "error with stack" if something wrong happend
func (s clusterService) FindByURLForAuth(ctx context.Context, clusterURL string) (*repository.Cluster, error) {
	// check the `clusterURL` paraneter to make sure it's a valid URL
	err := validateURL(clusterURL)
	if err != nil {
		return nil, errors.NewBadParameterError("cluster-url", clusterURL)
	}
	if !auth.IsSpecificServiceAccount(ctx, auth.Auth) {
		return nil, errors.NewUnauthorizedError("unauthorized access to cluster info")
	}
	result, err := s.Repositories().Clusters().FindByURL(ctx, clusterURL)
	if err != nil {
		return nil, err
	}
	return result, nil
}

const (
	// errEmptyFieldMsg the error template when a field is empty
	errEmptyFieldMsg = "empty field '%s' is not allowed"
	// errInvalidURLMsg the error template when an URL is invalid
	errInvalidURLMsg = "'%s' URL '%s' is invalid: %v"
	// errInvalidTypeMsg the error template when the type of cluster is invalid
	errInvalidTypeMsg = "invalid type of cluster: '%s' (expected 'OSO', 'OCP' or 'OSD')"
)

// validate checks if all data in the given cluster is valid, and fills the missing/optional URLs using the `APIURL`
func (s clusterService) validate(ctx context.Context, clustr *repository.Cluster) error {
	if strings.TrimSpace(clustr.Name) == "" {
		return errors.NewBadParameterErrorFromString(fmt.Sprintf(errEmptyFieldMsg, "name"))
	}
	err := validateURL(clustr.URL)
	if err != nil {
		return errors.NewBadParameterErrorFromString(fmt.Sprintf(errInvalidURLMsg, "API", clustr.URL, err))
	}
	if strings.TrimSpace(clustr.ConsoleURL) != "" {
		err := validateURL(clustr.ConsoleURL)
		if err != nil {
			return errors.NewBadParameterErrorFromString(fmt.Sprintf(errInvalidURLMsg, "console", clustr.ConsoleURL, err))
		}
	}
	if strings.TrimSpace(clustr.MetricsURL) != "" {
		err := validateURL(clustr.MetricsURL)
		if err != nil {
			return errors.NewBadParameterErrorFromString(fmt.Sprintf(errInvalidURLMsg, "metrics", clustr.MetricsURL, err))
		}
	}
	if strings.TrimSpace(clustr.LoggingURL) != "" {
		err = validateURL(clustr.LoggingURL)
		if err != nil {
			return errors.NewBadParameterErrorFromString(fmt.Sprintf(errInvalidURLMsg, "logging", clustr.LoggingURL, err))
		}
	}
	// validate the cluster type
	switch clustr.Type {
	case cluster.OSD, cluster.OCP, cluster.OSO:
		// ok
	default:
		return errors.NewBadParameterErrorFromString(fmt.Sprintf(errInvalidTypeMsg, clustr.Type))
	}
	// validate other non empty fields
	if strings.TrimSpace(clustr.AuthClientID) == "" {
		return errors.NewBadParameterErrorFromString(fmt.Sprintf(errEmptyFieldMsg, "auth-client-id"))
	}
	if strings.TrimSpace(clustr.AuthClientSecret) == "" {
		return errors.NewBadParameterErrorFromString(fmt.Sprintf(errEmptyFieldMsg, "auth-client-secret"))
	}
	if strings.TrimSpace(clustr.AuthDefaultScope) == "" {
		return errors.NewBadParameterErrorFromString(fmt.Sprintf(errEmptyFieldMsg, "auth-client-default-scope"))
	}
	if strings.TrimSpace(clustr.SAToken) == "" {
		return errors.NewBadParameterErrorFromString(fmt.Sprintf(errEmptyFieldMsg, "service-account-token"))
	}
	if strings.TrimSpace(clustr.SAUsername) == "" {
		return errors.NewBadParameterErrorFromString(fmt.Sprintf(errEmptyFieldMsg, "service-account-username"))
	}
	if strings.TrimSpace(clustr.TokenProviderID) == "" {
		existingClustr, err := s.Repositories().Clusters().FindByURL(ctx, clustr.URL)
		if err != nil {
			if notFound, _ := errors.IsNotFoundError(err); !notFound {
				// oops, something wrong happened, not just the cluster not found in the db
				return errs.Wrapf(err, "unable to validate cluster")
			}
		}

		if existingClustr != nil {
			// use the existing value in the DB
			clustr.TokenProviderID = existingClustr.TokenProviderID
		} else {
			// otherwise, assign same value as ID, for convenience
			clustr.ClusterID = uuid.NewV4()
			clustr.TokenProviderID = clustr.ClusterID.String()
		}
	}
	return nil
}

// validateURL validates the URL: return an error if the given url could not be parsed or if it is missing
// the `scheme` or `host` parts.
func validateURL(urlStr string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return err
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("missing scheme or host")
	}
	return nil
}

// Delete deletes the cluster identified by the given `clusterID`
func (s clusterService) Delete(ctx context.Context, clusterID uuid.UUID) error {
	// check that the token belongs to the `toolchain operator` SA
	if !auth.IsSpecificServiceAccount(ctx, auth.ToolChainOperator) {
		return errors.NewUnauthorizedError("unauthorized access to delete a cluster configuration")
	}
	return s.Repositories().Clusters().Delete(ctx, clusterID)
}

// InitializeClusterWatcher initializes a file watcher for the cluster config file
// When the file is updated the configuration synchronously reload the cluster configuration
func (s clusterService) InitializeClusterWatcher() (func() error, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Remove == fsnotify.Remove {
					time.Sleep(1 * time.Second) // Wait for one second before re-adding and reloading. It might be needed if the file is removed and then re-added in some environments
					err = watcher.Add(event.Name)
					if err != nil {
						log.Error(context.Background(), map[string]interface{}{
							"file": event.Name,
						}, "cluster config was removed but unable to re-add it to watcher")
					}
				}
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Remove == fsnotify.Remove {
					// Reload config if operation is Write or Remove.
					// Both can be part of file update depending on environment and actual operation.
					err = s.loader.ReloadClusterConfig()
					if err != nil {
						// Do not crash. Log the error and keep using the existing configuration
						log.Error(context.Background(), map[string]interface{}{
							"err":  err,
							"file": event.Name,
							"op":   event.Op.String(),
						}, "unable to reload cluster config file")
					} else {
						log.Info(context.Background(), map[string]interface{}{
							"file": event.Name,
							"op":   event.Op.String(),
						}, "cluster config file modified and reloaded")
						if err := s.CreateOrSaveClusterFromConfig(context.Background()); err != nil {
							// Do not crash. Log the error and keep using the existing configuration from DB
							log.Error(context.Background(), map[string]interface{}{
								"err":  err,
								"file": event.Name,
								"op":   event.Op.String(),
							}, "unable to save reloaded cluster config file")
						}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Error(context.Background(), map[string]interface{}{
					"err": err,
				}, "cluster config file watcher error")
			}
		}
	}()
	configPath := s.loader.GetClusterConfigurationFilePath()

	// this will make dev mode config path relative to current directory
	if configPath == "./configuration/conf-files/oso-clusters.conf" {
		configPath = "./../../" + configPath
	}
	configFilePath, err := configuration.PathExists(configPath)
	if err == nil && configFilePath != "" {
		err = watcher.Add(configFilePath)
		log.Info(context.Background(), map[string]interface{}{
			"file": configFilePath,
		}, "cluster config file watcher initialized")
	} else {
		// OK in Dev Mode
		log.Warn(context.Background(), map[string]interface{}{
			"file": configFilePath,
		}, "cluster config file watcher not initialized for non-existent file")
	}

	return watcher.Close, err
}

// LinkIdentityToCluster links Identity to Cluster
func (s clusterService) LinkIdentityToCluster(ctx context.Context, identityID uuid.UUID, clusterURL string, ignoreIfExists bool) error {
	if !auth.IsSpecificServiceAccount(ctx, auth.Auth) {
		log.Error(ctx, nil, "the account is not authorized to create identity cluster relationship")
		return errors.NewUnauthorizedError("account not authorized to create identity cluster relationship")
	}
	if err := validateURL(clusterURL); err != nil {
		return errors.NewBadParameterErrorFromString(fmt.Sprintf("cluster-url '%s' is invalid", clusterURL))
	}
	rc, err := s.Repositories().Clusters().FindByURL(ctx, clusterURL)
	if err != nil {
		return err
	}
	// do not fail silently even if identity is linked to cluster and ignoreIfExists is false
	if !ignoreIfExists {
		return s.createIdentityCluster(ctx, identityID, rc.ClusterID)
	}

	_, err = s.Repositories().IdentityClusters().Load(ctx, identityID, rc.ClusterID)
	if err != nil {
		if ok, _ := errors.IsNotFoundError(err); ok {
			return s.createIdentityCluster(ctx, identityID, rc.ClusterID)
		}
		return err
	}
	return nil
}

func (s clusterService) createIdentityCluster(ctx context.Context, identityID, clusterID uuid.UUID) error {
	identityCluster := &repository.IdentityCluster{IdentityID: identityID, ClusterID: clusterID}

	return s.ExecuteInTransaction(func() error {
		if err := s.Repositories().IdentityClusters().Create(ctx, identityCluster); err != nil {
			return errors.NewInternalErrorFromString(fmt.Sprintf("failed to link identity '%s' with cluster '%s': %v", identityID, clusterID, err))
		}
		return nil
	})
}

// RemoveIdentityToClusterLink removes Identity to Cluster link/relation
func (s clusterService) RemoveIdentityToClusterLink(ctx context.Context, identityID uuid.UUID, clusterURL string) error {
	if !auth.IsSpecificServiceAccount(ctx, auth.Auth) {
		log.Error(ctx, nil, "the account is not authorized to remove identity cluster relationship")
		return errors.NewUnauthorizedError("account not authorized to remove identity cluster relationship")
	}
	if err := validateURL(clusterURL); err != nil {
		return errors.NewBadParameterErrorFromString(fmt.Sprintf("cluster-url '%s' is invalid", clusterURL))
	}
	return s.ExecuteInTransaction(func() error {
		return s.Repositories().IdentityClusters().Delete(ctx, identityID, clusterURL)
	})
}

// List lists ALL clusters
// This method is allowed for the following service accounts:
// - Auth
// - OSO Proxy
// - Tenant
// - Jenkins Idler
// - Jenkins Proxy
func (s clusterService) List(ctx context.Context) ([]repository.Cluster, error) {
	if !auth.IsSpecificServiceAccount(ctx, auth.OsoProxy, auth.Tenant, auth.JenkinsIdler, auth.JenkinsProxy, auth.Auth) {
		return []repository.Cluster{}, errors.NewUnauthorizedError("unauthorized access to clusters info")
	}
	clusters, err := s.Repositories().Clusters().List(ctx)
	if err != nil {
		return []repository.Cluster{}, err
	}
	// hide all sensitive info in the cluster records to return
	for i, c := range clusters {
		c.AuthDefaultScope = ""
		c.AuthClientID = ""
		c.AuthClientSecret = ""
		c.SAToken = ""
		c.SAUsername = ""
		c.SATokenEncrypted = false
		c.TokenProviderID = ""
		// need to replace entry in slice since it's not a slice of pointers
		clusters[i] = c
	}
	return clusters, nil
}

// List lists ALL clusters, including sensitive information
// This method is allowed for the `Auth` service account only
func (s clusterService) ListForAuth(ctx context.Context) ([]repository.Cluster, error) {
	if !auth.IsSpecificServiceAccount(ctx, auth.Auth) {
		return []repository.Cluster{}, errors.NewUnauthorizedError("unauthorized access to clusters info")
	}
	clusters, err := s.Repositories().Clusters().List(ctx)
	if err != nil {
		return []repository.Cluster{}, err
	}
	return clusters, nil
}
