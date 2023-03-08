package models

import "time"

type BlockingRequest struct {
	Names           []string      `form:"names"`
	Group           string        `form:"group"`
	SessionDuration time.Duration `form:"session_duration" binding:"required"`
	Timeout         time.Duration `form:"timeout"`
}
