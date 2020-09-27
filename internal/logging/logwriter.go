package logging

import (
	"fmt"
	"github.com/sirupsen/logrus"
)

type ChiLogWriter struct {
}

func (lw *ChiLogWriter) Print(a ...interface{}) {
	if len(a) == 1 {
		msg := fmt.Sprintf("%s", a)
		if msg[0] == '[' && msg[len(msg)-1:] == "]" {
			msg = msg[1 : len(msg)-1]
		}
		logrus.Debugf(msg)
	} else {
		logrus.Debugf(fmt.Sprintf("%s", a[0]), a[1:])
	}
}
