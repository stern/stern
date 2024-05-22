package stern

import (
	"errors"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

var colorList = [][2]*color.Color{
	{color.New(color.FgHiCyan), color.New(color.FgCyan)},
	{color.New(color.FgHiGreen), color.New(color.FgGreen)},
	{color.New(color.FgHiMagenta), color.New(color.FgMagenta)},
	{color.New(color.FgHiYellow), color.New(color.FgYellow)},
	{color.New(color.FgHiBlue), color.New(color.FgBlue)},
	{color.New(color.FgHiRed), color.New(color.FgRed)},
}

func SetColorList(podColors, containerColors []string) error {
	colors, err := parseColors(podColors, containerColors)
	if err != nil {
		return err
	}
	colorList = colors
	return nil
}

func parseColors(podColors, containerColors []string) ([][2]*color.Color, error) {
	if len(podColors) == 0 {
		return nil, errors.New("pod-colors must not be empty")
	}
	if len(containerColors) == 0 {
		// if containerColors is empty, use podColors as containerColors
		return createColorPairs(podColors, podColors)
	}
	if len(containerColors) != len(podColors) {
		return nil, errors.New("pod-colors and container-colors must have the same length")
	}
	return createColorPairs(podColors, containerColors)
}

func createColorPairs(podColors, containerColors []string) ([][2]*color.Color, error) {
	colorList := make([][2]*color.Color, 0, len(podColors))
	for i := 0; i < len(podColors); i++ {
		podColor, err := sgrSequenceToColor(podColors[i])
		if err != nil {
			return nil, err
		}
		containerColor, err := sgrSequenceToColor(containerColors[i])
		if err != nil {
			return nil, err
		}
		colorList = append(colorList, [2]*color.Color{podColor, containerColor})
	}
	return colorList, nil
}

// sgrSequenceToColor converts a string representing SGR sequence
// separated by ";" into a *color.Color instance.
// For example, "31;4" means red foreground with underline.
// https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_(Select_Graphic_Rendition)_parameters
func sgrSequenceToColor(s string) (*color.Color, error) {
	parts := strings.Split(s, ";")
	attrs := make([]color.Attribute, 0, len(parts))
	for _, part := range parts {
		attr, err := strconv.ParseInt(strings.TrimSpace(part), 10, 32)
		if err != nil {
			return nil, err
		}
		attrs = append(attrs, color.Attribute(attr))
	}
	return color.New(attrs...), nil
}
