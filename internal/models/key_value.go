package models

type KeyValue struct {
	ID    int64  `gorm:"primaryKey"`
	Key   string `gorm:"uniqueIndex"`
	Value string
}
