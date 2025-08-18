package models

import "time"

type Verifier struct {
	UserID    int64
	Code      int32
	ExpiredAt time.Time
}
