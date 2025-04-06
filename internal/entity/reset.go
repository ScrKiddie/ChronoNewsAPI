package entity

type Reset struct {
	ID        int32  `gorm:"column:id;primaryKey;type:integer;autoIncrement;not null"`
	UserID    int32  `gorm:"column:user_id;type:integer;not null;uniqueIndex"`
	User      User   `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Code      string `gorm:"column:code;type:varchar(255);not null"`
	ExpiredAt int64  `gorm:"column:expired_at;type:bigint;not null"`
}

func (Reset) TableName() string {
	return "reset"
}
