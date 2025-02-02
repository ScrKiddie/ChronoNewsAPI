package model

import "mime/multipart"

type UserResponse struct {
	ID             int32  `json:"id,omitempty"`
	Name           string `json:"name,omitempty"`
	ProfilePicture string `json:"profilePicture,omitempty"`
	PhoneNumber    string `json:"phoneNumber,omitempty"`
	Email          string `json:"email,omitempty"`
	Password       string `json:"password,omitempty"`
}

type UserRegister struct {
	Name        string `validate:"required,min=3,max=255" json:"name"`
	PhoneNumber string `validate:"required,e164,max=20" json:"phoneNumber"`
	Email       string `validate:"required,email,max=255" json:"email"`
	Password    string `validate:"required,passwordformat,min=8,max=255" json:"password"`
}

type UserLogin struct {
	Email       string `validate:"omitempty,email,exclusiveor=PhoneNumber" json:"email"`
	PhoneNumber string `validate:"omitempty,e164,exclusiveor=Email,max=20" json:"phoneNumber"`
	Password    string `validate:"required,passwordformat,min=8,max=255" json:"password"`
}

type UserUpdateProfile struct {
	Name           string                `validate:"required,min=3,max=255" json:"name"`
	PhoneNumber    string                `validate:"required,e164,max=20" json:"phoneNumber"`
	Email          string                `validate:"required,email,max=255" json:"email"`
	ProfilePicture *multipart.FileHeader `validate:"omitempty,image" json:"profilePicture"`
}

type UserUpdatePassword struct {
	OldPassword     string `validate:"required,passwordformat,min=8,max=255" json:"oldPassword"`
	Password        string `validate:"required,passwordformat,min=8,max=255" json:"password"`
	ConfirmPassword string `validate:"required,eqfield=Password" json:"confirmPassword"`
}

type UserSearch struct {
	Name        string `validate:"omitempty" json:"name"`
	PhoneNumber string `validate:"omitempty" json:"phoneNumber"`
	Email       string `validate:"omitempty" json:"email"`
	Role        string `validate:"omitempty" json:"role"`
	Page        int64  `validate:"omitempty" json:"page"`
	Size        int64  `validate:"omitempty" json:"size"`
}

type UserCreate struct {
	Name           string                `validate:"required,min=3,max=255" json:"name"`
	PhoneNumber    string                `validate:"required,e164,max=20" json:"phoneNumber"`
	Email          string                `validate:"required,email,max=255" json:"email"`
	ProfilePicture *multipart.FileHeader `validate:"omitempty,image" json:"profilePicture"`
	Password       string                `validate:"required,passwordformat,min=8,max=255" json:"password"`
	Role           string                `validate:"required,oneof=user journalist" json:"role"`
}

type UserUpdate struct {
	ID             int32                 `validate:"required,numeric"`
	Name           string                `validate:"required,min=3,max=255" json:"name"`
	PhoneNumber    string                `validate:"required,e164,max=20" json:"phoneNumber"`
	Email          string                `validate:"required,email,max=255" json:"email"`
	ProfilePicture *multipart.FileHeader `validate:"omitempty,image" json:"profilePicture"`
	Password       string                `validate:"required,passwordformat,min=8,max=255" json:"password"`
	Role           string                `validate:"required,oneof=user journalist" json:"role"`
}

type UserDelete struct {
	ID int32 `validate:"required,numeric"`
}

type UserGet struct {
	ID int32 `validate:"required,numeric"`
}
