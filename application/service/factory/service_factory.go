package factory

import (
	"fmt"
	"time"

	"github.com/fabric8-services/fabric8-cluster/application/repository"
	"github.com/fabric8-services/fabric8-cluster/application/service"
	"github.com/fabric8-services/fabric8-cluster/application/service/context"
	"github.com/fabric8-services/fabric8-cluster/application/transaction"
	clusterservice "github.com/fabric8-services/fabric8-cluster/cluster/service"
	"github.com/fabric8-services/fabric8-cluster/configuration"
	"github.com/fabric8-services/fabric8-common/log"

	"github.com/pkg/errors"
)

type serviceContextImpl struct {
	repositories              repository.Repositories
	transactionalRepositories repository.Repositories
	transactionManager        transaction.TransactionManager
	inTransaction             bool
	services                  service.Services
}

func NewServiceContext(repos repository.Repositories, tm transaction.TransactionManager, config *configuration.ConfigurationData, options ...Option) context.ServiceContext {
	ctx := new(serviceContextImpl)
	ctx.repositories = repos
	ctx.transactionManager = tm
	ctx.inTransaction = false

	sc := ctx
	ctx.services = NewServiceFactory(func() context.ServiceContext { return sc }, config, options...)
	return ctx
}

func (s *serviceContextImpl) Repositories() repository.Repositories {
	if s.inTransaction {
		return s.transactionalRepositories
	}
	return s.repositories
}

func (s *serviceContextImpl) Services() service.Services {
	return s.services
}

func (s *serviceContextImpl) ExecuteInTransaction(todo func() error) error {
	if !s.inTransaction {
		// If we are not in a transaction already, start a new transaction
		var tx transaction.Transaction
		var err error
		if tx, err = s.transactionManager.BeginTransaction(); err != nil {
			log.Error(nil, map[string]interface{}{
				"err": err,
			}, "database BeginTransaction failed!")

			return errors.WithStack(err)
		}

		// Set the transaction flag to true
		s.inTransaction = true

		// Set the transactional repositories property
		s.transactionalRepositories = tx.(repository.Repositories)

		defer s.endTransaction()

		return func() error {
			errorChan := make(chan error, 1)
			txTimeout := time.After(transaction.DatabaseTransactionTimeout())

			go func() {
				defer func() {
					if err := recover(); err != nil {
						errorChan <- errors.New(fmt.Sprintf("Unknown error: %v", err))
					}
				}()
				errorChan <- todo()
			}()

			select {
			case err := <-errorChan:
				if err != nil {
					log.Debug(nil, nil, "Rolling back the transaction...")
					tx.Rollback()
					log.Error(nil, map[string]interface{}{
						"err": err,
					}, "database transaction failed!")
					return errors.WithStack(err)
				}

				tx.Commit()
				log.Debug(nil, nil, "Commit the transaction!")
				return nil
			case <-txTimeout:
				log.Debug(nil, nil, "Rolling back the transaction...")
				tx.Rollback()
				log.Error(nil, nil, "database transaction timeout!")
				return errors.New("database transaction timeout!")
			}
		}()
	} else {
		// If we are in a transaction, simply execute the passed function
		return todo()
	}
}

func (s *serviceContextImpl) endTransaction() {
	s.inTransaction = false
}

// ServiceContextProducer the service factory producer function
type ServiceContextProducer func() context.ServiceContext

// ServiceFactory the service factory
type ServiceFactory struct {
	contextProducer ServiceContextProducer
	config          *configuration.ConfigurationData
}

// NewServiceFactory initializes a new factory with some options to use alternative implementation of the underlying services
func NewServiceFactory(producer ServiceContextProducer, config *configuration.ConfigurationData, options ...Option) *ServiceFactory {
	f := &ServiceFactory{contextProducer: producer, config: config}
	log.Info(nil, map[string]interface{}{}, "configuring a new service factory with %d options", len(options))
	// and options
	for _, opt := range options {
		opt(f)
	}
	return f
}

// Option an option to configure the Service Factory
type Option func(f *ServiceFactory)

func (f *ServiceFactory) getContext() context.ServiceContext {
	return f.contextProducer()
}

// ClusterService returns a new cluster service implementation
func (f *ServiceFactory) ClusterService() service.ClusterService {
	return clusterservice.NewClusterService(f.getContext(), f.config)
}
