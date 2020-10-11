package logging

import (
	"path"
	"runtime"

	"github.com/sirupsen/logrus"
)

// ContextHook will add go source information (file, line, func)
type ContextHook struct{}

// Levels defines which logging levels fire the hook. In our case, all levels.
func (hook ContextHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire is the method that's executed when logging event is logged. This method will go back the call stack
// and find which method executed the call.
func (hook ContextHook) Fire(entry *logrus.Entry) error {
	if pc, file, line, ok := runtime.Caller(9); ok {
		funcName := runtime.FuncForPC(pc).Name()

		entry.Data["file"] = path.Base(file)
		entry.Data["line"] = line
		entry.Data["func"] = path.Base(funcName)
	}

	return nil
}
