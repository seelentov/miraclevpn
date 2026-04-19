// Package models provides data models for the application.
package models

const (
	ServerTypeOVPN       = "ovpn"
	ServerTypeAmneziaWG  = "amneziawg"
)

type Server struct {
	ID int64 `gorm:"primaryKey" json:"id"`

	Host       string `gorm:"uniqueIndex" json:"host"`
	Type       string `gorm:"default:ovpn" json:"type"`
	Region     string `json:"region"`
	RegionName string `json:"region_name"`
	Service    string `json:"service"`

	RegionFlagURL string `json:"region_flag_url"`

	MaxUsers int `json:"max_users"`
	MinUsers int `json:"min_users"`

	Preview bool `json:"preview"`
	Active  bool `json:"active"`

	Priority int `json:"priority"`
}
