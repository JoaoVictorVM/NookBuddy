package input

import "time"

type rawKind int

const (
	rawIgnored rawKind = iota
	rawReady
	rawClick
	rawKey
)

type rawEvent struct {
	kind rawKind
	when time.Time
}

type backend interface {
	start() error
	next() (rawEvent, bool)
	stop()
}
