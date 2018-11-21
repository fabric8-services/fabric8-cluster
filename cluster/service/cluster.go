package service

import (
	"context"
	"github.com/fabric8-services/fabric8-cluster/application/service"
	"github.com/fabric8-services/fabric8-cluster/application/service/base"
	servicectx "github.com/fabric8-services/fabric8-cluster/application/service/context"
	"github.com/fabric8-services/fabric8-cluster/cluster/repository"
	"github.com/fabric8-services/fabric8-cluster/configuration"
	"github.com/fabric8-services/fabric8-common/httpsupport"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"time"
)

const (
	OSD = "OSD"
	OCP = "OCP"
	OSO = "OSO"
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
	for _, clusterConfig := range c.loader.GetOSOClusters() {
		cluster := &repository.Cluster{
			Name:       clusterConfig.Name,
			URL:        httpsupport.AddTrailingSlashToURL(clusterConfig.APIURL),
			ConsoleURL: httpsupport.AddTrailingSlashToURL(clusterConfig.ConsoleURL),
			MetricsURL: httpsupport.AddTrailingSlashToURL(clusterConfig.MetricsURL),
			LoggingURL: httpsupport.AddTrailingSlashToURL(clusterConfig.LoggingURL),
			AppDNS:     clusterConfig.AppDNS,
			//CapacityExhausted: clusterConfig.CapacityExhausted,

			SaToken:          clusterConfig.ServiceAccountToken,
			SaUsername:       clusterConfig.ServiceAccountUsername,
			TokenProviderID:  clusterConfig.TokenProviderID,
			AuthClientID:     clusterConfig.AuthClientID,
			AuthClientSecret: clusterConfig.AuthClientSecret,
			AuthDefaultScope: clusterConfig.AuthClientDefaultScope,
			Type:             OSO,
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
						log.WithFields(map[string]interface{}{
							"file": event.Name,
						}).Errorln("cluster config was removed but unable to re-add it to watcher")
					}
				}
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Remove == fsnotify.Remove {
					// Reload config if operation is Write or Remove.
					// Both can be part of file update depending on environment and actual operation.
					err = c.loader.ReloadClusterConfig()
					if err != nil {
						// Do not crash. Log the error and keep using the existing configuration
						log.WithFields(map[string]interface{}{
							"err":  err,
							"file": event.Name,
							"op":   event.Op.String(),
						}).Errorln("unable to reload cluster config file")
					} else {
						log.WithFields(map[string]interface{}{
							"file": event.Name,
							"op":   event.Op.String(),
						}).Infoln("cluster config file modified and reloaded")
					}
					if err := c.CreateOrSaveOSOClusterFromConfig(context.Background()); err != nil {
						// Do not crash. Log the error and keep using the existing configuration from DB
						log.WithFields(map[string]interface{}{
							"err":  err,
							"file": event.Name,
							"op":   event.Op.String(),
						}).Errorln("unable to save reloaded cluster config file")
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.WithFields(map[string]interface{}{
					"err": err,
				}).Errorln("cluster config file watcher error")
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
		log.WithFields(map[string]interface{}{
			"file": configFilePath,
		}).Infoln("cluster config file watcher initialized")
	} else {
		// OK in Dev Mode
		log.WithFields(map[string]interface{}{
			"file": configFilePath,
		}).Warnln("cluster config file watcher not initialized for non-existent file")
	}

	return watcher.Close, err
}
