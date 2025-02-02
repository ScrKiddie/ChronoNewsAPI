package model

import "mime/multipart"

type UserResponse struct {
	ID             int32  `json:"id,omitempty"`
	Name           string `json:"name,omitempty"`
	ProfilePicture string `json:"profilePicture,omitempty"`
	PhoneNumber    string `json:"phoneNumber,omitempty"`
	Email          string `json:"email,omitempty"`
	Password       string `json:"password,omitempty"`
	Token          string `json:"token,omitempty"`
	About          string `json:"about,omitempty"`
	PrevCursor     int32  `json:"prevCursor,omitempty"`
	NextCursor     int32  `json:"nextCursor,omitempty"`
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

type UserAuthorization struct {
	ID int32
}

type UserUpdateProfile struct {
	ID             int32
	Name           string                `validate:"required,min=3,max=255" json:"name"`
	PhoneNumber    string                `validate:"required,e164,max=20" json:"phoneNumber"`
	Email          string                `validate:"required,email,max=255" json:"email"`
	ProfilePicture *multipart.FileHeader `validate:"omitempty,image" json:"profilePicture"`
	About          string                `validate:"max=500" json:"about,omitempty"`
}

type UserUpdatePassword struct {
	ID              int32
	OldPassword     string `validate:"required,passwordformat,min=8,max=255" json:"oldPassword"`
	Password        string `validate:"required,passwordformat,min=8,max=255" json:"password"`
	ConfirmPassword string `validate:"required,eqfield=Password" json:"confirmPassword"`
}

type UserSearch struct {
	ID          int32
	Name        string `validate:"omitempty" json:"name"`
	PhoneNumber string `validate:"omitempty" json:"phoneNumber"`
	Email       string `validate:"omitempty" json:"email"`
	Role        string `validate:"omitempty" json:"role"`
	Page        int64  `validate:"omitempty" json:"page"`
	Size        int64  `validate:"omitempty" json:"size"`
}
