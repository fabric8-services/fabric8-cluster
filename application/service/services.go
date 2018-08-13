package service

import (
	"context"

	"github.com/satori/go.uuid"
)

/*
Steps for adding a new Service:
1. Add a new service interface to application/service/services.go
2. Create an implementation of the service interface
3. Add a new method to service.Services interface in application/service/services.go for accessing the service interface
   defined in step 1
4. Add a new method to application/service/factory/service_factory.go which implements the service access method
   from step #3 and uses the service constructor from step 2
5. Add a new method to gormapplication/application.go which implements the service access method from step #3
   and use the factory method from the step #4
*/

type InvitationService interface {
	// Issue creates a new invitation for a user.
	Issue(ctx context.Context, issuingUserID uuid.UUID, inviteTo string, invitations []invitation.Invitation) error
	// Rescind revokes an invitation for a user.
	Rescind(ctx context.Context, rescindingUserID, invitationID uuid.UUID) error
	// Accept processes the invitation acceptance action from the user, converting the invitation into real memberships/roles
	Accept(ctx context.Context, currentIdentityID uuid.UUID, token uuid.UUID) (string, error)
}

type FooService interface {
	CreateOrganization(ctx context.Context, creatorIdentityID uuid.UUID, organizationName string) (*uuid.UUID, error)
}

//Services creates instances of service layer objects
type Services interface {
	FooService() FooService
}
