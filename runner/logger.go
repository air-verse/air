package runner

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
)

// TODO: support more colors
var colorMap = map[string]color.Attribute{
	"red":     color.FgRed,
	"green":   color.FgGreen,
	"yellow":  color.FgYellow,
	"blue":    color.FgBlue,
	"magenta": color.FgMagenta,
	"cyan":    color.FgCyan,
	"white":   color.FgWhite,
}

type logFunc func(string, ...interface{})

type logger struct {
	colors  map[string]string
	loggers map[string]logFunc
}

func newLogger(cfg *config) *logger {
	colors := cfg.colorInfo()
	loggers := make(map[string]logFunc, len(colors))
	for name, nameColor := range colors {
		loggers[name] = newLogFunc(nameColor)
	}
	loggers["default"] = defaultLogger()
	return &logger{
		colors:  colors,
		loggers: loggers,
	}
}

func newLogFunc(nameColor string) logFunc {
	return func(msg string, v ...interface{}) {
		t := time.Now().Format("15:04:05.000")
		fmtStr := "[%s] %s\n"
		format := fmt.Sprintf(fmtStr, t, msg)
		color.New(getColor(nameColor)).Fprintf(os.Stdout, format, v...)
	}
}

func getColor(name string) color.Attribute {
	if v, ok := colorMap[name]; ok {
		return v
	}
	return color.FgWhite
}

func (l *logger) main() logFunc {
	return l.getLogger("main")
}

func (l *logger) build() logFunc {
	return l.getLogger("build")
}

func (l *logger) runner() logFunc {
	return l.getLogger("runner")
}

func (l *logger) watcher() logFunc {
	return l.getLogger("watcher")
}

func (l *logger) app() logFunc {
	return l.getLogger("app")
}

func defaultLogger() logFunc {
	return newLogFunc("white")
}

func (l *logger) getLogger(name string) logFunc {
	v, ok := l.loggers[name]
	if !ok {
		dft, _ := l.loggers["default"]
		return dft
	}
	return v
}

type appLogWriter struct {
	l logFunc
}

func newAppLogWriter(l *logger) appLogWriter {
	return appLogWriter{
		l: l.app(),
	}
}

func (w appLogWriter) Write(data []byte) (n int, err error) {
	w.l(string(data))
	return len(data), nil
}
