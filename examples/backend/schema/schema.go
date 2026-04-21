// Package schema defines dbx schemas for the backend example.
package schema

import (
	"time"

	columnx "github.com/DaiYuANg/arcgo/dbx/column"
	schemax "github.com/DaiYuANg/arcgo/dbx/schema"
)

// UserRow is the dbx row model for the users table.
type UserRow struct {
	ID        int64     `dbx:"id"`
	Name      string    `dbx:"name"`
	Email     string    `dbx:"email"`
	Age       int       `dbx:"age"`
	CreatedAt time.Time `dbx:"created_at,codec=rfc3339_time"`
	UpdatedAt time.Time `dbx:"updated_at,codec=rfc3339_time"`
}

// UserSchema describes the users table for the backend example.
type UserSchema struct {
	schemax.Schema[UserRow]
	ID        columnx.Column[UserRow, int64]     `dbx:"id,pk,auto"`
	Name      columnx.Column[UserRow, string]    `dbx:"name"`
	Email     columnx.Column[UserRow, string]    `dbx:"email,unique"`
	Age       columnx.Column[UserRow, int]       `dbx:"age"`
	CreatedAt columnx.Column[UserRow, time.Time] `dbx:"created_at,codec=rfc3339_time"`
	UpdatedAt columnx.Column[UserRow, time.Time] `dbx:"updated_at,codec=rfc3339_time"`
}
