package models

type KeyValue struct {
	ID    int64  `gorm:"primaryKey" json:"id"`
	Key   string `gorm:"uniqueIndex" json:"key"`
	Value string `json:"value"`
}
