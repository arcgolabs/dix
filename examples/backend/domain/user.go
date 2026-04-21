// Package domain defines the backend example user models.
package domain

import "time"

// User is the public user model exposed by the backend example.
type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Age       int       `json:"age"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateUserInput contains the fields required to create a user.
type CreateUserInput struct {
	Name  string `json:"name"  validate:"required,min=2,max=64"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age"   validate:"gte=0,lte=130"`
}

// UpdateUserInput contains the fields that can be updated on a user.
type UpdateUserInput struct {
	Name  *string `json:"name,omitempty"  validate:"omitempty,min=2,max=64"`
	Email *string `json:"email,omitempty" validate:"omitempty,email"`
	Age   *int    `json:"age,omitempty"   validate:"omitempty,gte=0,lte=130"`
}
