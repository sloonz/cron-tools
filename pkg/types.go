package crontools

import (
	"github.com/gbrlsnchs/jwt/v3"
)

type Job struct {
	jwt.Payload
	User    string   `json:"user,omitempty"`
	Host    string   `json:"host,omitempty"`
	ID      string   `json:"id"`
	Command []string `json:"command"`
}

const StoreTimeFormat = "2006-01-02T150405Z0700"
