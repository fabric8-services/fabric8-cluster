package service

import (
	"context"
	"fmt"
	"github.com/fabric8-services/fabric8-cluster/application/service"
	"github.com/fabric8-services/fabric8-cluster/application/service/base"
	servicectx "github.com/fabric8-services/fabric8-cluster/application/service/context"
	"github.com/fabric8-services/fabric8-cluster/cluster/repository"
	"github.com/fabric8-services/fabric8-cluster/configuration"
	"github.com/fabric8-services/fabric8-common/errors"
	"github.com/fabric8-services/fabric8-common/httpsupport"
	"github.com/fabric8-services/fabric8-common/log"
	"github.com/fsnotify/fsnotify"
	errs "github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"net/url"
	"strings"
	"time"
)

type clusterService struct {
	base.BaseService
	loader ConfigLoader
}

type ConfigLoader interface {
	ReloadClusterConfig() error
	GetClusterConfigurationFilePath() string
	GetClusters() map[string]configuration.Cluster
}

// NewClusterService creates a new cluster service
func NewClusterService(context servicectx.ServiceContext, loader ConfigLoader) service.ClusterService {
	return &clusterService{
		BaseService: base.NewBaseService(context),
		loader:      loader,
	}
}

// CreateOrSaveClusterFromConfig creates clusters or save updated cluster info from config
func (c clusterService) CreateOrSaveClusterFromConfig(ctx context.Context) error {
	for _, configCluster := range c.loader.GetClusters() {
		rc := &repository.Cluster{
			Name:              configCluster.Name,
			URL:               httpsupport.AddTrailingSlashToURL(configCluster.APIURL),
			ConsoleURL:        httpsupport.AddTrailingSlashToURL(configCluster.ConsoleURL),
			MetricsURL:        httpsupport.AddTrailingSlashToURL(configCluster.MetricsURL),
			LoggingURL:        httpsupport.AddTrailingSlashToURL(configCluster.LoggingURL),
			AppDNS:            configCluster.AppDNS,
			CapacityExhausted: configCluster.CapacityExhausted,
			Type:              configCluster.Type,

			SaToken:          configCluster.ServiceAccountToken,
			SaUsername:       configCluster.ServiceAccountUsername,
			SaTokenEncrypted: *configCluster.ServiceAccountTokenEncrypted,
			TokenProviderID:  configCluster.TokenProviderID,
			AuthClientID:     configCluster.AuthClientID,
			AuthClientSecret: configCluster.AuthClientSecret,
			AuthDefaultScope: configCluster.AuthClientDefaultScope,
		}

		err := c.ExecuteInTransaction(func() error {
			return c.Repositories().Clusters().CreateOrSave(ctx, rc)
		})

		if err != nil {
			return err
		}
	}
	return nil
}

// InitializeClusterWatcher initializes a file watcher for the cluster config file
// When the file is updated the configuration synchronously reload the cluster configuration
func (c clusterService) InitializeClusterWatcher() (func() error, error) {
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
					err = c.loader.ReloadClusterConfig()
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
						if err := c.CreateOrSaveClusterFromConfig(context.Background()); err != nil {
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
	configPath := c.loader.GetClusterConfigurationFilePath()

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
func (c clusterService) LinkIdentityToCluster(ctx context.Context, identityID, clusterURL string) error {
	if err := validateParams(identityID, clusterURL); err != nil {
		return errs.Wrapf(err, "failed to link identity %s to cluster with url '%s'", identityID, clusterURL)
	}

	id, err := uuid.FromString(identityID)
	if err != nil {
		return errors.NewBadParameterError("identity-id", fmt.Sprintf("identity-id %s is not a valid UUID", identityID))
	}

	rc, err := c.Repositories().Clusters().LoadClusterByURL(ctx, clusterURL)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"cluster_url": clusterURL,
			"err":         err,
		}, "failed to load cluster with url %s", clusterURL)
		return errors.NewBadParameterError("cluster-url", fmt.Sprintf("cluster with requested url %s doesn't exist", clusterURL))
	}

	clusterID := rc.ClusterID
	_, err = c.Repositories().IdentityClusters().Load(ctx, id, clusterID)
	if err != nil {
		if ok, _ := errors.IsNotFoundError(err); ok {
			return c.createIdentityCluster(ctx, id, clusterID)
		}
		return err
	}
	return nil
}

func validateParams(identityID string, url string) error {
	if strings.TrimSpace(identityID) == "" {
		return errors.NewBadParameterErrorFromString("empty field identity-id is not allowed")
	}

	if err := validateURL(&url); err != nil {
		return errors.NewBadParameterErrorFromString(fmt.Sprintf("cluster-url '%s' is invalid: %v", url, err))
	}
	return nil
}

// validateURL validates the URL: return an error if the given url could not be parsed or if it is missing
// the `scheme` or `host` parts.
func validateURL(urlStr *string) error {
	u, err := url.Parse(*urlStr)
	if err != nil {
		return err
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("missing scheme or host")
	}
	// make sure that the URL ends with a slash in all cases
	*urlStr = httpsupport.AddTrailingSlashToURL(*urlStr)
	return nil
}

func (c clusterService) createIdentityCluster(ctx context.Context, identityID, clusterID uuid.UUID) error {
	identityCluster := &repository.IdentityCluster{IdentityID: identityID, ClusterID: clusterID}

	return c.ExecuteInTransaction(func() error {
		if err := c.Repositories().IdentityClusters().Create(ctx, identityCluster); err != nil {
			return errs.Wrapf(err, "failed to link identity %s with cluster %s", identityID, clusterID)
		}
		return nil
	})
}
