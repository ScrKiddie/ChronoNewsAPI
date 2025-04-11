package model

import "mime/multipart"

type PostResponse struct {
	ID            int32  `json:"id"`
	CategoryID    int32  `json:"categoryID,omitempty"`
	UserID        int32  `json:"userID,omitempty"`
	Title         string `json:"title"`
	Summary       string `json:"summary,omitempty"`
	Content       string `json:"content,omitempty"`
	PublishedDate int64  `json:"publishedDate"`
	LastUpdated   int64  `json:"lastUpdated"`
	Thumbnail     string `json:"thumbnail"`
}

type PostResponseWithPreload struct {
	ID            int32             `json:"id"`
	Category      *CategoryResponse `json:"category,omitempty"`
	User          *UserResponse     `json:"user,omitempty"`
	Title         string            `json:"title"`
	Summary       string            `json:"summary,omitempty"`
	Content       string            `json:"content,omitempty"`
	PublishedDate int64             `json:"publishedDate"`
	LastUpdated   int64             `json:"lastUpdated"`
	Thumbnail     string            `json:"thumbnail"`
	ViewCount     int64             `json:"viewCount"`
}

type PostGet struct {
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
}

type PostCreate struct {
	Title      string `validate:"required,max=255"`
	Summary    string `validate:"required,max=1000"`
	Content    string
	UserID     int32                 `validate:"omitempty,required"`
	CategoryID int32                 `validate:"required"`
	Thumbnail  *multipart.FileHeader `validate:"omitempty,image=1200_675"`
}

type PostUpdate struct {
	ID              int32  `validate:"required"`
	Title           string `validate:"required,max=255"`
	Summary         string `validate:"required,max=1000"`
	Content         string
	UserID          int32                 `validate:"omitempty,required"`
	CategoryID      int32                 `validate:"required"`
	Thumbnail       *multipart.FileHeader `validate:"omitempty,image=1200_675"`
	DeleteThumbnail bool
}

type PostDelete struct {
	ID int32 `validate:"required"`
}
