package entity

type File struct {
	ID     int32  `gorm:"column:id;primaryKey;type:integer;autoIncrement;not null"`
	PostID int32  `gorm:"column:post_id;type:integer;not null"`
	Post   Post   `gorm:"foreignKey:PostID"`
	Name   string `gorm:"column:name;type:varchar(100);"`
}

func (File) TableName() string {
	return "file"
}
