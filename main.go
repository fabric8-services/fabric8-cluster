package main

import (
	"flag"
	"net/http"
	"os"
	"os/user"
	"runtime"
	"time"

	"github.com/fabric8-services/fabric8-cluster/app"
	"github.com/fabric8-services/fabric8-cluster/application/transaction"
	"github.com/fabric8-services/fabric8-cluster/configuration"
	"github.com/fabric8-services/fabric8-cluster/controller"
	"github.com/fabric8-services/fabric8-cluster/migration"
	"github.com/fabric8-services/fabric8-cluster/sentry"
	"github.com/fabric8-services/fabric8-common/goamiddleware"
	"github.com/fabric8-services/fabric8-common/log"
	"github.com/fabric8-services/fabric8-common/token"

	"context"
	"github.com/fabric8-services/fabric8-cluster/gormapplication"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/logging/logrus"
	"github.com/goadesign/goa/middleware"
	"github.com/goadesign/goa/middleware/gzip"
	"github.com/goadesign/goa/middleware/security/jwt"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	// --------------------------------------------------------------------
	// Parse flags
	// --------------------------------------------------------------------
	var configFile string
	var serviceAccountConfigFile string
	var osoClusterConfigFile string
	var printConfig bool
	var migrateDB bool
	flag.StringVar(&configFile, "config", "", "Path to the config file to read")
	flag.StringVar(&serviceAccountConfigFile, "serviceAccountConfig", "", "Path to the service account configuration file")
	flag.StringVar(&osoClusterConfigFile, "osoClusterConfigFile", "", "Path to the OSO cluster configuration file")
	flag.BoolVar(&printConfig, "printConfig", false, "Prints the config (including merged environment variables) and exits")
	flag.BoolVar(&migrateDB, "migrateDatabase", false, "Migrates the database to the newest version and exits.")
	flag.Parse()

	// Override default -config switch with environment variable only if -config switch was
	// not explicitly given via the command line.
	configFile = configFileFromFlags("config", "F8_CONFIG_FILE_PATH")
	osoClusterConfigFile = configFileFromFlags("osoClusterConfigFile", "F8_OSO_CLUSTER_CONFIG_FILE")

	config, err := configuration.NewConfigurationData(configFile, osoClusterConfigFile)
	if err != nil {
		log.Panic(nil, map[string]interface{}{
			"config_file":                 configFile,
			"service_account_config_file": serviceAccountConfigFile,
			"oso_cluster_config_file":     osoClusterConfigFile,
			"err": err,
		}, "failed to setup the configuration")
	}

	if printConfig {
		os.Exit(0)
	}

	// Initialized developer mode flag and log level for the logger
	log.InitializeLogger(config.IsLogJSON(), config.GetLogLevel())

	printUserInfo()

	var db *gorm.DB
	for {
		db, err = gorm.Open("postgres", config.GetPostgresConfigString())
		if err != nil {
			log.Logger().Errorf("ERROR: Unable to open connection to database %v", err)
			log.Logger().Infof("Retrying to connect in %v...", config.GetPostgresConnectionRetrySleep())
			if db != nil {
				db.Close()
			}
			time.Sleep(config.GetPostgresConnectionRetrySleep())
		} else {
			defer db.Close()
			break
		}
	}

	// Initialize sentry client
	haltSentry, err := sentry.Initialize(config, controller.Commit)
	if err != nil {
		log.Panic(nil, map[string]interface{}{
			"err": err,
		}, "failed to setup the sentry client")
	}
	defer haltSentry()

	// Initialize cluster config watcher
	haltWatcher, err := config.InitializeClusterWatcher()
	if err != nil {
		log.Panic(nil, map[string]interface{}{
			"err": err,
		}, "failed to setup the cluster config watcher")
	}
	defer haltWatcher()

	if config.DeveloperModeEnabled() && log.IsDebug() {
		db = db.Debug()
	}

	if config.GetPostgresConnectionMaxIdle() > 0 {
		log.Logger().Infof("Configured connection pool max idle %v", config.GetPostgresConnectionMaxIdle())
		db.DB().SetMaxIdleConns(config.GetPostgresConnectionMaxIdle())
	}
	if config.GetPostgresConnectionMaxOpen() > 0 {
		log.Logger().Infof("Configured connection pool max open %v", config.GetPostgresConnectionMaxOpen())
		db.DB().SetMaxOpenConns(config.GetPostgresConnectionMaxOpen())
	}

	// Set the database transaction timeout
	transaction.SetDatabaseTransactionTimeout(config.GetPostgresTransactionTimeout())

	// Migrate the schema
	err = migration.Migrate(db.DB(), config.GetPostgresDatabase())
	if err != nil {
		log.Panic(nil, map[string]interface{}{
			"err": err,
		}, "failed migration")
	}

	// Nothing to here except exit, since the migration is already performed.
	if migrateDB {
		os.Exit(0)
	}

	// Create service
	service := goa.New("cluster")

	// Mount middleware
	service.Use(middleware.RequestID())
	// Use our own log request to inject identity id and modify other properties
	service.Use(log.LogRequest(config.DeveloperModeEnabled()))
	service.Use(gzip.Middleware(9))
	service.Use(app.ErrorHandler(service, true))
	service.Use(middleware.Recover())

	service.WithLogger(goalogrus.New(log.Logger()))

	// Create DB
	appDB := gormapplication.NewGormDB(db, config)

	// Setup Security
	tokenManager, err := token.DefaultManager(config)
	if err != nil {
		log.Panic(nil, map[string]interface{}{
			"err": err,
		}, "failed to create token manager")
	}
	// Middleware that extracts and stores the token in the context
	jwtMiddlewareTokenContext := goamiddleware.TokenContext(tokenManager, app.NewJWTSecurity())
	service.Use(jwtMiddlewareTokenContext)

	service.Use(token.InjectTokenManager(tokenManager))
	service.Use(log.LogRequest(config.DeveloperModeEnabled()))
	app.UseJWTMiddleware(service, jwt.New(tokenManager.PublicKeys(), nil, app.NewJWTSecurity()))

	// Mount "status" controller
	statusCtrl := controller.NewStatusController(service, controller.NewGormDBChecker(db), config)
	app.MountStatusController(service, statusCtrl)

	// Mount "clusters" controller
	appDB.Clusters().CreateOrSaveOSOClusterFromConfig(context.Background(), config)
	clustersCtrl := controller.NewClustersController(service, config)
	app.MountClustersController(service, clustersCtrl)

	log.Logger().Infoln("Git Commit SHA: ", controller.Commit)
	log.Logger().Infoln("UTC Build Time: ", controller.BuildTime)
	log.Logger().Infoln("UTC Start Time: ", controller.StartTime)
	log.Logger().Infoln("Dev mode:       ", config.DeveloperModeEnabled())
	log.Logger().Infoln("GOMAXPROCS:     ", runtime.GOMAXPROCS(-1))
	log.Logger().Infoln("NumCPU:         ", runtime.NumCPU())

	http.Handle("/api/", service.Mux)
	http.Handle("/favicon.ico", http.NotFoundHandler())

	// Start/mount metrics http
	if config.GetHTTPAddress() == config.GetMetricsHTTPAddress() {
		http.Handle("/metrics", prometheus.Handler())
	} else {
		go func(metricAddress string) {
			mx := http.NewServeMux()
			mx.Handle("/metrics", prometheus.Handler())
			if err := http.ListenAndServe(metricAddress, mx); err != nil {
				log.Error(nil, map[string]interface{}{
					"addr": metricAddress,
					"err":  err,
				}, "unable to connect to metrics server")
				service.LogError("startup", "err", err)
			}
		}(config.GetMetricsHTTPAddress())
	}

	// Start http
	if err := http.ListenAndServe(config.GetHTTPAddress(), nil); err != nil {
		log.Error(nil, map[string]interface{}{
			"addr": config.GetHTTPAddress(),
			"err":  err,
		}, "unable to connect to server")
		service.LogError("startup", "err", err)
	}
}

func configFileFromFlags(flagName string, envVarName string) string {
	configSwitchIsSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == flagName {
			configSwitchIsSet = true
		}
	})
	if !configSwitchIsSet {
		if envConfigPath, ok := os.LookupEnv(envVarName); ok {
			return envConfigPath
		}
	}
	return ""
}

func printUserInfo() {
	u, err := user.Current()
	if err != nil {
		log.Warn(nil, map[string]interface{}{
			"err": err,
		}, "failed to get current user")
	} else {
		log.Info(nil, map[string]interface{}{
			"username": u.Username,
			"uuid":     u.Uid,
		}, "Running as user name '%s' with UID %s.", u.Username, u.Uid)
		g, err := user.LookupGroupId(u.Gid)
		if err != nil {
			log.Warn(nil, map[string]interface{}{
				"err": err,
			}, "failed to lookup group")
		} else {
			log.Info(nil, map[string]interface{}{
				"groupname": g.Name,
				"gid":       g.Gid,
			}, "Running as as group '%s' with GID %s.", g.Name, g.Gid)
		}
	}
}
