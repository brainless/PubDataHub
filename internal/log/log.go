package log

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Logger *logrus.Logger

func InitLogger(verbose bool) {
	Logger = logrus.New()
	Logger.SetOutput(os.Stdout)
	Logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	if verbose {
		Logger.SetLevel(logrus.DebugLevel)
	} else {
		Logger.SetLevel(logrus.InfoLevel)
	}
}

// InitLoggerForTUI initializes logger with appropriate level for TUI mode
// In TUI mode, we want to reduce log noise while keeping important messages
func InitLoggerForTUI(verbose bool) {
	Logger = logrus.New()
	Logger.SetOutput(os.Stdout)
	Logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	if verbose {
		Logger.SetLevel(logrus.DebugLevel)
	} else {
		// In TUI mode, only show warnings and errors to reduce clutter
		// The status bar will show download progress instead of logs
		Logger.SetLevel(logrus.WarnLevel)
	}
}
