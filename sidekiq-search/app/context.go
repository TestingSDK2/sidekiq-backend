package app

import (
	"net/http"
	"strconv"

	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
	"github.com/sirupsen/logrus"
)

// Context per request state
type Context struct {
	Logger        logrus.FieldLogger
	RemoteAddress string
	User          *model.Account
	SignupUser    *model.AccountSignup
	Profile       int
	Vars          map[string]string
	ResourceID    int
}

// WithLogger sets logger for context
func (ctx *Context) WithLogger(logger logrus.FieldLogger) *Context {
	ret := *ctx
	ret.Logger = logger
	return &ret
}

// WithRemoteAddress sets remote address for context
func (ctx *Context) WithRemoteAddress(address string) *Context {
	ret := *ctx
	ret.RemoteAddress = address
	return &ret
}

// WithUserProfile sets user for context
func (ctx *Context) WithUserProfile(user *model.Account, profile int) *Context {
	ret := *ctx
	ret.User = user
	ret.Profile = profile
	return &ret
}

// WithUser sets user for context
func (ctx *Context) WithUser(user *model.Account) *Context {
	ret := *ctx
	ret.User = user
	return &ret
}

// GetResourceID - gets and sets the resourceID for the given resource name
func (ctx *Context) GetResourceID(name string) (int, error) {
	id, err := strconv.Atoi(ctx.Vars[name])
	if err == nil {
		if id <= 0 {
			return id, &ValidationError{"Invalid ResourceID"}
		}
		ctx.ResourceID = id
	}
	return id, err
}

// AuthorizationError helper for when user is not authorized
func (ctx *Context) AuthorizationError(isInValidToken bool) *UserError {
	if isInValidToken {
		return &UserError{Message: "Token has expired", StatusCode: http.StatusUnauthorized}
	}
	return &UserError{Message: "Invalid Credentials", StatusCode: http.StatusForbidden}
}
