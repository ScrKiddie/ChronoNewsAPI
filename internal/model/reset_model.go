package model

type ResetRequest struct {
	Code            string `validate:"required,min=8,max=255" json:"code"`
	Password        string `validate:"required,passwordformat,min=8,max=255" json:"password"`
	ConfirmPassword string `validate:"required,eqfield=Password" json:"confirmPassword"`
}

type ResetEmailRequest struct {
	Email string `validate:"required,email,max=255" json:"email"`
}
