package logging

import (
	"path"
	"runtime"

	"github.com/sirupsen/logrus"
)

// ContextHook ...
type ContextHook struct{}

// Levels ...
func (hook ContextHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire ...
func (hook ContextHook) Fire(entry *logrus.Entry) error {
	if pc, file, line, ok := runtime.Caller(9); ok {
		funcName := runtime.FuncForPC(pc).Name()

		entry.Data["file"] = path.Base(file)
		entry.Data["line"] = line
		entry.Data["func"] = path.Base(funcName)
	}

	return nil
}
