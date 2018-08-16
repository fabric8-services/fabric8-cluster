package graph

import (
	"github.com/fabric8-services/fabric8-cluster/test"

	"github.com/satori/go.uuid"
)

// identityWrapper represents a user domain object
type identityWrapper struct {
	baseWrapper
	identity *test.Identity
}

func newIdentityWrapper(g *TestGraph, params []interface{}) interface{} {
	w := identityWrapper{baseWrapper: baseWrapper{g}}

	w.identity = &test.Identity{
		Username: "TestUserIdentity-" + uuid.NewV4().String(),
		ID:       uuid.NewV4(),
	}

	return &w
}

func (w *identityWrapper) Identity() *test.Identity {
	return w.identity
}
