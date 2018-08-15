package configuration

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/fabric8-services/fabric8-cluster/rest"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

// String returns the current configuration as a string
func (c *ConfigurationData) String() string {
	allSettings := c.v.AllSettings()
	y, err := yaml.Marshal(&allSettings)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"settings": allSettings,
			"err":      err,
		}).Panicln("Failed to marshall config to string")
	}
	return fmt.Sprintf("%s\n", y)
}

const (
	// Constants for viper variable names. Will be used to set
	// default values as well as to get each value

	// General
	varHTTPAddress          = "http.address"
	varMetricsHTTPAddress   = "metrics.http.address"
	varDeveloperModeEnabled = "developer.mode.enabled"
	varCleanTestDataEnabled = "clean.test.data"
	varDBLogsEnabled        = "enable.db.logs"
	defaultConfigFile       = "config.yaml"
	varLogLevel             = "log.level"
	varLogJSON              = "log.json"

	// Postgres
	varPostgresHost                 = "postgres.host"
	varPostgresPort                 = "postgres.port"
	varPostgresUser                 = "postgres.user"
	varPostgresDatabase             = "postgres.database"
	varPostgresPassword             = "postgres.password"
	varPostgresSSLMode              = "postgres.sslmode"
	varPostgresConnectionTimeout    = "postgres.connection.timeout"
	varPostgresTransactionTimeout   = "postgres.transaction.timeout"
	varPostgresConnectionRetrySleep = "postgres.connection.retrysleep"
	varPostgresConnectionMaxIdle    = "postgres.connection.maxidle"
	varPostgresConnectionMaxOpen    = "postgres.connection.maxopen"

	// Other services URLs
	varClusterServiceURL = "cluster.url"
	varAuthURL           = "cluster.auth.url"
	varAuthKeysPath      = "cluster.auth.keys.path"

	// sentry
	varEnvironment = "environment"
	varSentryDSN   = "sentry.dsn"
)

type osoClusterConfig struct {
	Clusters []OSOCluster
}

// OSOCluster represents an OSO cluster configuration
type OSOCluster struct {
	Name                   string `mapstructure:"name"`
	APIURL                 string `mapstructure:"api-url"`
	ConsoleURL             string `mapstructure:"console-url"` // Optional in oso-clusters.conf
	MetricsURL             string `mapstructure:"metrics-url"` // Optional in oso-clusters.conf
	LoggingURL             string `mapstructure:"logging-url"` // Optional in oso-clusters.conf
	AppDNS                 string `mapstructure:"app-dns"`
	ServiceAccountToken    string `mapstructure:"service-account-token"`
	ServiceAccountUsername string `mapstructure:"service-account-username"`
	TokenProviderID        string `mapstructure:"token-provider-id"`
	AuthClientID           string `mapstructure:"auth-client-id"`
	AuthClientSecret       string `mapstructure:"auth-client-secret"`
	AuthClientDefaultScope string `mapstructure:"auth-client-default-scope"`
	CapacityExhausted      bool   `mapstructure:"capacity-exhausted"` // Optional in oso-clusters.conf ('false' by default)
}

// ConfigurationData encapsulates the Viper configuration object which stores the configuration data in-memory.
type ConfigurationData struct {
	// Main Configuration
	v *viper.Viper

	// OSO Cluster Configuration is a map of clusters where the key == the OSO cluster API URL
	clusters              map[string]OSOCluster
	clusterConfigFilePath string

	defaultConfigurationError error

	mux sync.RWMutex
}

// NewConfigurationData creates a configuration reader object using configurable configuration file paths
func NewConfigurationData(mainConfigFile string, osoClusterConfigFile string) (*ConfigurationData, error) {
	c := &ConfigurationData{
		v: viper.New(),
	}

	// Set up the main configuration
	c.v.SetEnvPrefix("CLUSTER")
	c.v.AutomaticEnv()
	c.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	c.v.SetTypeByDefaultValue(true)
	c.setConfigDefaults()

	if mainConfigFile != "" {
		c.v.SetConfigType("yaml")
		c.v.SetConfigFile(mainConfigFile)
		err := c.v.ReadInConfig() // Find and read the config file
		if err != nil {           // Handle errors reading the config file
			return nil, errors.Errorf("Fatal error config file: %s \n", err)
		}
	}

	// Set up the OSO cluster configuration (stored in a separate config file)
	clusterConfigFilePath, err := c.initClusterConfig(osoClusterConfigFile, defaultOsoClusterConfigPath)
	if err != nil {
		return nil, err
	}
	c.clusterConfigFilePath = clusterConfigFilePath

	// Check sensitive default configuration
	if c.DeveloperModeEnabled() {
		c.appendDefaultConfigErrorMessage("developer Mode is enabled")
	}
	if c.GetPostgresPassword() == defaultDBPassword {
		c.appendDefaultConfigErrorMessage("default DB password is used")
	}
	c.validateURL(c.GetClusterServiceURL(), "Cluster service")
	if c.GetClusterServiceURL() == "http://localhost" {
		c.appendDefaultConfigErrorMessage("environment is expected to be set to 'production' or 'prod-preview'")
	}
	c.validateURL(c.GetAuthServiceURL(), "Auth service")
	if c.GetSentryDSN() == "" {
		c.appendDefaultConfigErrorMessage("Sentry DSN is empty")
	}
	if c.defaultConfigurationError != nil {
		log.WithFields(map[string]interface{}{
			"default_configuration_error": c.defaultConfigurationError.Error(),
		}).Warningln("Default config is used! This is OK in Dev Mode.")
	}

	return c, nil
}

func (c *ConfigurationData) validateURL(serviceURL, serviceName string) {
	if serviceURL == "" {
		c.appendDefaultConfigErrorMessage(fmt.Sprintf("%s url is empty", serviceName))
	} else {
		_, err := url.Parse(serviceURL)
		if err != nil {
			c.appendDefaultConfigErrorMessage(fmt.Sprintf("invalid %s url: %s", serviceName, err.Error()))
		}
	}
}

func (c *ConfigurationData) initClusterConfig(osoClusterConfigFile, defaultClusterConfigFile string) (string, error) {
	clusterViper, defaultConfigErrorMsg, usedClusterConfigFile, err := readFromJSONFile(osoClusterConfigFile, defaultClusterConfigFile, osoClusterConfigFileName)
	if err != nil {
		return usedClusterConfigFile, err
	}
	if defaultConfigErrorMsg != nil {
		c.appendDefaultConfigErrorMessage(*defaultConfigErrorMsg)
	}

	var clusterConf osoClusterConfig
	err = clusterViper.Unmarshal(&clusterConf)
	if err != nil {
		return usedClusterConfigFile, err
	}
	c.clusters = map[string]OSOCluster{}
	for _, cluster := range clusterConf.Clusters {
		if cluster.ConsoleURL == "" {
			cluster.ConsoleURL, err = convertAPIURL(cluster.APIURL, "console", "console")
			if err != nil {
				return usedClusterConfigFile, err
			}
		}
		if cluster.MetricsURL == "" {
			cluster.MetricsURL, err = convertAPIURL(cluster.APIURL, "metrics", "")
			if err != nil {
				return usedClusterConfigFile, err
			}
		}
		if cluster.LoggingURL == "" {
			// This is not a typo; the logging host is the same as the console host in current k8s
			cluster.LoggingURL, err = convertAPIURL(cluster.APIURL, "console", "console")
			if err != nil {
				return usedClusterConfigFile, err
			}
		}
		c.clusters[cluster.APIURL] = cluster
	}

	err = c.checkClusterConfig()
	return usedClusterConfigFile, err
}

// checkClusterConfig checks if there is any missing keys or empty values in oso-clusters.conf
func (c *ConfigurationData) checkClusterConfig() error {
	if len(c.clusters) == 0 {
		return errors.New("empty cluster config file")
	}

	err := errors.New("")
	ok := true
	for _, cluster := range c.clusters {
		iVal := reflect.ValueOf(&cluster).Elem()
		typ := iVal.Type()
		for i := 0; i < iVal.NumField(); i++ {
			f := iVal.Field(i)
			tag := typ.Field(i).Tag.Get("mapstructure")
			switch f.Interface().(type) {
			case string:
				if f.String() == "" {
					err = errors.Errorf("%s; key %v is missing in cluster config", err.Error(), tag)
					ok = false
				}
			case bool:
				// Ignore
			default:
				err = errors.Errorf("%s; wrong type of key %v", err.Error(), tag)
				ok = false
			}
		}
	}
	if !ok {
		return err
	}
	return nil
}

func convertAPIURL(apiURL string, newPrefix string, newPath string) (string, error) {
	newURL, err := url.Parse(apiURL)
	if err != nil {
		return "", err
	}
	newHost, err := rest.ReplaceDomainPrefix(newURL.Host, newPrefix)
	if err != nil {
		return "", err
	}
	newURL.Host = newHost
	newURL.Path = newPath
	return newURL.String(), nil
}

func readFromJSONFile(configFilePath string, defaultConfigFilePath string, configFileName string) (*viper.Viper, *string, string, error) {
	jsonViper := viper.New()
	jsonViper.SetTypeByDefaultValue(true)

	var err error
	var etcJSONConfigUsed bool
	var defaultConfigErrorMsg *string
	if configFilePath != "" {
		// If a JSON configuration file has been specified, check if it exists
		if _, err := os.Stat(configFilePath); err != nil {
			return nil, nil, configFilePath, err
		}
	} else {
		// If the JSON configuration file has not been specified
		// then we default to <defaultConfigFile>
		configFilePath, err = pathExists(defaultConfigFilePath)
		if err != nil {
			return nil, nil, defaultConfigFilePath, err
		}
		etcJSONConfigUsed = configFilePath != ""
	}

	if !etcJSONConfigUsed {
		errMsg := fmt.Sprintf("%s is not used", defaultConfigFilePath)
		defaultConfigErrorMsg = &errMsg
	}
	usedFile := configFilePath

	jsonViper.SetConfigType("json")
	if configFilePath == "" {
		// Load the built-in config file (used in dev mode)
		usedFile = "./configuration/conf-files/" + configFileName
		data, err := Asset(configFileName)
		if err != nil {
			return nil, nil, usedFile, err
		}
		jsonViper.ReadConfig(bytes.NewBuffer(data))
	} else {
		jsonViper.SetConfigFile(configFilePath)
		err := jsonViper.ReadInConfig()
		if err != nil {
			return nil, nil, usedFile, errors.Errorf("failed to load the JSON config file (%s): %s \n", configFilePath, err)
		}
	}

	return jsonViper, defaultConfigErrorMsg, usedFile, nil
}

func (c *ConfigurationData) appendDefaultConfigErrorMessage(message string) {
	if c.defaultConfigurationError == nil {
		c.defaultConfigurationError = errors.New(message)
	} else {
		c.defaultConfigurationError = errors.Errorf("%s; %s", c.defaultConfigurationError.Error(), message)
	}
}

func pathExists(pathToCheck string) (string, error) {
	_, err := os.Stat(pathToCheck)
	if err == nil {
		return pathToCheck, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	return "", nil
}

func getMainConfigFile() string {
	// This was either passed as a env var or set inside main.go from --config
	envConfigPath, _ := os.LookupEnv("CLUSTER_CONFIG_FILE_PATH")
	return envConfigPath
}

func getOSOClusterConfigFile() string {
	envOSOClusterConfigFile, _ := os.LookupEnv("CLUSTER_OSO_CLUSTER_CONFIG_FILE")
	return envOSOClusterConfigFile
}

// InitializeClusterWatcher initializes a file watcher for the cluster config file
// When the file is updated the configuration synchronously reload the cluster configuration
func (c *ConfigurationData) InitializeClusterWatcher() (func() error, error) {
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
					err := c.reloadClusterConfig()
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

	configFilePath, err := pathExists(c.clusterConfigFilePath)
	if err == nil && configFilePath != "" {
		err = watcher.Add(configFilePath)
		log.WithFields(map[string]interface{}{
			"file": c.clusterConfigFilePath,
		}).Infoln("cluster config file watcher initialized")
	} else {
		// OK in Dev Mode
		log.WithFields(map[string]interface{}{
			"file": c.clusterConfigFilePath,
		}).Warnln("cluster config file watcher not initialized for non-existent file")
	}

	return watcher.Close, err
}

func (c *ConfigurationData) reloadClusterConfig() error {
	c.mux.Lock()
	defer c.mux.Unlock()

	_, err := c.initClusterConfig("", c.clusterConfigFilePath)
	return err
}

// DefaultConfigurationError returns an error if the default values is used
// for sensitive configuration like service account secrets or private keys.
// Error contains all the details.
// Returns nil if the default configuration is not used.
func (c *ConfigurationData) DefaultConfigurationError() error {
	// Lock for reading because config file watcher can update config errors
	c.mux.RLock()
	defer c.mux.RUnlock()

	return c.defaultConfigurationError
}

// GetClusterServiceUrl returns Cluster Service URL
func (c *ConfigurationData) GetClusterServiceURL() string {
	if c.v.IsSet(varClusterServiceURL) {
		return c.v.GetString(varClusterServiceURL)
	}
	switch c.GetEnvironment() {
	case prodEnvironment:
		return "https://cluster.openshift.io"
	case prodPreviewEnvironment:
		return "https://cluster.prod-preview.openshift.io"
	default:
		return "http://localhost"
	}
}

// GetAuthServiceUrl returns Auth Service URL
func (c *ConfigurationData) GetAuthServiceURL() string {
	if c.v.IsSet(varAuthURL) {
		return c.v.GetString(varAuthURL)
	}
	if c.DeveloperModeEnabled() {
		return "https://auth.prod-preview.openshift.io"
	}
	return ""
}

// GetAuthKeysPath returns the path to auth keys endpoint
func (c *ConfigurationData) GetAuthKeysPath() string {
	return c.v.GetString(varAuthKeysPath)
}

// GetOSOClusters returns a map of OSO cluster configurations by cluster API URL
func (c *ConfigurationData) GetOSOClusters() map[string]OSOCluster {
	// Lock for reading because config file watcher can update cluster configuration
	c.mux.RLock()
	defer c.mux.RUnlock()
	return c.clusters
}

// GetOSOClusterByURL returns a OSO cluster configurations by matching URL
// Regardless of trailing slashes if cluster API URL == "https://api.openshift.com"
// or "https://api.openshift.com/" it will match any "https://api.openshift.com*"
// like "https://api.openshift.com", "https://api.openshift.com/", or "https://api.openshift.com/patch"
// Returns nil if no matching API URL found
func (c *ConfigurationData) GetOSOClusterByURL(url string) *OSOCluster {
	// Lock for reading because config file watcher can update cluster configuration
	c.mux.RLock()
	defer c.mux.RUnlock()

	for apiURL, cluster := range c.clusters {
		if strings.HasPrefix(rest.AddTrailingSlashToURL(url), apiURL) {
			return &cluster
		}
	}

	return nil
}

// GetDefaultConfigurationFile returns the default configuration file.
func (c *ConfigurationData) GetDefaultConfigurationFile() string {
	return defaultConfigFile
}

// GetConfigurationData is a wrapper over NewConfigurationData which reads configuration file path
// from the environment variable.
func GetConfigurationData() (*ConfigurationData, error) {
	return NewConfigurationData(getMainConfigFile(), getOSOClusterConfigFile())
}

func (c *ConfigurationData) setConfigDefaults() {
	//---------
	// Postgres
	//---------

	// We already call this in NewConfigurationData() - do we need it again??
	c.v.SetTypeByDefaultValue(true)

	c.v.SetDefault(varPostgresHost, "localhost")
	c.v.SetDefault(varPostgresPort, 5434)
	c.v.SetDefault(varPostgresUser, "postgres")
	c.v.SetDefault(varPostgresDatabase, "postgres")
	c.v.SetDefault(varPostgresPassword, defaultDBPassword)
	c.v.SetDefault(varPostgresSSLMode, "disable")
	c.v.SetDefault(varPostgresConnectionTimeout, 5)
	c.v.SetDefault(varPostgresConnectionMaxIdle, -1)
	c.v.SetDefault(varPostgresConnectionMaxOpen, -1)

	// Number of seconds to wait before trying to connect again
	c.v.SetDefault(varPostgresConnectionRetrySleep, time.Duration(time.Second))

	// Timeout of a transaction in minutes
	c.v.SetDefault(varPostgresTransactionTimeout, time.Duration(5*time.Minute))

	//-----
	// HTTP
	//-----
	c.v.SetDefault(varHTTPAddress, "0.0.0.0:8087")
	c.v.SetDefault(varMetricsHTTPAddress, "0.0.0.0:8087")

	//-----
	// Misc
	//-----

	// Enable development related features
	c.v.SetDefault(varDeveloperModeEnabled, false)

	c.v.SetDefault(varLogLevel, defaultLogLevel)

	// By default, test data should be cleaned from DB, unless explicitely said otherwise.
	c.v.SetDefault(varCleanTestDataEnabled, true)
	// By default, DB logs are not output in the console
	c.v.SetDefault(varDBLogsEnabled, false)

	// prod-preview or prod
	c.v.SetDefault(varEnvironment, "local")

	c.v.SetDefault(varAuthKeysPath, "/api/token/keys")
}

// GetPostgresHost returns the postgres host as set via default, config file, or environment variable
func (c *ConfigurationData) GetPostgresHost() string {
	return c.v.GetString(varPostgresHost)
}

// GetPostgresPort returns the postgres port as set via default, config file, or environment variable
func (c *ConfigurationData) GetPostgresPort() int64 {
	return c.v.GetInt64(varPostgresPort)
}

// GetPostgresUser returns the postgres user as set via default, config file, or environment variable
func (c *ConfigurationData) GetPostgresUser() string {
	return c.v.GetString(varPostgresUser)
}

// GetPostgresDatabase returns the postgres database as set via default, config file, or environment variable
func (c *ConfigurationData) GetPostgresDatabase() string {
	return c.v.GetString(varPostgresDatabase)
}

// GetPostgresPassword returns the postgres password as set via default, config file, or environment variable
func (c *ConfigurationData) GetPostgresPassword() string {
	return c.v.GetString(varPostgresPassword)
}

// GetPostgresSSLMode returns the postgres sslmode as set via default, config file, or environment variable
func (c *ConfigurationData) GetPostgresSSLMode() string {
	return c.v.GetString(varPostgresSSLMode)
}

// GetPostgresConnectionTimeout returns the postgres connection timeout as set via default, config file, or environment variable
func (c *ConfigurationData) GetPostgresConnectionTimeout() int64 {
	return c.v.GetInt64(varPostgresConnectionTimeout)
}

// GetPostgresConnectionRetrySleep returns the number of seconds (as set via default, config file, or environment variable)
// to wait before trying to connect again
func (c *ConfigurationData) GetPostgresConnectionRetrySleep() time.Duration {
	return c.v.GetDuration(varPostgresConnectionRetrySleep)
}

// GetPostgresTransactionTimeout returns the number of minutes to timeout a transaction
func (c *ConfigurationData) GetPostgresTransactionTimeout() time.Duration {
	return c.v.GetDuration(varPostgresTransactionTimeout)
}

// GetPostgresConnectionMaxIdle returns the number of connections that should be keept alive in the database connection pool at
// any given time. -1 represents no restrictions/default behavior
func (c *ConfigurationData) GetPostgresConnectionMaxIdle() int {
	return c.v.GetInt(varPostgresConnectionMaxIdle)
}

// GetPostgresConnectionMaxOpen returns the max number of open connections that should be open in the database connection pool.
// -1 represents no restrictions/default behavior
func (c *ConfigurationData) GetPostgresConnectionMaxOpen() int {
	return c.v.GetInt(varPostgresConnectionMaxOpen)
}

// GetPostgresConfigString returns a ready to use string for usage in sql.Open()
func (c *ConfigurationData) GetPostgresConfigString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d",
		c.GetPostgresHost(),
		c.GetPostgresPort(),
		c.GetPostgresUser(),
		c.GetPostgresPassword(),
		c.GetPostgresDatabase(),
		c.GetPostgresSSLMode(),
		c.GetPostgresConnectionTimeout(),
	)
}

// GetHTTPAddress returns the HTTP address (as set via default, config file, or environment variable)
// that the cluster server binds to (e.g. "0.0.0.0:8087")
func (c *ConfigurationData) GetHTTPAddress() string {
	return c.v.GetString(varHTTPAddress)
}

// GetMetricsHTTPAddress returns the address the /metrics endpoing will be mounted.
// By default GetMetricsHTTPAddress is the same as GetHTTPAddress
func (c *ConfigurationData) GetMetricsHTTPAddress() string {
	return c.v.GetString(varMetricsHTTPAddress)
}

// DeveloperModeEnabled returns if development related features (as set via default, config file, or environment variable),
// e.g. token generation endpoint are enabled
func (c *ConfigurationData) DeveloperModeEnabled() bool {
	return c.v.GetBool(varDeveloperModeEnabled)
}

// IsCleanTestDataEnabled returns `true` if the test data should be cleaned after each test. (default: true)
func (c *ConfigurationData) IsCleanTestDataEnabled() bool {
	return c.v.GetBool(varCleanTestDataEnabled)
}

// IsDBLogsEnabled returns `true` if the DB logs (ie, SQL queries) should be output in the console. (default: false)
func (c *ConfigurationData) IsDBLogsEnabled() bool {
	return c.v.GetBool(varDBLogsEnabled)
}

// GetDevModePublicKey returns additional public key and its ID which should be used by the Cluster service in Dev Mode
// For example a public key from Keycloak
// Returns false if in in Dev Mode
func (c *ConfigurationData) GetDevModePublicKey() (bool, []byte, string) {
	if c.DeveloperModeEnabled() {
		return true, []byte(devModePublicKey), devModePublicKeyID
	}
	return false, nil, ""
}

// GetSentryDSN returns the secret needed to securely communicate with https://errortracking.prod-preview.openshift.io/openshift_io/fabric8-cluster/
func (c *ConfigurationData) GetSentryDSN() string {
	return c.v.GetString(varSentryDSN)
}

// GetLogLevel returns the logging level (as set via config file or environment variable)
func (c *ConfigurationData) GetLogLevel() string {
	return c.v.GetString(varLogLevel)
}

// IsLogJSON returns if we should log json format (as set via config file or environment variable)
func (c *ConfigurationData) IsLogJSON() bool {
	if c.v.IsSet(varLogJSON) {
		return c.v.GetBool(varLogJSON)
	}
	if c.DeveloperModeEnabled() {
		return false
	}
	return true
}

// GetEnvironment returns the current environment application is deployed in
// like 'production', 'prod-preview', 'local', etc as the value of environment variable
// `CLUSTER_ENVIRONMENT` is set.
func (c *ConfigurationData) GetEnvironment() string {
	return c.v.GetString(varEnvironment)
}
