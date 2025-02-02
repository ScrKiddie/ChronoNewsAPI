package model

type Pagination struct {
	Page      int64 `json:"page"`
	Size      int64 `json:"size"`
	TotalItem int64 `json:"totalItem"`
	TotalPage int64 `json:"totalPage"`
}
