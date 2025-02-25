package entity

type Comment struct {
	ID        int32  `gorm:"column:id;primaryKey;type:integer;autoIncrement;not null"`
	PostID    int32  `gorm:"column:post_id;type:integer;not null"`
	Post      Post   `gorm:"foreignKey:PostID"`
	UserID    int32  `gorm:"column:user_id;type:integer;not null"`
	User      User   `gorm:"foreignKey:UserID"`
	ParentID  *int32 `gorm:"column:parent_id;type:integer"`
	Content   string `gorm:"column:content;type:text;not null"`
	CreatedAt int64  `gorm:"column:created_at;type:bigint;not null"`
}

func (Comment) TableName() string {
	return "comment"
}
