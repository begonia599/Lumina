package model

import "time"

type UserSetting struct {
	ID        int       `json:"id"`
	Key       string    `json:"key"`
	Value     any       `json:"value"`
	UpdatedAt time.Time `json:"updatedAt"`
}
