package sentry

import (
	"context"
	"fmt"

	"github.com/fabric8-services/fabric8-common/auth"
	"github.com/fabric8-services/fabric8-common/sentry"

	raven "github.com/getsentry/raven-go"
	"github.com/goadesign/goa/middleware/security/jwt"
)

// SentryConfiguration the configuration for Sentry
type Configuration interface {
	GetSentryDSN() string
	GetEnvironment() string
}

// Initialize initializes sentry client
func Initialize(config Configuration, commit string) (func(), error) {
	sentryDSN := config.GetSentryDSN()

	return sentry.InitializeSentryClient(
		&sentryDSN,
		sentry.WithRelease(commit),
		sentry.WithEnvironment(config.GetEnvironment()),
		sentry.WithUser(extractUserInfo()))
}

func extractUserInfo() func(ctx context.Context) (*raven.User, error) {
	return func(ctx context.Context) (*raven.User, error) {
		m, err := auth.ReadManagerFromContext(ctx)
		if err != nil {
			return nil, err
		}

		token := jwt.ContextJWT(ctx)
		if token == nil {
			return nil, fmt.Errorf("no token found in context")
		}
		t, err := m.ParseToken(ctx, token.Raw)
		if err != nil {
			return nil, err
		}

		return &raven.User{
			Username: t.Username,
			Email:    t.Email,
			ID:       t.Subject,
		}, nil
	}
}
