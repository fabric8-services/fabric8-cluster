package transaction_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-cluster/application/transaction"
	"github.com/fabric8-services/fabric8-cluster/gormtestsupport"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TransactionTestSuite struct {
	gormtestsupport.DBTestSuite
}

func TestTransaction(t *testing.T) {
	suite.Run(t, &TransactionTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *TransactionTestSuite) TestTransactionOK() {
	err := transaction.Transactional(s.Application, func(tr transaction.TransactionalResources) error {
		return nil
	})
	require.NoError(s.T(), err)
}

func (s *TransactionTestSuite) TestTransactionFail() {
	err := transaction.Transactional(s.Application, func(tr transaction.TransactionalResources) error {
		return errors.New("Oopsie Woopsie")
	})
	require.Error(s.T(), err)
}
