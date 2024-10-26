package verisure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/nav-inc/datetime"
	"io"
	_ "log"
	"strings"
	"time"
)

type LogItem struct {
	Date     time.Time      `json:"date"`
	Level    Severity       `json:"severity"`
	Message  string         `json:"msg"`
	Source   map[string]any `json:"source"`
	Extended map[string]any `json:"extended"`
}

func (l LogItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"date":     l.Date,
		"severity": l.Level,
		"msg":      l.Message,
	})
}

func (l *LogItem) UnmarshalJSON(data []byte) error {

	var itemMap map[string]any

	location, _ := time.LoadLocation("Europe/Stockholm")

	err := json.Unmarshal(data, &itemMap)
	if err != nil {
		return err
	}

	if dateString, ok := itemMap["date"]; ok {
		if l.Date, err = StringToISOTime(dateString.(string), location); err != nil {
			l.Date = time.Time{}
		}
	}

	if severityString, ok := itemMap["severity"]; ok {
		if l.Level, err = NewSeverity(severityString.(string)); err != nil {
			l.Level = ALL_LEVELS
		}
	}

	l.Message = itemMap["msg"].(string)

	if extendedMap, ok := itemMap["extended"]; ok {
		l.Extended = extendedMap.(map[string]any)
	}

	if sourceMap, ok := itemMap["source"]; ok {
		l.Source = sourceMap.(map[string]any)
	}

	return nil
}

func NewLogItem(jsonString string) (LogItem, error) {
	var logItem LogItem
	err := json.Unmarshal([]byte(jsonString), &logItem)
	if err != nil {
		return logItem, err
	}
	return logItem, nil
}

func (item *LogItem) Print(out io.Writer) {
	// Write Date
	location, _ := time.LoadLocation("Europe/Stockholm")
	printProperty(out, "date", ISOTimeToString(item.Date, location))

	// Write Colored Severity
	printProperty(out, "severity", printLevelColor(item.Level))

	// Write source map
	if item.Source != nil {
		printMap(out, "source", item.Source)
	}

	// Write extended map
	if item.Extended != nil {
		printMap(out, "extended", item.Extended)
	}

	// Write message
	printProperty(out, "message", item.Message)

	fmt.Fprint(out, "\n")
}

func printMap(out io.Writer, mapName string, xMap map[string]interface{}) {
	var buf bytes.Buffer
	itemArray := []string{}
	for key, value := range xMap {
		itemArray = append(itemArray, fmt.Sprintf("%s: %v", printItalic(key), value))
	}
	buf.WriteString(strings.Join(itemArray, " | "))
	fmt.Fprintf(out, " %18s: [ %s ]\n", fmt.Sprintf("%s%s", "-", printUnderline(mapName)), buf.String())
}

func printProperty(out io.Writer, key string, value string) {
	fmt.Fprintf(out, " %18s: %s\n", fmt.Sprintf("%s%s", "-", printUnderline(key)), value)
}

var printUnderline = color.New(color.Underline).SprintfFunc()

var printItalic = color.New(color.Italic).SprintfFunc()

func StringToISOTime(timeString string, location *time.Location) (time.Time, error) {
	timestamp, err := datetime.Parse(timeString, location)
	if err != nil {
		return time.Time{}, err
	}
	return timestamp.In(location), nil
}

func ISOTimeToString(time time.Time, location *time.Location) string {
	return time.In(location).Format("2006-01-02T15:04:05.999Z")
}

func printLevelColor(severity Severity) string {
	var levelColor *color.Color
	severityString := string(severity)
	switch strings.ToLower(severityString) {
	case "debug":
		levelColor = color.New(color.FgMagenta)
	case "info":
		levelColor = color.New(color.FgBlue)
	case "informational":
		levelColor = color.New(color.FgBlue)
	case "warn":
		levelColor = color.New(color.FgYellow)
	case "warning":
		levelColor = color.New(color.FgYellow)
	case "error":
		levelColor = color.New(color.FgRed)
	case "dpanic":
		levelColor = color.New(color.FgRed)
	case "panic":
		levelColor = color.New(color.FgRed)
	case "fatal":
		levelColor = color.New(color.FgCyan)
	case "critical":
		levelColor = color.New(color.FgCyan)
	default:
		return severityString
	}
	return levelColor.SprintFunc()(severityString)
}
