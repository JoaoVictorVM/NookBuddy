package storage

import "time"

type Notice int

const (
	NoticeNone Notice = iota
	NoticeRecoveredFromCorruption
)

var CosmeticIDs = []string{"window", "bookshelf", "flower", "painting", "rug"}

type PlayerState struct {
	ClicksProgress     int
	KeysProgress       int
	ProjectsReady      int
	Gold               int
	UpgradeClicksLevel int
	UpgradeKeysLevel   int
	UpgradeValueLevel  int
	OwnedCosmetics     []string
}

type SaveResult struct {
	OK      bool
	SavedAt time.Time
}
