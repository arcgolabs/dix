package repo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/arcgolabs/dbx"
	mapperx "github.com/arcgolabs/dbx/mapper"
	"github.com/arcgolabs/dbx/querydsl"
	"github.com/arcgolabs/dix/examples/backend/domain"
	backendschema "github.com/arcgolabs/dix/examples/backend/schema"
)

// UserRepository persists backend users.
type UserRepository interface {
	List(ctx context.Context, search string, limit, offset int) ([]domain.User, int, error)
	GetByID(ctx context.Context, id int64) (domain.User, bool, error)
	Create(ctx context.Context, in domain.CreateUserInput) (domain.User, error)
	Update(ctx context.Context, id int64, in domain.UpdateUserInput) (domain.User, bool, error)
	Delete(ctx context.Context, id int64) (bool, error)
}

type userRepo struct {
	db     *dbx.DB
	schema backendschema.UserSchema
}

// NewUserRepository creates a user repository backed by dbx.
func NewUserRepository(db *dbx.DB, userSchema backendschema.UserSchema) UserRepository {
	return &userRepo{db: db, schema: userSchema}
}

func (r *userRepo) List(ctx context.Context, search string, limit, offset int) ([]domain.User, int, error) {
	s := r.schema
	mapper := mapperx.MustMapper[backendschema.UserRow](s)

	q := querydsl.Select(querydsl.AllColumns(s).Values()...).From(s)
	if search != "" {
		pattern := "%" + strings.TrimSpace(search) + "%"
		q = q.Where(querydsl.Or(
			querydsl.Like(s.Name, pattern),
			querydsl.Like(s.Email, pattern),
		))
	}
	q = q.OrderBy(s.ID.Asc())

	all, err := dbx.QueryAll[backendschema.UserRow](ctx, r.db, q, mapper)
	if err != nil {
		return nil, 0, err
	}
	total := all.Len()

	if offset >= total {
		return []domain.User{}, total, nil
	}
	page := all.Drop(offset).Take(min(limit, total-offset))

	users := make([]domain.User, page.Len())
	page.Range(func(i int, row backendschema.UserRow) bool {
		users[i] = rowToDomain(row)
		return true
	})
	return users, total, nil
}

func (r *userRepo) GetByID(ctx context.Context, id int64) (domain.User, bool, error) {
	s := r.schema
	mapper := mapperx.MustMapper[backendschema.UserRow](s)

	rows, err := dbx.QueryAll[backendschema.UserRow](ctx, r.db,
		querydsl.Select(querydsl.AllColumns(s).Values()...).From(s).Where(s.ID.Eq(id)),
		mapper,
	)
	if err != nil {
		return domain.User{}, false, err
	}
	if rows.IsEmpty() {
		return domain.User{}, false, nil
	}
	row, _ := rows.GetFirst()
	return rowToDomain(row), true, nil
}

func (r *userRepo) Create(ctx context.Context, in domain.CreateUserInput) (domain.User, error) {
	s := r.schema
	mapper := mapperx.MustMapper[backendschema.UserRow](s)
	now := time.Now().UTC()

	rows, err := dbx.QueryAll[backendschema.UserRow](ctx, r.db,
		querydsl.InsertInto(s).
			Columns(s.Name, s.Email, s.Age, s.CreatedAt, s.UpdatedAt).
			Values(
				s.Name.Set(in.Name),
				s.Email.Set(in.Email),
				s.Age.Set(in.Age),
				s.CreatedAt.Set(now),
				s.UpdatedAt.Set(now),
			).
			Returning(querydsl.AllColumns(s).Values()...),
		mapper,
	)
	if err != nil {
		return domain.User{}, err
	}
	row, _ := rows.GetFirst()
	return rowToDomain(row), nil
}

func (r *userRepo) Update(ctx context.Context, id int64, in domain.UpdateUserInput) (domain.User, bool, error) {
	s := r.schema
	mapper := mapperx.MustMapper[backendschema.UserRow](s)

	assignments := []querydsl.Assignment{s.UpdatedAt.Set(time.Now().UTC())}
	if in.Name != nil {
		assignments = append(assignments, s.Name.Set(*in.Name))
	}
	if in.Email != nil {
		assignments = append(assignments, s.Email.Set(*in.Email))
	}
	if in.Age != nil {
		assignments = append(assignments, s.Age.Set(*in.Age))
	}

	rows, err := dbx.QueryAll[backendschema.UserRow](ctx, r.db,
		querydsl.Update(s).Set(assignments...).Where(s.ID.Eq(id)).Returning(querydsl.AllColumns(s).Values()...),
		mapper,
	)
	if err != nil {
		return domain.User{}, false, err
	}
	if rows.IsEmpty() {
		return domain.User{}, false, nil
	}
	row, _ := rows.GetFirst()
	return rowToDomain(row), true, nil
}

func (r *userRepo) Delete(ctx context.Context, id int64) (bool, error) {
	s := r.schema
	res, err := dbx.Exec(ctx, r.db, querydsl.DeleteFrom(s).Where(s.ID.Eq(id)))
	if err != nil {
		return false, fmt.Errorf("delete user: %w", err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("delete user rows affected: %w", err)
	}
	return ra > 0, nil
}

func rowToDomain(row backendschema.UserRow) domain.User {
	return domain.User{
		ID:        row.ID,
		Name:      row.Name,
		Email:     row.Email,
		Age:       row.Age,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}
