// Package api registers the HTTP routes for the backend example.
package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/DaiYuANg/arcgo/examples/dix/backend/domain"
	"github.com/DaiYuANg/arcgo/httpx"
	"github.com/danielgtaylor/huma/v2"
)

type listUsersInput struct {
	Limit int    `query:"limit" validate:"omitempty,min=1,max=100"`
	Page  int    `query:"page"  validate:"omitempty,min=1"`
	Q     string `query:"q"     validate:"omitempty,max=100"`
}

type listUsersOutput struct {
	Body struct {
		Items []domain.User `json:"items"`
		Total int           `json:"total"`
		Page  int           `json:"page"`
		Limit int           `json:"limit"`
	} `json:"body"`
}

type getUserInput struct {
	ID int64 `path:"id" validate:"required,min=1"`
}

type getUserOutput struct {
	Body domain.User `json:"body"`
}

type createUserInput struct {
	Body domain.CreateUserInput `json:"body"`
}

type createUserOutput struct {
	Body domain.User `json:"body"`
}

type updateUserInput struct {
	ID   int64                  `path:"id"`
	Body domain.UpdateUserInput `json:"body"`
}

type updateUserOutput struct {
	Body domain.User `json:"body"`
}

type deleteUserInput struct {
	ID int64 `path:"id"`
}

type deleteUserOutput struct {
	Body struct {
		Deleted bool `json:"deleted"`
	} `json:"body"`
}

type healthOutput struct {
	Body struct {
		Status string `json:"status"`
		Time   string `json:"time"`
	} `json:"body"`
}

// UserService defines the user management operations exposed through the API.
type UserService interface {
	List(ctx context.Context, search string, limit, offset int) ([]domain.User, int, error)
	Get(ctx context.Context, id int64) (domain.User, bool, error)
	Create(ctx context.Context, in domain.CreateUserInput) (domain.User, error)
	Update(ctx context.Context, id int64, in domain.UpdateUserInput) (domain.User, bool, error)
	Delete(ctx context.Context, id int64) (bool, error)
}

// RegisterRoutes registers the backend example routes on the provided server.
func RegisterRoutes(server httpx.ServerRuntime, svc UserService) {
	registerHealthRoute(server)

	api := server.Group("/api/v1")
	registerListUsersRoute(api, svc)
	registerGetUserRoute(api, svc)
	registerCreateUserRoute(api, svc)
	registerUpdateUserRoute(api, svc)
	registerDeleteUserRoute(api, svc)
}

func registerHealthRoute(server httpx.ServerRuntime) {
	httpx.MustGet(server, "/health", func(_ context.Context, _ *struct{}) (*healthOutput, error) {
		out := &healthOutput{}
		out.Body.Status = "ok"
		out.Body.Time = time.Now().UTC().Format(time.RFC3339)
		return out, nil
	}, huma.OperationTags("system"))
}

func registerListUsersRoute(api *httpx.Group, svc UserService) {
	httpx.MustGroupGet(api, "/users", func(ctx context.Context, input *listUsersInput) (*listUsersOutput, error) {
		limit, page := input.Limit, input.Page
		if limit <= 0 {
			limit = 10
		}
		if limit > 100 {
			limit = 100
		}
		if page <= 0 {
			page = 1
		}

		offset := (page - 1) * limit
		items, total, err := svc.List(ctx, input.Q, limit, offset)
		if err != nil {
			return nil, wrapServiceError("list users", err)
		}

		out := &listUsersOutput{}
		out.Body.Items = items
		out.Body.Total = total
		out.Body.Page = page
		out.Body.Limit = limit
		return out, nil
	}, huma.OperationTags("users"))
}

func registerGetUserRoute(api *httpx.Group, svc UserService) {
	httpx.MustGroupGet(api, "/users/{id}", func(ctx context.Context, input *getUserInput) (*getUserOutput, error) {
		user, ok, err := svc.Get(ctx, input.ID)
		if err != nil {
			return nil, wrapServiceError("get user", err)
		}
		if !ok {
			return nil, httpx.NewError(http.StatusNotFound, "user not found")
		}
		return &getUserOutput{Body: user}, nil
	}, huma.OperationTags("users"))
}

func registerCreateUserRoute(api *httpx.Group, svc UserService) {
	httpx.MustGroupPost(api, "/users", func(ctx context.Context, input *createUserInput) (*createUserOutput, error) {
		user, err := svc.Create(ctx, input.Body)
		if err != nil {
			return nil, wrapServiceError("create user", err)
		}
		return &createUserOutput{Body: user}, nil
	}, huma.OperationTags("users"))
}

func registerUpdateUserRoute(api *httpx.Group, svc UserService) {
	httpx.MustGroupPut(api, "/users/{id}", func(ctx context.Context, input *updateUserInput) (*updateUserOutput, error) {
		user, ok, err := svc.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, wrapServiceError("update user", err)
		}
		if !ok {
			return nil, httpx.NewError(http.StatusNotFound, "user not found")
		}
		return &updateUserOutput{Body: user}, nil
	}, huma.OperationTags("users"))
}

func registerDeleteUserRoute(api *httpx.Group, svc UserService) {
	httpx.MustGroupDelete(api, "/users/{id}", func(ctx context.Context, input *deleteUserInput) (*deleteUserOutput, error) {
		deleted, err := svc.Delete(ctx, input.ID)
		if err != nil {
			return nil, wrapServiceError("delete user", err)
		}
		if !deleted {
			return nil, httpx.NewError(http.StatusNotFound, "user not found")
		}

		out := &deleteUserOutput{}
		out.Body.Deleted = true
		return out, nil
	}, huma.OperationTags("users"))
}

func wrapServiceError(action string, err error) error {
	return fmt.Errorf("%s: %w", action, err)
}
