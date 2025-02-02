package entity

type Category struct {
	ID   int32  `gorm:"column:id;primaryKey;type:integer;autoIncrement;not null"`
	Name string `gorm:"column:name;type:varchar(100);not null"`
}

func (Category) TableName() string {
	return "category"
}
