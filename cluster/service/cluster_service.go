package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/fabric8-services/fabric8-cluster/application/service"
	"github.com/fabric8-services/fabric8-cluster/application/service/base"
	servicectx "github.com/fabric8-services/fabric8-cluster/application/service/context"
	"github.com/fabric8-services/fabric8-cluster/cluster"
	"github.com/fabric8-services/fabric8-cluster/cluster/repository"
	"github.com/fabric8-services/fabric8-cluster/configuration"
	"github.com/fabric8-services/fabric8-common/errors"
	"github.com/fabric8-services/fabric8-common/httpsupport"
	"github.com/fabric8-services/fabric8-common/log"

	"github.com/fsnotify/fsnotify"
	errs "github.com/pkg/errors"
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

// NewClusterService creates a new cluster service with the default implementation
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
			SAToken:           configCluster.ServiceAccountToken,
			SAUsername:        configCluster.ServiceAccountUsername,
			TokenProviderID:   configCluster.TokenProviderID,
			AuthClientID:      configCluster.AuthClientID,
			AuthClientSecret:  configCluster.AuthClientSecret,
			AuthDefaultScope:  configCluster.AuthClientDefaultScope,
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

// CreateOrSaveCluster creates clusters or save updated cluster info
func (c clusterService) CreateOrSaveCluster(ctx context.Context, clustr *repository.Cluster) error {
	err := validate(clustr)
	if err != nil {
		return errs.Wrapf(err, "failed to create or save cluster named '%s'", clustr.Name)
	}
	return c.ExecuteInTransaction(func() error {
		return c.Repositories().Clusters().CreateOrSave(ctx, clustr)
	})
}

const (
	errEmptyFieldMsg           = "empty field '%s' is not allowed"
	errInvalidURLMsg           = "'%s' URL is invalid: '%s'"
	errInvalidURLGenerationMsg = "unable to generate '%s' URL from '%s' (expected an 'api' subdomain)"
	errInvalidTypeMsg          = "invalid type of cluster: '%s' (expected 'OSO', 'OCP' or 'OSD')"
)

// validate checks if all data in the given cluster is valid, and fills the missing/optional URLs using the `APIURL`
func validate(clustr *repository.Cluster) error {
	if clustr.Name == "" {
		return errors.NewBadParameterErrorFromString(fmt.Sprintf(errEmptyFieldMsg, "name"))
	}
	apiURL, valid := validURL(clustr.URL)
	if !valid {
		return errors.NewBadParameterErrorFromString(fmt.Sprintf(errInvalidURLMsg, "API", clustr.URL))
	}
	// check other urls (console, logging, metrics)
	for kind, urlStr := range map[string]*string{
		"console": &clustr.ConsoleURL,
		"logging": &clustr.LoggingURL,
		"metrics": &clustr.MetricsURL} {
		// check the url
		if *urlStr == "" {
			// "forge" the console-url from the api-url
			forgedURL, err := forgeURL(apiURL, kind)
			if err != nil {
				return err
			}
			*urlStr = forgedURL
		} else if _, valid := validURL(*urlStr); !valid {
			// validate the URL
			return errors.NewBadParameterErrorFromString(fmt.Sprintf(errInvalidURLMsg, kind, *urlStr))
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
	if clustr.AuthClientID == "" {
		return errors.NewBadParameterErrorFromString(fmt.Sprintf(errEmptyFieldMsg, "auth-client-id"))
	}
	if clustr.AuthClientSecret == "" {
		return errors.NewBadParameterErrorFromString(fmt.Sprintf(errEmptyFieldMsg, "auth-client-secret"))
	}
	if clustr.AuthDefaultScope == "" {
		return errors.NewBadParameterErrorFromString(fmt.Sprintf(errEmptyFieldMsg, "auth-client-default-scope"))
	}
	if clustr.SAToken == "" {
		return errors.NewBadParameterErrorFromString(fmt.Sprintf(errEmptyFieldMsg, "service-account-token"))
	}
	if clustr.SAUsername == "" {
		return errors.NewBadParameterErrorFromString(fmt.Sprintf(errEmptyFieldMsg, "service-account-username"))
	}
	if clustr.TokenProviderID == "" {
		return errors.NewBadParameterErrorFromString(fmt.Sprintf(errEmptyFieldMsg, "token-provider-id"))
	}
	return nil
}

// validateURL validates the URL: return `false` if the given url could not be parsed or if it is missing
// the `scheme` or `host` parts. Returns `true` otherwise.
func validURL(urlStr string) (url.URL, bool) {
	u, err := url.Parse(urlStr)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return url.URL{}, false
	}
	return *u, true
}

func forgeURL(baseURL url.URL, subdomain string) (string, error) {
	domains := strings.Split(baseURL.Host, ".")
	if len(domains) < 3 || domains[0] != "api" {
		return "", errors.NewBadParameterErrorFromString(fmt.Sprintf(errInvalidURLGenerationMsg, subdomain, baseURL.String()))
	}
	domains[0] = subdomain
	return fmt.Sprintf("%s://%s%s", baseURL.Scheme, strings.Join(domains, "."), baseURL.Path), nil
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
