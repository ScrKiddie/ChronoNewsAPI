package entity

type Article struct {
	ID            int32    `gorm:"column:id;primaryKey;type:integer;autoIncrement;not null"`
	UserID        int32    `gorm:"column:user_id;type:integer;not null"`
	User          User     `gorm:"foreignKey:UserID"`
	CategoryID    int32    `gorm:"column:category_id;type:integer;not null"`
	Category      Category `gorm:"foreignKey:CategoryID"`
	Title         string   `gorm:"column:title;type:varchar(255);not null"`
	Summary       string   `gorm:"column:summary;type:varchar(1000);not null"`
	Content       string   `gorm:"column:content;type:text;not null"`
	PublishedDate int64    `gorm:"column:published_date;type:bigint;not null"`
	LastUpdated   int64    `gorm:"column:last_updated;type:bigint;not null"`
	Banner        string   `gorm:"column:banner;type:varchar(100);"`
}

func (Article) TableName() string {
	return "article"
}
