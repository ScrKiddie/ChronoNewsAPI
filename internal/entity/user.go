package entity

type User struct {
	ID          int32  `gorm:"column:id;primaryKey;type:integer;autoIncrement;not null"`
	Name        string `gorm:"type:varchar(255);not null;column:name"`
	PhoneNumber string `gorm:"type:varchar(20);not null;column:phone_number"`
	Email       string `gorm:"type:varchar(255);unique;not null;column:email"`
	Password    string `gorm:"type:varchar(255);column:password"`
	Role        string `gorm:"type:user_type;not null;column:role"`
	Files       []File `gorm:"foreignKey:UsedByUserID"`
}

func (User) TableName() string {
	return "user"
}
