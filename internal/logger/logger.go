package logger

import (
	"io"
	"log"
	"strings"

	"github.com/fatih/color"
	"github.com/fujiwara/logutils"
)

//Setup logger
func Setup(out io.Writer, minLevel string) {
	filter := &logutils.LevelFilter{
		Levels: []logutils.LogLevel{"debug", "info", "warn", "error"},
		ModifierFuncs: []logutils.ModifierFunc{
			nil,
			nil,
			logutils.Color(color.FgYellow),
			logutils.Color(color.FgRed, color.BgBlack),
		},
		MinLevel: logutils.LogLevel(strings.ToLower(minLevel)),
		Writer:   out,
	}
	log.SetOutput(filter)
}
