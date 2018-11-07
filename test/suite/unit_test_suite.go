package suite

import (
	"github.com/fabric8-services/fabric8-cluster/configuration"
	"github.com/fabric8-services/fabric8-common/log"
	"github.com/fabric8-services/fabric8-common/test/suite"
)

// NewUnitTestSuite instantiates a new UnitTestSuite
func NewUnitTestSuite() UnitTestSuite {
	return UnitTestSuite{}
}

// RemoteTestSuite is a base for unit tests
type UnitTestSuite struct {
	suite.UnitTestSuite
	Config *configuration.ConfigurationData
}

// SetupSuite implements suite.SetupAllSuite
func (s *UnitTestSuite) SetupSuite() {
	s.UnitTestSuite.SetupSuite()
	s.setupConfig()
}

func (s *UnitTestSuite) setupConfig() {
	config, err := configuration.GetConfigurationData()
	if err != nil {
		log.Panic(nil, map[string]interface{}{
			"err": err,
		}, "failed to setup the configuration")
	}
	s.Config = config
}

// TearDownSuite implements suite.TearDownAllSuite
func (s *UnitTestSuite) TearDownSuite() {
	s.Config = nil // Summon the GC!
}
