package version

import "fmt"

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func String() string {
	return fmt.Sprintf("NookBuddy v%s (%s, %s)", Version, Commit, Date)
}
