package input

import "time"

type EventType int

const (
	Click EventType = iota
	Key
)

type Event struct {
	Type      EventType
	Timestamp time.Time
}

type Status int

const (
	Unavailable Status = iota
	Available
)
