package controller_test

import (
	"os"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-cluster/app/test"
	"github.com/fabric8-services/fabric8-cluster/configuration"
	. "github.com/fabric8-services/fabric8-cluster/controller"
	"github.com/fabric8-services/fabric8-cluster/gormtestsupport"
	"github.com/fabric8-services/fabric8-cluster/resource"

	"github.com/goadesign/goa"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	expectedDefaultConfDevModeErrorMessage  = "Error: /etc/fabric8/oso-clusters.conf is not used; developer Mode is enabled; default DB password is used; environment is expected to be set to 'production' or 'prod-preview'; Sentry DSN is empty"
	expectedDefaultConfProdModeErrorMessage = "Error: /etc/fabric8/oso-clusters.conf is not used; default DB password is used; environment is expected to be set to 'production' or 'prod-preview'; Auth service url is empty; Sentry DSN is empty"
)

type TestStatusREST struct {
	gormtestsupport.DBTestSuite
}

func TestRunStatusREST(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestStatusREST{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (rest *TestStatusREST) UnSecuredController() (*goa.Service, *StatusController) {
	svc := goa.New("Status-Service")
	return svc, NewStatusController(svc, NewGormDBChecker(rest.DB), rest.Configuration)
}

func (rest *TestStatusREST) UnSecuredControllerWithUnreachableDB() (*goa.Service, *StatusController) {
	svc := goa.New("Status-Service")
	return svc, NewStatusController(svc, &dummyDBChecker{}, rest.Configuration)
}

func (rest *TestStatusREST) TestShowStatusInDevModeOK() {
	t := rest.T()
	svc, ctrl := rest.UnSecuredController()
	_, res := test.ShowStatusOK(t, svc.Context, svc, ctrl)

	assert.Equal(t, "0", res.Commit, "Commit not found")
	assert.Equal(t, StartTime, res.StartTime, "StartTime is not correct")
	assert.Equal(t, expectedDefaultConfDevModeErrorMessage, res.ConfigurationStatus)
	assert.Equal(t, "OK", res.DatabaseStatus)

	_, err := time.Parse("2006-01-02T15:04:05Z", res.StartTime)
	assert.Nil(t, err, "Incorrect layout of StartTime")

	require.NotNil(t, res.DevMode)
	assert.True(t, *res.DevMode)
}

func (rest *TestStatusREST) TestShowStatusWithoutDBFails() {
	svc, ctrl := rest.UnSecuredControllerWithUnreachableDB()
	_, res := test.ShowStatusServiceUnavailable(rest.T(), svc.Context, svc, ctrl)

	assert.Equal(rest.T(), "Error: DB is unreachable", res.DatabaseStatus)
}

func (rest *TestStatusREST) TestShowStatusWithDefaultConfigInProdModeFails() {
	existingDevMode := os.Getenv("F8CLUSTER_DEVELOPER_MODE_ENABLED")
	defer func() {
		os.Setenv("F8CLUSTER_DEVELOPER_MODE_ENABLED", existingDevMode)
		rest.resetConfiguration()
	}()

	os.Setenv("F8CLUSTER_DEVELOPER_MODE_ENABLED", "false")
	rest.resetConfiguration()
	svc, ctrl := rest.UnSecuredController()
	_, res := test.ShowStatusServiceUnavailable(rest.T(), svc.Context, svc, ctrl)
	assert.Equal(rest.T(), expectedDefaultConfProdModeErrorMessage, res.ConfigurationStatus)
	assert.Equal(rest.T(), "OK", res.DatabaseStatus)

	// If the DB is not available then status should return the corresponding error
	svc, ctrl = rest.UnSecuredControllerWithUnreachableDB()
	_, res = test.ShowStatusServiceUnavailable(rest.T(), svc.Context, svc, ctrl)

	assert.Equal(rest.T(), expectedDefaultConfProdModeErrorMessage, res.ConfigurationStatus)
	assert.Equal(rest.T(), "Error: DB is unreachable", res.DatabaseStatus)
}

func (rest *TestStatusREST) resetConfiguration() {
	config, err := configuration.GetConfigurationData()
	require.Nil(rest.T(), err)
	rest.Configuration = config
}

type dummyDBChecker struct {
}

func (c *dummyDBChecker) Ping() error {
	return errors.New("DB is unreachable")
}
