package models

import "time"

type User struct {
	UserID       int    `gorm:"column:user_id;primaryKey"`
	Username     string `gorm:"column:username"`
	PasswordHash string `gorm:"column:password_hash"`
	IsModerator  bool   `gorm:"column:is_moderator"`
}

type Star struct {
	StarID           int     `gorm:"primaryKey;column:star_id"`
	StarName         string  `gorm:"column:star_name"`
	ShortDescription string  `gorm:"column:short_description"`
	Description      string  `gorm:"column:description"`
	ImageURL         string  `gorm:"column:image_url"`
	IsActive         bool    `gorm:"column:is_active"`
	RA               float64 `gorm:"column:ra"`
	Dec              float64 `gorm:"column:dec"`

	// связь многие-ко-многим через observation_stars
	Observations []Observation `gorm:"many2many:observation_stars;foreignKey:StarID;joinForeignKey:star_id;References:ObservationID;joinReferences:observation_id"`
}

type Observation struct {
	ObservationID  int        `gorm:"primaryKey;column:observation_id"`
	CreatorID      int        `gorm:"column:creator_id"`
	ModeratorID    *int       `gorm:"column:moderator_id"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	Status         string     `gorm:"column:status"`
	FormationDate  *time.Time `gorm:"column:formation_date"`
	CompletionDate *time.Time `gorm:"column:completion_date"`

	Creator   User  `gorm:"foreignKey:CreatorID;references:UserID"`
	Moderator *User `gorm:"foreignKey:ModeratorID;references:UserID"`

	Stars            []Star            `gorm:"many2many:observation_stars;foreignKey:ObservationID;joinForeignKey:ObservationID;References:StarID;joinReferences:StarID"`
	ObservationStars []ObservationStar `gorm:"foreignKey:ObservationID"`
}

type ObservationStar struct {
	ObservationID     int        `gorm:"primaryKey;column:observation_id"`
	StarID            int        `gorm:"primaryKey;column:star_id"`
	OrderNumber       int        `gorm:"column:order_number"`
	Quantity          int        `gorm:"column:quantity"`
	ResultValue       *float64   `gorm:"column:result_value"`
	ObservationDate   *time.Time `gorm:"column:observation_date"`
	ObserverLatitude  float64    `gorm:"column:observer_latitude"`
	ObserverLongitude float64    `gorm:"column:observer_longitude"`

	Observation Observation `gorm:"foreignKey:ObservationID;references:ObservationID"`
	Star        Star        `gorm:"foreignKey:StarID;references:StarID"`
}
