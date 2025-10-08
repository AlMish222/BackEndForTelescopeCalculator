package models

import "time"

type User struct {
	UserID       int    `gorm:"column:user_id;primaryKey"`
	Username     string `gorm:"column:username"`
	PasswordHash string `gorm:"column:password_hash"`
	IsModerator  bool   `gorm:"column:is_moderator"`
}

type Star struct {
	StarID      int     `gorm:"column:star_id;primaryKey"`
	Name        string  `gorm:"column:name"`
	Description string  `gorm:"column:description"`
	ImageURL    string  `gorm:"column:image_url"`
	IsActive    bool    `gorm:"column:is_active"`
	Deleted     bool    `gorm:"column:deleted"`
	RA          float64 `gorm:"column:ra"`
	Dec         float64 `gorm:"column:dec"`

	// связь многие-ко-многим через observation_stars
	Observations []Observation `gorm:"many2many:observation_stars;foreignKey:StarID;joinForeignKey:star_id;References:ObservationID;joinReferences:observation_id"`
}

type Observation struct {
	ObservationID     int        `gorm:"column:observation_id;primaryKey"`
	Status            string     `gorm:"column:status"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
	FormationDate     *time.Time `gorm:"column:formation_date"`
	CompletionDate    *time.Time `gorm:"column:completion_date"`
	ObservationDate   *time.Time `gorm:"column:observation_date"`
	CreatorID         int        `gorm:"column:creator_id"`
	Creator           User       `gorm:"foreignKey:CreatorID;references:UserID"`
	ModeratorID       *int       `gorm:"column:moderator_id"`
	Moderator         *User      `gorm:"foreignKey:ModeratorID;references:UserID"`
	ObserverLatitude  float64    `gorm:"column:observer_latitude"`
	ObserverLongitude float64    `gorm:"column:observer_longitude"`
	ResultValue       *float64   `gorm:"column:result_value"`

	Stars []Star `gorm:"many2many:observation_stars;foreignKey:ObservationID;joinForeignKey:observation_id;References:StarID;joinReferences:star_id"`
}

type ObservationStar struct {
	ObservationID int  `gorm:"column:observation_id;primaryKey"`
	StarID        int  `gorm:"column:star_id;primaryKey"`
	IsMain        bool `gorm:"column:is_main"`
	OrderNumber   int  `gorm:"column:order_number"`
	Quantity      int  `gorm:"column:quantity"`
}
