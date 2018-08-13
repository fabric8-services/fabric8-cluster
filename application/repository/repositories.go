package repository

import (
	account "github.com/fabric8-services/fabric8-cluster/account/repository"
	"github.com/fabric8-services/fabric8-cluster/auth"
	invitation "github.com/fabric8-services/fabric8-cluster/authorization/invitation/repository"
	resource "github.com/fabric8-services/fabric8-cluster/authorization/resource/repository"
	resourcetype "github.com/fabric8-services/fabric8-cluster/authorization/resourcetype/repository"
	role "github.com/fabric8-services/fabric8-cluster/authorization/role/repository"
	token "github.com/fabric8-services/fabric8-cluster/authorization/token/repository"
	"github.com/fabric8-services/fabric8-cluster/token/provider"
)

//Repositories stands for a particular implementation of the business logic of our application
type Repositories interface {
	Identities() account.IdentityRepository
	Users() account.UserRepository
	OauthStates() auth.OauthStateReferenceRepository
	ExternalTokens() provider.ExternalTokenRepository
	VerificationCodes() account.VerificationCodeRepository
	InvitationRepository() invitation.InvitationRepository
	ResourceRepository() resource.ResourceRepository
	ResourceTypeRepository() resourcetype.ResourceTypeRepository
	ResourceTypeScopeRepository() resourcetype.ResourceTypeScopeRepository
	IdentityRoleRepository() role.IdentityRoleRepository
	RoleRepository() role.RoleRepository
	DefaultRoleMappingRepository() role.DefaultRoleMappingRepository
	RoleMappingRepository() role.RoleMappingRepository
	TokenRepository() token.TokenRepository
}
