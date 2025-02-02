package entity

type Like struct {
	ID        int32   `gorm:"column:id;primaryKey;autoIncrement;not null"`
	ArticleID int32   `gorm:"column:article_id;type:integer;not null"`
	Article   Article `gorm:"foreignKey:ArticleID"`
	UserID    int32   `gorm:"column:user_id;type:integer;not null"`
	User      User    `gorm:"foreignKey:UserID"`
}

func (Like) TableName() string {
	return "like"
}
