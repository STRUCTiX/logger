package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

func NewStandardOutput(file *os.File) OutputWriter {
	var writer = StandardWriter{
		ColorsEnabled: true,
		Target:        file,
	}

	defaultOutputSettings := parseVerbosityLevel(os.Getenv("LOG_LEVEL"))
	logEnv := "*"
	if os.Getenv("LOG") != "" {
		logEnv = os.Getenv("LOG")
	}
	writer.Settings = parsePackageSettings(logEnv, defaultOutputSettings)

	return writer
}

type StandardWriter struct {
	ColorsEnabled bool
	Target        *os.File
	Settings      map[string]*OutputSettings
}

func (standardWriter StandardWriter) Init() {}

func (standardWriter StandardWriter) Write(log *Log) {
	if standardWriter.IsEnabled(log.Package, log.Level) {
		fmt.Fprintln(standardWriter.Target, standardWriter.Format(log))
	}
}

func (standardWriter *StandardWriter) IsEnabled(logger, level string) bool {
	settings := standardWriter.LoggerSettings(logger)

	if level == "INFO" {
		return settings.Info
	}

	if level == "ERROR" {
		return settings.Error
	}

	if level == "TIMER" {
		return settings.Timer
	}

	return false
}

func (standardWriter *StandardWriter) LoggerSettings(p string) *OutputSettings {
	if settings, ok := standardWriter.Settings[p]; ok {
		return settings
	}

	// If there is a "*" (Select all) setting, return that
	if settings, ok := standardWriter.Settings["*"]; ok {
		return settings
	}

	return muted
}

func (standardWriter *StandardWriter) Format(log *Log) string {
	if standardWriter.ColorsEnabled {
		return standardWriter.PrettyFormat(log)
	} else {
		return standardWriter.JSONFormat(log)
	}
}

func (standardWriter *StandardWriter) JSONFormat(log *Log) string {
	str, err := json.Marshal(log)
	if err != nil {
		return fmt.Sprintf(`{ "logger-error": "%v" }`, err)
	}

	return string(str)
}

func (standardWriter *StandardWriter) PrettyFormat(log *Log) string {
	return fmt.Sprintf("%s %s %s%s",
		time.Now().Format("15:04:05.000"),
		standardWriter.PrettyLabel(log),
		log.Message,
		standardWriter.PrettyAttrs(log.Attrs))
}

func (standardWriter *StandardWriter) PrettyAttrs(attrs *Attrs) string {
	if attrs == nil {
		return ""
	}

	result := ""
	for key, val := range *attrs {
		result = fmt.Sprintf("%s %s=%v", result, key, val)
	}

	return result
}

func (standardWriter *StandardWriter) PrettyLabel(log *Log) string {
	return fmt.Sprintf("%s%s%s:%s",
		colorFor(log.Package),
		log.Package,
		standardWriter.PrettyLabelExt(log),
		reset)
}

func (standardWriter *StandardWriter) PrettyLabelExt(log *Log) string {
	if log.Level == "ERROR" {
		return fmt.Sprintf("(%s!%s)", red, colorFor(log.Package))
	}

	if log.Level == "TIMER" {
		return fmt.Sprintf("(%s%s%s)", reset, fmt.Sprintf("%v", time.Duration(log.ElapsedNano)), colorFor(log.Package))
	}

	return ""
}

// Accepts: foo,bar,qux@timer
//          *
//          *@error
//          *@error,database@timer
func parsePackageSettings(input string, defaultOutputSettings *OutputSettings) map[string]*OutputSettings {
	all := map[string]*OutputSettings{}
	items := strings.Split(input, ",")

	for _, item := range items {
		name, verbosity := parsePackageName(item)
		if verbosity == nil {
			verbosity = defaultOutputSettings
		}

		all[name] = verbosity
	}

	return all
}

// Accepts: users
//          database@timer
//          server@error
func parsePackageName(input string) (string, *OutputSettings) {
	parsed := strings.Split(input, "@")
	name := strings.TrimSpace(parsed[0])

	if len(parsed) > 1 {
		return name, parseVerbosityLevel(parsed[1])
	}

	return name, nil
}

func parseVerbosityLevel(val string) *OutputSettings {
	val = strings.ToUpper(strings.TrimSpace(val))

	if val == "MUTE" {
		return &OutputSettings{}
	}

	s := &OutputSettings{
		Info:  true,
		Timer: true,
		Error: true,
	}

	if val == "TIMER" {
		s.Info = false
	}

	if val == "ERROR" {
		s.Info = false
		s.Timer = false
	}

	return s
}
