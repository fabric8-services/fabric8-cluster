package sentry

import (
	"context"
	"testing"

	testsuite "github.com/fabric8-services/fabric8-cluster/test/suite"
	"github.com/fabric8-services/fabric8-common/test/auth"
	"github.com/fabric8-services/fabric8-common/token"

	"github.com/dgrijalva/jwt-go"
	"github.com/getsentry/raven-go"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestSentry(t *testing.T) {
	suite.Run(t, &TestWhiteboxSentry{})
}

type TestWhiteboxSentry struct {
	testsuite.UnitTestSuite
}

func failOnNoToken(t *testing.T) context.Context {
	return token.ContextWithTokenManager(context.Background(), auth.TokenManager)
}

func failOnParsingToken(t *testing.T) context.Context {
	ctx := failOnNoToken(t)
	// Here we add a token which is incomplete
	token := jwt.New(jwt.GetSigningMethod("RS256"))
	ctx = goajwt.WithJWT(ctx, token)
	return ctx
}

func (s *TestWhiteboxSentry) TestExtractUserInfo() {
	f := extractUserInfo()
	ctx, identity, err := auth.EmbedUserTokenInContext(nil, nil)
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

func (s *TestWhiteboxSentry) TestInitialize() {
	haltSentry, err := Initialize(s.Config, "")
	require.NoError(s.T(), err)
	require.NotNil(s.T(), haltSentry)
	require.NotPanics(s.T(), func() {
		haltSentry()
	})
}
