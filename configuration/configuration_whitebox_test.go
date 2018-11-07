package configuration

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-common/resource"

	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
)

var reqLong *goa.RequestData
var reqShort *goa.RequestData
var config *ConfigurationData

func init() {

	// ensure that the content here is executed only once.
	reqLong = &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}
	reqShort = &goa.RequestData{
		Request: &http.Request{Host: "api.domain.org"},
	}
	resetConfiguration()
}

func resetConfiguration() {
	var err error
	config, err = GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
}

func TestGetLogLevelOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	key := "F8_LOG_LEVEL"
	realEnvValue := os.Getenv(key)

	os.Unsetenv(key)
	defer func() {
		os.Setenv(key, realEnvValue)
		resetConfiguration()
	}()

	assert.Equal(t, defaultLogLevel, config.GetLogLevel())

	os.Setenv(key, "warning")
	resetConfiguration()

	assert.Equal(t, "warning", config.GetLogLevel())
}

func TestGetTransactionTimeoutOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	key := "F8_POSTGRES_TRANSACTION_TIMEOUT"
	realEnvValue := os.Getenv(key)

	os.Unsetenv(key)
	defer func() {
		os.Setenv(key, realEnvValue)
		resetConfiguration()
	}()

	assert.Equal(t, time.Duration(5*time.Minute), config.GetPostgresTransactionTimeout())

	os.Setenv(key, "6m")
	resetConfiguration()

	assert.Equal(t, time.Duration(6*time.Minute), config.GetPostgresTransactionTimeout())
}
