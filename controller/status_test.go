package controller_test

import (
	"os"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-cluster/app/test"
	"github.com/fabric8-services/fabric8-cluster/configuration"
	. "github.com/fabric8-services/fabric8-cluster/controller"
	"github.com/fabric8-services/fabric8-cluster/gormtestsupport"

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

type StatusControllerTestSuite struct {
	gormtestsupport.DBTestSuite
}

func TestStatusController(t *testing.T) {
	suite.Run(t, &StatusControllerTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *StatusControllerTestSuite) UnSecuredController() (*goa.Service, *StatusController) {
	svc := goa.New("Status-Service")
	return svc, NewStatusController(svc, NewGormDBChecker(s.DB), s.Configuration)
}

func (s *StatusControllerTestSuite) UnSecuredControllerWithUnreachableDB() (*goa.Service, *StatusController) {
	svc := goa.New("Status-Service")
	return svc, NewStatusController(svc, &dummyDBChecker{}, s.Configuration)
}

func (s *StatusControllerTestSuite) TestShowStatusInDevModeOK() {
	t := s.T()
	svc, ctrl := s.UnSecuredController()
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

func (s *StatusControllerTestSuite) TestShowStatusWithoutDBFails() {
	svc, ctrl := s.UnSecuredControllerWithUnreachableDB()
	_, res := test.ShowStatusServiceUnavailable(s.T(), svc.Context, svc, ctrl)

	assert.Equal(s.T(), "Error: DB is unreachable", res.DatabaseStatus)
}

func (s *StatusControllerTestSuite) TestShowStatusWithDefaultConfigInProdModeFails() {
	existingDevMode := os.Getenv("F8_DEVELOPER_MODE_ENABLED")
	defer func() {
		os.Setenv("F8_DEVELOPER_MODE_ENABLED", existingDevMode)
		s.resetConfiguration()
	}()

	os.Setenv("F8_DEVELOPER_MODE_ENABLED", "false")
	s.resetConfiguration()
	svc, ctrl := s.UnSecuredController()
	_, res := test.ShowStatusServiceUnavailable(s.T(), svc.Context, svc, ctrl)
	assert.Equal(s.T(), expectedDefaultConfProdModeErrorMessage, res.ConfigurationStatus)
	assert.Equal(s.T(), "OK", res.DatabaseStatus)

	// If the DB is not available then status should return the corresponding error
	svc, ctrl = s.UnSecuredControllerWithUnreachableDB()
	_, res = test.ShowStatusServiceUnavailable(s.T(), svc.Context, svc, ctrl)

	assert.Equal(s.T(), expectedDefaultConfProdModeErrorMessage, res.ConfigurationStatus)
	assert.Equal(s.T(), "Error: DB is unreachable", res.DatabaseStatus)
}

func (s *StatusControllerTestSuite) resetConfiguration() {
	config, err := configuration.GetConfigurationData()
	require.Nil(s.T(), err)
	s.Configuration = config
}

type dummyDBChecker struct {
}

func (c *dummyDBChecker) Ping() error {
	return errors.New("DB is unreachable")
}
