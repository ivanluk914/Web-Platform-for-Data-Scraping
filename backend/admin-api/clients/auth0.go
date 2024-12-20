package clients

import (
	"context"
	"net/http"
	"net/url"

	"admin-api/config"
	apperrors "admin-api/errors"
	"admin-api/models"

	"github.com/auth0/go-auth0"
	"github.com/auth0/go-auth0/authentication"
	"github.com/auth0/go-auth0/management"
	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/pkg/errors"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
)

var (
	Auth0AdminRole = &management.Role{
		ID:   auth0.String("rol_9wVRSPWcCNB3AypM"),
		Name: auth0.String("Admin"),
	}
	Auth0MemberRole = &management.Role{
		ID:   auth0.String("rol_ojPUsNcwlWeofPmS"),
		Name: auth0.String("Member"),
	}
	Auth0UserRole = &management.Role{
		ID:   auth0.String("rol_wgtsNMZVvH6xhrnu"),
		Name: auth0.String("User"),
	}
)

type authClient struct {
	logger         *otelzap.Logger
	authentication *authentication.Authentication
	management     *management.Management
}

type AuthClient interface {
	GetUserFromContext(ctx context.Context) (*models.User, error)
	ListUsers(ctx context.Context, page int64, pageSize int64) ([]*models.User, int64, error)
	ListAllUsers(ctx context.Context) ([]*models.User, error)
	GetUser(ctx context.Context, userID string) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	DeleteUser(ctx context.Context, userID string) error
	ListUserRoles(ctx context.Context, userID string) ([]models.UserRole, error)
	AssignUserRole(ctx context.Context, userID string, role models.UserRole) error
	RemoveUserRole(ctx context.Context, userID string, role models.UserRole) error
}

func NewAuthClient(logger *otelzap.Logger, httpClient *http.Client, cfg config.Auth0Config) (AuthClient, error) {
	ctx := context.Background()

	issuerUrl, err := url.Parse(cfg.Domain)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse issuer url")
	}

	authAPI, err := authentication.New(
		ctx,
		issuerUrl.Hostname(),
		authentication.WithClientID(cfg.ClientID),
		authentication.WithClientSecret(cfg.ClientSecret),
		authentication.WithClient(httpClient),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create authentication client")
	}
	managementAPI, err := management.New(
		issuerUrl.Hostname(),
		management.WithClientCredentials(ctx, cfg.ClientID, cfg.ClientSecret),
		management.WithClient(httpClient),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create management client")
	}
	authClient := &authClient{
		logger:         logger,
		authentication: authAPI,
		management:     managementAPI,
	}
	return authClient, nil
}

func (c *authClient) GetUserFromContext(ctx context.Context) (*models.User, error) {
	ctxValue := ctx.Value(jwtmiddleware.ContextKey{})
	if ctxValue == nil {
		return nil, apperrors.ErrNoAuthContext
	}
	claims, ok := ctxValue.(*validator.ValidatedClaims)
	if !ok {
		return nil, apperrors.ErrInvalidClaims
	}
	return c.GetUser(ctx, claims.RegisteredClaims.Subject)
}

func (c *authClient) ListUsers(ctx context.Context, page int64, pageSize int64) ([]*models.User, int64, error) {
	auth0Users, err := c.management.User.List(ctx, management.Page(int(page)), management.PerPage(int(pageSize)))
	if err != nil {
		return nil, 0, err
	}

	users := make([]*models.User, 0, len(auth0Users.Users))
	for _, user := range auth0Users.Users {
		users = append(users, mapAuth0UserToUser(user, nil))
	}
	return users, int64(auth0Users.Total), nil
}

func (c *authClient) ListAllUsers(ctx context.Context) ([]*models.User, error) {
	var auth0Users []*management.User
	var page int
	for {
		res, err := c.management.User.List(ctx, management.Page(page), management.PerPage(100))
		if err != nil {
			return nil, err
		}
		auth0Users = append(auth0Users, res.Users...)

		if !res.HasNext() {
			break
		}

		page++
	}

	users := make([]*models.User, 0, len(auth0Users))
	for _, user := range auth0Users {
		users = append(users, mapAuth0UserToUser(user, nil))
	}
	return users, nil
}

func (c *authClient) GetUser(ctx context.Context, userID string) (*models.User, error) {
	user, err := c.management.User.Read(ctx, userID)
	if err != nil {
		return nil, err
	}
	userRoles, err := c.ListUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}
	return mapAuth0UserToUser(user, userRoles), nil
}

func (c *authClient) UpdateUser(ctx context.Context, user *models.User) error {
	return c.management.User.Update(ctx, *user.ID, mapUserToAuth0User(user))
}

func (c *authClient) DeleteUser(ctx context.Context, userID string) error {
	return c.management.User.Delete(ctx, userID)
}

func (c *authClient) ListUserRoles(ctx context.Context, userID string) ([]models.UserRole, error) {
	var auth0Roles []*management.Role
	var page int
	for {
		res, err := c.management.User.Roles(ctx, userID, management.Page(page), management.PerPage(100))
		if err != nil {
			return nil, err
		}
		auth0Roles = append(auth0Roles, res.Roles...)

		if !res.HasNext() {
			break
		}

		page++
	}

	roles := make([]models.UserRole, 0, len(auth0Roles))
	for _, role := range auth0Roles {
		roles = append(roles, mapAuth0RoleToUserRole(role))
	}
	return roles, nil
}

func (c *authClient) AssignUserRole(ctx context.Context, userID string, role models.UserRole) error {
	r := mapUserRoleToAuth0Role(role)
	if r == nil {
		return errors.Errorf("invalid role %d", role)
	}
	return c.management.User.AssignRoles(ctx, userID, []*management.Role{r})
}

func (c *authClient) RemoveUserRole(ctx context.Context, userID string, role models.UserRole) error {
	r := mapUserRoleToAuth0Role(role)
	if r == nil {
		return errors.Errorf("invalid role %d", role)
	}
	return c.management.User.RemoveRoles(ctx, userID, []*management.Role{r})
}

func mapUserRoleToAuth0Role(role models.UserRole) *management.Role {
	switch role {
	case models.UserRoleUser:
		return Auth0UserRole
	case models.UserRoleMember:
		return Auth0MemberRole
	case models.UserRoleAdmin:
		return Auth0AdminRole
	default:
		return nil
	}
}

func mapAuth0RoleToUserRole(role *management.Role) models.UserRole {
	switch role.GetID() {
	case Auth0UserRole.GetID():
		return models.UserRoleUser
	case Auth0MemberRole.GetID():
		return models.UserRoleMember
	case Auth0AdminRole.GetID():
		return models.UserRoleAdmin
	default:
		return models.UserRoleUnknown
	}
}

func mapAuth0UserToUser(user *management.User, roles []models.UserRole) *models.User {
	if user == nil {
		return nil
	}
	return &models.User{
		ID:         user.ID,
		Email:      user.Email,
		Name:       user.Name,
		Picture:    user.Picture,
		GivenName:  user.GivenName,
		FamilyName: user.FamilyName,
		Username:   user.Username,
		Nickname:   user.Nickname,
		ScreenName: user.ScreenName,
		Connection: user.Connection,
		Location:   user.Location,
		LastLogin:  user.LastLogin,
		Roles:      roles,
	}
}

func mapUserToAuth0User(user *models.User) *management.User {
	if user == nil {
		return nil
	}

	auth0User := &management.User{}
	if user.Email != nil {
		auth0User.Email = user.Email
	}
	if user.Name != nil {
		auth0User.Name = user.Name
	}
	if user.Picture != nil {
		auth0User.Picture = user.Picture
	}
	if user.GivenName != nil {
		auth0User.GivenName = user.GivenName
	}
	if user.FamilyName != nil {
		auth0User.FamilyName = user.FamilyName
	}
	if user.Username != nil {
		auth0User.Username = user.Username
	}
	if user.Nickname != nil {
		auth0User.Nickname = user.Nickname
	}

	return auth0User
}
