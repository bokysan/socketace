package version


const UnknownVersion = "unknown"
const ProtocolVersion = "v1.0.0"

// provided at compile time
var (
	GitCommit  string // long commit hash of source tree, e.g. "0b5ed7a"
	GitBranch  string // current branch name the code is built off, e.g. "master"
	GitTag     string // current tag name the code is built off, e.g. "v1.5.0"
	GitSummary string // output of "git describe --tags --dirty --always", e.g. "4cb95ca-dirty"
	GitState   string // whether there are uncommitted changes, e.g. "clean" or "dirty"
	BuildDate  string // RFC3339 formatted UTC date, e.g. "2016-08-04T18:07:54Z"
	Version    string // contents of ./VERSION file, if exists
	GoVersion  string // the version of go, e.g. "go version go1.10.3 darwin/amd64"
	ProtoVersion string = "v1.0.0"
)

func AppVersion() string {
	if GitTag != "" {
		return GitTag
	} else if Version != "" {
		return Version
	}

	return UnknownVersion
}
