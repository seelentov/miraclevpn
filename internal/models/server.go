// Package models provides data models for the application.
package models

type Server struct {
	ID int64 `gorm:"primaryKey" json:"id"`

	Host       string `gorm:"uniqueIndex" json:"host"`
	Region     string `json:"region"`
	RegionName string `json:"region_name"`
	Service    string `json:"service"`

	RegionFlagURL string `json:"region_flag_url"`

	MaxUsers string `json:"max_users"`
	MinUsers string `json:"min_users"`

	Preview bool `json:"preview"`
	Active  bool `json:"active"`
}
