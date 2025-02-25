package entity

type Like struct {
	ID     int32 `gorm:"column:id;primaryKey;autoIncrement;not null"`
	PostID int32 `gorm:"column:post_id;type:integer;not null"`
	Post   Post  `gorm:"foreignKey:PostID"`
	UserID int32 `gorm:"column:user_id;type:integer;not null"`
	User   User  `gorm:"foreignKey:UserID"`
}

func (Like) TableName() string {
	return "like"
}
