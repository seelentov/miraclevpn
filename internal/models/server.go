// Package models provides data models for the application.
package models

type Server struct {
	ID int64 `gorm:"primaryKey"`

	Host    string `gorm:"uniqueIndex"`
	Region  string
	Service string

	Active bool
}
