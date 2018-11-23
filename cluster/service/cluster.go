package service

import (
	"context"
	"github.com/fabric8-services/fabric8-cluster/application/service"
	"github.com/fabric8-services/fabric8-cluster/application/service/base"
	servicectx "github.com/fabric8-services/fabric8-cluster/application/service/context"
	"github.com/fabric8-services/fabric8-cluster/cluster/repository"
	"github.com/fabric8-services/fabric8-cluster/configuration"
	"github.com/fabric8-services/fabric8-common/httpsupport"
	"github.com/fabric8-services/fabric8-common/log"
	"github.com/fsnotify/fsnotify"
	"time"
)

type clusterService struct {
	base.BaseService
	loader ConfigLoader
}

type ConfigLoader interface {
	ReloadClusterConfig() error
	GetOSOConfigurationFilePath() string
	GetOSOClusters() map[string]configuration.OSOCluster
}

// NewClusterService creates a new cluster service
func NewClusterService(context servicectx.ServiceContext, loader ConfigLoader) service.ClusterService {
	return &clusterService{
		BaseService: base.NewBaseService(context),
		loader:      loader,
	}
}

// CreateOrSaveOSOClusterFromConfig creates clusters or save updated cluster info from config
func (c clusterService) CreateOrSaveOSOClusterFromConfig(ctx context.Context) error {
	for _, osoCluster := range c.loader.GetOSOClusters() {
		cluster := &repository.Cluster{
			Name:              osoCluster.Name,
			URL:               httpsupport.AddTrailingSlashToURL(osoCluster.APIURL),
			ConsoleURL:        httpsupport.AddTrailingSlashToURL(osoCluster.ConsoleURL),
			MetricsURL:        httpsupport.AddTrailingSlashToURL(osoCluster.MetricsURL),
			LoggingURL:        httpsupport.AddTrailingSlashToURL(osoCluster.LoggingURL),
			AppDNS:            osoCluster.AppDNS,
			CapacityExhausted: osoCluster.CapacityExhausted,
			Type:              osoCluster.Type,

			SaToken:          osoCluster.ServiceAccountToken,
			SaUsername:       osoCluster.ServiceAccountUsername,
			TokenProviderID:  osoCluster.TokenProviderID,
			AuthClientID:     osoCluster.AuthClientID,
			AuthClientSecret: osoCluster.AuthClientSecret,
			AuthDefaultScope: osoCluster.AuthClientDefaultScope,
		}

		err := c.ExecuteInTransaction(func() error {
			return c.Repositories().Clusters().CreateOrSave(ctx, cluster)
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
						if err := c.CreateOrSaveOSOClusterFromConfig(context.Background()); err != nil {
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
	osoConfigPath := c.loader.GetOSOConfigurationFilePath()

	// this will make dev mode config path relative to current directory
	if osoConfigPath == "./configuration/conf-files/oso-clusters.conf" {
		osoConfigPath = "./../../" + osoConfigPath
	}
	configFilePath, err := configuration.PathExists(osoConfigPath)
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
