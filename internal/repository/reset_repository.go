package repository

import "chronoverseapi/internal/entity"

type ResetRepository struct {
	CommonRepository[entity.Reset]
}

func NewResetRepository() *ResetRepository {
	return &ResetRepository{}
}
