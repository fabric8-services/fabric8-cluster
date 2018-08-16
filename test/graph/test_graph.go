package graph

import (
	"context"
	"testing"

	"github.com/fabric8-services/fabric8-cluster/application"

	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

// TestGraph manages an object graph of domain objects for the purposes of testing
type TestGraph struct {
	t          *testing.T
	app        application.Application
	ctx        context.Context
	references map[string]interface{}
	db         *gorm.DB
}

// baseWrapper is the base struct for other Wrapper structs
type baseWrapper struct {
	graph *TestGraph
}

func (w *baseWrapper) identityIDFromWrapper(wrapper interface{}) uuid.UUID {
	switch t := wrapper.(type) {
	case *identityWrapper:
		return t.identity.ID
	}
	require.True(w.graph.t, false, "wrapper must be either user wrapper or identity wrapper")
	return uuid.UUID{}
}

// Identifier is used to explicitly set the unique identifier for a graph object
type Identifier struct {
	value string
}

// NewTestGraph creates a new test graph
func NewTestGraph(t *testing.T, app application.Application, ctx context.Context, db *gorm.DB) TestGraph {
	return TestGraph{t: t, app: app, ctx: ctx, references: make(map[string]interface{}), db: db}
}

// register registers a new wrapper object with the test graph's internal list of objects
func (g *TestGraph) register(id string, wrapper interface{}) {
	if _, found := g.references[id]; found {
		require.True(g.t, false, "object identifier '%s' already registered", id)
	} else {
		g.references[id] = wrapper
	}
}

func (g *TestGraph) generateIdentifier(params []interface{}) string {
	for i := range params {
		switch t := params[i].(type) {
		case Identifier:
			return t.value
		}
	}
	return uuid.NewV4().String()
}

type wrapperConstructor func(g *TestGraph, params []interface{}) interface{}

func (g *TestGraph) createAndRegister(constructor wrapperConstructor, params []interface{}) interface{} {
	wrapper := constructor(g, params)
	g.register(g.generateIdentifier(params), wrapper)
	return wrapper
}

func (g *TestGraph) ID(value string) Identifier {
	return Identifier{value}
}

func (g *TestGraph) CreateIdentity(params ...interface{}) *identityWrapper {
	return g.createAndRegister(newIdentityWrapper, params).(*identityWrapper)
}
