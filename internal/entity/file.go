package entity

type File struct {
	ID        int32   `gorm:"column:id;primaryKey;type:integer;autoIncrement;not null"`
	ArticleID int32   `gorm:"column:article_id;type:integer;not null"`
	Article   Article `gorm:"foreignKey:ArticleID"`
	Name      string  `gorm:"column:name;type:varchar(100);"`
}

func (File) TableName() string {
	return "file"
}
