package args

type CallbackOption func(string) error

var General struct {
	Verbose               []bool         `short:"v" long:"verbose"             env:"VERBOSITY"            description:"Show verbose debug information"`
	ConfigurationFile     CallbackOption `short:"c" long:"config"              env:"CONFIG"               description:"Configuration file (yaml-formatted)" no-ini:"true"`
	ConfigurationFilePath string
	LogFile               *string `short:"l" long:"log-file"            env:"LOG_FILE"             description:"Log file (file will be appended). If not set, defaults to stderr." default:"-"`
	LogFormat             string  `short:"f" long:"log-format"          env:"LOG_FORMAT"           description:"Log file format (json or text)." choice:"text" choice:"json" default:"text"`
	LogColor              string  `short:"C" long:"log-color"           env:"LOG_COLOR"            description:"Should the log output be colored? true, false or auto" choice:"yes" choice:"no" choice:"true" choice:"false" choice:"auto" default:"auto"`
	LogFullTimestamp      bool    `          long:"log-full-timestamp"  env:"LOG_FULL_TIMESTAMP"   description:"Display full timestamp in logs."`
	LogReportCaller       bool    `          long:"log-report-caller"   env:"LOG_REPORT_CALLER"    description:"If you wish to add the calling method as a field."`
	Experimental          bool    `          long:"experimental"        env:"WSPROXY_EXPERIMENTAL" description:"Enable experimental features"`
}
