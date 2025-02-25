package model

import "mime/multipart"

type UserResponse struct {
	ID             int32  `json:"id,omitempty"`
	Name           string `json:"name,omitempty"`
	ProfilePicture string `json:"profilePicture,omitempty"`
	PhoneNumber    string `json:"phoneNumber,omitempty"`
	Email          string `json:"email,omitempty"`
	Password       string `json:"password,omitempty"`
	Role           string `json:"role,omitempty"`
}

type UserRegister struct {
	Name        string `validate:"required,min=3,max=255" json:"name"`
	PhoneNumber string `validate:"required,max=20" json:"phoneNumber"`
	Email       string `validate:"required,email,max=255" json:"email"`
	Password    string `validate:"required,passwordformat,min=8,max=255" json:"password"`
}

type UserLogin struct {
	Email       string `validate:"omitempty,email,exclusiveor=PhoneNumber" json:"email"`
	PhoneNumber string `validate:"omitempty,exclusiveor=Email,max=20" json:"phoneNumber"`
	Password    string `validate:"required,passwordformat,min=8,max=255" json:"password"`
}

type UserUpdateProfile struct {
	Name           string                `validate:"required,min=3,max=255"`
	PhoneNumber    string                `validate:"required,max=20"`
	Email          string                `validate:"required,email,max=255"`
	ProfilePicture *multipart.FileHeader `validate:"omitempty,image"`
}

type UserUpdatePassword struct {
	OldPassword     string `validate:"required,passwordformat,min=8,max=255" json:"oldPassword"`
	Password        string `validate:"required,passwordformat,min=8,max=255" json:"password"`
	ConfirmPassword string `validate:"required,eqfield=Password" json:"confirmPassword"`
}

type UserSearch struct {
	Name        string `validate:"omitempty"`
	PhoneNumber string `validate:"omitempty"`
	Email       string `validate:"omitempty"`
	Role        string `validate:"omitempty"`
	Page        int64  `validate:"omitempty"`
	Size        int64  `validate:"omitempty"`
}

type UserCreate struct {
	Name           string                `validate:"required,min=3,max=255"`
	PhoneNumber    string                `validate:"required,max=20"`
	Email          string                `validate:"required,email,max=255"`
	ProfilePicture *multipart.FileHeader `validate:"omitempty,image"`
	Password       string                `validate:"required,passwordformat,min=8,max=255"`
	Role           string                `validate:"required,oneof=admin journalist"`
}

type UserUpdate struct {
	ID             int32                 `validate:"required"`
	Name           string                `validate:"required,min=3,max=255"`
	PhoneNumber    string                `validate:"required,max=20"`
	Email          string                `validate:"required,email,max=255"`
	ProfilePicture *multipart.FileHeader `validate:"omitempty,image"`
	Password       string                `validate:"omitempty,passwordformat,min=8,max=255"`
	Role           string                `validate:"required,oneof=admin journalist"`
}

type UserDelete struct {
	ID int32 `validate:"required"`
}

type UserGet struct {
	ID int32 `validate:"required"`
}
