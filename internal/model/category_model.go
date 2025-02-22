package model

type CategoryResponse struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

type CategoryCreate struct {
	Name string `validate:"required,min=3,max=100" json:"name"`
}

type CategoryUpdate struct {
	ID   int32  `validate:"required,numeric" json:"id"`
	Name string `validate:"required,min=3,max=100" json:"name"`
}

type CategoryDelete struct {
	ID int32 `validate:"required,numeric" json:"id"`
}

type CategoryGet struct {
	ID int32 `validate:"required,numeric" json:"id"`
}
