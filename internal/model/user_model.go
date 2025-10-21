package model

import "mime/multipart"

type UserResponse struct {
	ID             int32  `json:"id,omitempty"`
	Name           string `json:"name,omitempty"`
	ProfilePicture string `json:"profilePicture,omitempty"`
	PhoneNumber    string `json:"phoneNumber,omitempty"`
	Email          string `json:"email,omitempty"`
	Role           string `json:"role,omitempty"`
}

type UserRegister struct {
	Name        string `validate:"required,min=3,max=255" json:"name"`
	PhoneNumber string `validate:"required,max=20" json:"phoneNumber"`
	Email       string `validate:"required,email,max=255" json:"email"`
	Password    string `validate:"required,passwordformat,min=8,max=255" json:"password"`
}

type UserLogin struct {
	Email        string `validate:"required,email,max=255" json:"email"`
	Password     string `validate:"required,passwordformat,min=8,max=255" json:"password"`
	TokenCaptcha string `json:"tokenCaptcha" validate:"required"`
}

type UserUpdateProfile struct {
	Name                 string                `validate:"required,min=3,max=255"`
	PhoneNumber          string                `validate:"required,max=20"`
	Email                string                `validate:"required,email,max=255"`
	ProfilePicture       *multipart.FileHeader `validate:"omitempty,image=800_800_2"`
	DeleteProfilePicture bool
}

type UserUpdatePassword struct {
	OldPassword     string `validate:"required,passwordformat,min=8,max=255" json:"oldPassword"`
	Password        string `validate:"required,passwordformat,min=8,max=255" json:"password"`
	ConfirmPassword string `validate:"required,eqfield=Password" json:"confirmPassword"`
}

type UserSearch struct {
	Name        string
	PhoneNumber string
	Email       string
	Role        string
	Page        int64
	Size        int64
}

type UserCreate struct {
	Name           string                `validate:"required,min=3,max=255"`
	PhoneNumber    string                `validate:"required,max=20"`
	Email          string                `validate:"required,email,max=255"`
	ProfilePicture *multipart.FileHeader `validate:"omitempty,image=800_800_2"`
	Role           string                `validate:"required,oneof=admin journalist"`
}

type UserUpdate struct {
	ID                   int32                 `validate:"required"`
	Name                 string                `validate:"required,min=3,max=255"`
	PhoneNumber          string                `validate:"required,max=20"`
	Email                string                `validate:"required,email,max=255"`
	ProfilePicture       *multipart.FileHeader `validate:"omitempty,image=800_800_2"`
	Password             string                `validate:"omitempty,passwordformat,min=8,max=255"`
	Role                 string                `validate:"required,oneof=admin journalist"`
	DeleteProfilePicture bool
}

type UserDelete struct {
	ID int32 `validate:"required"`
}

type UserGet struct {
	ID int32 `validate:"required"`
}
