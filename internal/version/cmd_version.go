package version

import (
	"github.com/k0kubun/go-ansi"
	"os"
)

const (
	Bold           = "\x1b[1m"
	Reset          = "\x1b[0m"
	LightGray      = "\x1b[37m"
	DarkGray       = "\x1b[90m"
	White          = "\x1b[97m"
	BackgroundBlue = "\x1b[44m"
)

// Import is a command which imports emails from the given IMAP server into our loal database.
type Command struct {
}

func (i *Command) String() string {
	return "Version details"
}

func (i *Command) Execute(args []string) error {
	PrintVersion()
	ansi.Printf(DarkGray+" Author      "+White+"%+v"+Reset+"\n", "Bojan Cekrlic <github.com/bokysan>")
	ansi.Printf(DarkGray+" Git tag     "+White+"%+v"+Reset+"\n", GitTag)
	ansi.Printf(DarkGray+" Git branch  "+White+"%+v"+Reset+"\n", GitBranch)
	ansi.Printf(DarkGray+" Git state   "+White+"%+v"+Reset+"\n", GitState)
	ansi.Printf(DarkGray+" Go version  "+White+"%+v"+Reset+"\n", GoVersion)
	os.Exit(0)
	return nil
}

func PrintVersion() {
	ansi.Printf(Bold+BackgroundBlue+
		LightGray+" UNIVERSAL PROXY "+White+"%s"+LightGray+" "+Reset+"\n"+
		DarkGray+" Built on    "+White+"%+v\n"+
		DarkGray+" Git version "+White+"%+v"+DarkGray+"/"+White+"%+v"+Reset+"\n",
		Version, BuildDate, GitBranch, GitCommit)
}
