package transaction_test

import (
	"github.com/fabric8-services/fabric8-cluster/application/transaction"
	"github.com/fabric8-services/fabric8-cluster/gormtestsupport"
	"github.com/fabric8-services/fabric8-cluster/resource"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestTransaction struct {
	gormtestsupport.DBTestSuite
}

func TestRunTransaction(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestTransaction{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (test *TestTransaction) TestTransactionOK() {
	err := transaction.Transactional(test.Application, func(tr transaction.TransactionalResources) error {
		return nil
	})
	require.NoError(test.T(), err)
}

func (test *TestTransaction) TestTransactionFail() {
	err := transaction.Transactional(test.Application, func(tr transaction.TransactionalResources) error {
		return errors.New("Oopsie Woopsie")
	})
	// then
	require.Error(test.T(), err)
}
