package sentry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/fabric8-services/fabric8-cluster/configuration"

	"github.com/fabric8-services/fabric8-common/auth"
	testauth "github.com/fabric8-services/fabric8-common/test/auth"
	testsuite "github.com/fabric8-services/fabric8-common/test/suite"

	jwt "github.com/dgrijalva/jwt-go"
	raven "github.com/getsentry/raven-go"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/stretchr/testify/require"
)

func TestSentry(t *testing.T) {
	suite.Run(t, &SentryWhiteboxTestSuite{})
}

type SentryWhiteboxTestSuite struct {
	testsuite.UnitTestSuite
	config *configuration.ConfigurationData
}

func (s *SentryWhiteboxTestSuite) SetupSuite() {
	config, err := configuration.GetConfigurationData()
	require.NoError(s.T(), err)
	s.config = config
}

func failOnNoToken(t *testing.T) context.Context {
	return auth.ContextWithTokenManager(context.Background(), testauth.TokenManager)
}

func failOnParsingToken(t *testing.T) context.Context {
	ctx := failOnNoToken(t)
	// Here we add a token which is incomplete
	token := jwt.New(jwt.GetSigningMethod("RS256"))
	ctx = goajwt.WithJWT(ctx, token)
	return ctx
}

func (s *SentryWhiteboxTestSuite) TestExtractUserInfo() {
	f := extractUserInfo()
	ctx, identity, err := testauth.EmbedUserTokenInContext(nil, nil)
	require.NoError(s.T(), err)

	tests := []struct {
		name    string
		ctx     context.Context
		want    *raven.User
		wantErr bool
	}{
		{
			name:    "Given some random context",
			ctx:     context.Background(),
			wantErr: true,
		},
		{
			name:    "fail on no token",
			ctx:     failOnNoToken(s.T()),
			wantErr: true,
		},
		{
			name:    "fail on parsing token",
			ctx:     failOnParsingToken(s.T()),
			wantErr: true,
		},
		{
			name:    "pass on parsing token",
			ctx:     ctx,
			wantErr: false,
			want: &raven.User{
				Username: identity.Username,
				ID:       identity.ID.String(),
				Email:    identity.Email,
			},
		},
	}
	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			got, err := f(tt.ctx)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equalf(t, tt.want, got, "extractUserInfo() = %v, want %v", got, tt.want)
		})
	}
}

func (s *SentryWhiteboxTestSuite) TestInitialize() {
	haltSentry, err := Initialize(s.config, "")
	require.NoError(s.T(), err)
	require.NotNil(s.T(), haltSentry)
	require.NotPanics(s.T(), func() {
		haltSentry()
	})
}
