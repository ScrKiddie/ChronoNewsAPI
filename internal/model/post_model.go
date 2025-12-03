package model

import "mime/multipart"

type PostResponse struct {
	ID         int32  `json:"id"`
	CategoryID int32  `json:"categoryID,omitempty"`
	UserID     int32  `json:"userID,omitempty"`
	Title      string `json:"title"`
	Summary    string `json:"summary,omitempty"`
	Content    string `json:"content,omitempty"`
	CreatedAt  int64  `json:"createdAt"`
	UpdatedAt  int64  `json:"updatedAt"`
	Thumbnail  string `json:"thumbnail"`
}

type PostResponseWithPreload struct {
	ID        int32             `json:"id"`
	Category  *CategoryResponse `json:"category,omitempty"`
	User      *UserResponse     `json:"user,omitempty"`
	Title     string            `json:"title"`
	Summary   string            `json:"summary,omitempty"`
	Content   string            `json:"content,omitempty"`
	CreatedAt int64             `json:"createdAt"`
	UpdatedAt int64             `json:"updatedAt"`
	Thumbnail string            `json:"thumbnail"`
	ViewCount int64             `json:"viewCount"`
}

type PostGet struct {
	ID int32 `validate:"required"`
}

type PostIncrementView struct {
	ID int32 `validate:"required"`
}

type PostSearch struct {
	UserID       int32
	Title        string
	CategoryName string
	UserName     string
	Summary      string
	Page         int64
	Size         int64
	Sort         string
	StartDate    int64
	EndDate      int64
	ExcludeIDs   string
}

type PostCreate struct {
	Title      string                `validate:"required,max=255"`
	Summary    string                `validate:"required,max=1000"`
	Content    string                `validate:"max=65535"`
	UserID     int32                 `validate:"omitempty,required"`
	CategoryID int32                 `validate:"required"`
	Thumbnail  *multipart.FileHeader `validate:"omitempty,image=1200_675_2"`
}

type PostUpdate struct {
	ID              int32                 `validate:"required"`
	Title           string                `validate:"required,max=255"`
	Summary         string                `validate:"required,max=1000"`
	Content         string                `validate:"max=65535"`
	UserID          int32                 `validate:"omitempty,required"`
	CategoryID      int32                 `validate:"required"`
	Thumbnail       *multipart.FileHeader `validate:"omitempty,image=1200_675_2"`
	DeleteThumbnail bool
}

type PostDelete struct {
	ID int32 `validate:"required"`
}
