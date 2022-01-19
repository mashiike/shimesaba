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
		Levels: []logutils.LogLevel{"debug", "info", "notice", "warn", "error"},
		ModifierFuncs: []logutils.ModifierFunc{
			logutils.Color(color.FgHiBlack),
			nil,
			logutils.Color(color.FgHiBlue),
			logutils.Color(color.FgYellow),
			logutils.Color(color.FgRed, color.BgBlack),
		},
		MinLevel: logutils.LogLevel(strings.ToLower(minLevel)),
		Writer:   out,
	}
	log.SetOutput(filter)
}
