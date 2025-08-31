package models

import "time"

type Lesson struct {
    ID           uint      `gorm:"primaryKey"`
    Grade        int       `gorm:"not null"`
    GradeLetter  string    `gorm:"size:1"`
    LessonDay    int       `gorm:"not null"` // 1=Mon, 7=Sun
    LessonStart time.Time `gorm:"type:time"`
	LessonEnd   time.Time `gorm:"type:time"`
    LessonName   string    `gorm:"not null"`
    LessonTeacher string
    LessonClass   string
    LessonGroup   string
}

type User struct {
    ID           uint      `gorm:"primaryKey"`
    Email        string    `gorm:"uniqueIndex;not null"`
    AccessToken  string    `gorm:"not null"`
    RefreshToken string
    TokenType    string
    Expiry       time.Time

    Grade       int    // e.g. 11, 12
    GradeLetter string `gorm:"size:1"` // e.g. "A", "B"

    Groups []UserGroup `gorm:"foreignKey:UserID"`
}

type UserGroup struct {
    ID          uint   `gorm:"primaryKey"`
    UserID      uint   `gorm:"not null;index"`
    LessonName  string `gorm:"not null"`
    LessonGroup string

    User User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// LessonGroupFilter is used to filter lessons by name+group
type LessonGroupFilter struct {
	LessonName  string `json:"lesson_name"`
	LessonGroup string `json:"lesson_group"`
}