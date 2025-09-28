package model

import "mime/multipart"

type FileUpload struct {
	File *multipart.FileHeader `validate:"required,image=16383_16383_10"`
}

type ImageUploadResponse struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}
