package gormtestsupport

import (
	"github.com/fabric8-services/fabric8-cluster/application"
	"github.com/fabric8-services/fabric8-cluster/configuration"
	"github.com/fabric8-services/fabric8-cluster/gormapplication"
	"github.com/fabric8-services/fabric8-common/log"
	testsuite "github.com/fabric8-services/fabric8-common/test/suite"
)

// DBTestSuite is a base for tests using a gorm db
type DBTestSuite struct {
	testsuite.DBTestSuite
	Configuration *configuration.ConfigurationData
	Application   application.Application
}

// NewDBTestSuite instantiates a new DBTestSuite
func NewDBTestSuite() DBTestSuite {
	config, err := configuration.GetConfigurationData()
	if err != nil {
		log.Panic(nil, map[string]interface{}{
			"err": err,
		}, "failed to setup the configuration")
	}
	return DBTestSuite{DBTestSuite: testsuite.NewDBTestSuite(config), Configuration: config}
}

// SetupSuite implements suite.SetupAllSuite
func (s *DBTestSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.Application = gormapplication.NewGormDB(s.DB, s.Configuration)
}
