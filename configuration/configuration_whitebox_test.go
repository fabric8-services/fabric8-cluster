package configuration

import (
	"os"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-common/resource"
	testsuite "github.com/fabric8-services/fabric8-common/test/suite"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestConfiguration(t *testing.T) {
	suite.Run(t, &ConfigurationWhiteboxTestSuite{})
}

type ConfigurationWhiteboxTestSuite struct {
	testsuite.UnitTestSuite
	config *ConfigurationData
}

func (s *ConfigurationWhiteboxTestSuite) SetupTest() {
	resource.Require(s.T(), resource.UnitTest)
	config, err := GetConfigurationData()
	require.NoError(s.T(), err)
	s.config = config
}

func (s *ConfigurationWhiteboxTestSuite) TestGetLogLevelOK() {
	key := "F8_LOG_LEVEL"
	realEnvValue := os.Getenv(key)

	os.Unsetenv(key)
	defer func() {
		os.Setenv(key, realEnvValue)
	}()

	assert.Equal(s.T(), defaultLogLevel, s.config.GetLogLevel())

	os.Setenv(key, "warning")
	assert.Equal(s.T(), "warning", s.config.GetLogLevel())
}

func (s *ConfigurationWhiteboxTestSuite) TestGetTransactionTimeoutOK() {
	key := "F8_POSTGRES_TRANSACTION_TIMEOUT"
	realEnvValue := os.Getenv(key)

	os.Unsetenv(key)
	defer func() {
		os.Setenv(key, realEnvValue)
	}()

	assert.Equal(s.T(), time.Duration(5*time.Minute), s.config.GetPostgresTransactionTimeout())

	os.Setenv(key, "6m")

	assert.Equal(s.T(), time.Duration(6*time.Minute), s.config.GetPostgresTransactionTimeout())
}
