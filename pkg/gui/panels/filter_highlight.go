package panels

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jesseduffield/lazydocker/pkg/utils"
)

type filterMatchStyle struct {
	foreground string
	bold       bool
	reverse    bool
	underline  bool
}

type ansiStyleState struct {
	foreground string
	bold       bool
	reverse    bool
	underline  bool
}

func highlightFilterMatches(cells []string, score filterItemScore, styleKeys []string) []string {
	style := parseFilterMatchStyle(styleKeys)
	if style.isEmpty() || len(score.terms) == 0 {
		return cells
	}

	positionsByCell := map[int]map[int]struct{}{}
	for _, term := range score.terms {
		cell := term.match.displayCell
		if cell < 0 || cell >= len(cells) {
			continue
		}
		if positionsByCell[cell] == nil {
			positionsByCell[cell] = map[int]struct{}{}
		}
		for _, runeIndex := range term.match.runeIndexes {
			positionsByCell[cell][runeIndex] = struct{}{}
		}
	}

	if len(positionsByCell) == 0 {
		return cells
	}

	highlightedCells := append([]string(nil), cells...)
	for cell, positions := range positionsByCell {
		highlightedCells[cell] = highlightANSIText(cells[cell], positions, style)
	}
	return highlightedCells
}

func highlightANSIText(value string, positions map[int]struct{}, style filterMatchStyle) string {
	if len(positions) == 0 || style.isEmpty() {
		return value
	}

	var result strings.Builder
	result.Grow(len(value) + len(positions)*12)

	baseStyle := ansiStyleState{foreground: "39"}
	matchActive := false
	visibleRuneIndex := 0
	for byteIndex := 0; byteIndex < len(value); {
		if sequence, params, nextIndex, ok := sgrSequenceAt(value, byteIndex); ok {
			result.WriteString(sequence)
			baseStyle.apply(params)
			if matchActive {
				result.WriteString(style.startSequence())
			}
			byteIndex = nextIndex
			continue
		}

		_, marked := positions[visibleRuneIndex]
		if marked && !matchActive {
			result.WriteString(style.startSequence())
			matchActive = true
		}

		_, runeSize := utf8.DecodeRuneInString(value[byteIndex:])
		result.WriteString(value[byteIndex : byteIndex+runeSize])
		byteIndex += runeSize

		_, nextMarked := positions[visibleRuneIndex+1]
		if matchActive && !nextMarked {
			result.WriteString(style.restoreSequence(baseStyle))
			matchActive = false
		}
		visibleRuneIndex++
	}

	if matchActive {
		result.WriteString(style.restoreSequence(baseStyle))
	}
	return result.String()
}

func parseFilterMatchStyle(keys []string) filterMatchStyle {
	style := filterMatchStyle{}
	foregrounds := map[string]string{
		"default": "39",
		"black":   "30",
		"red":     "31",
		"green":   "32",
		"yellow":  "33",
		"blue":    "34",
		"magenta": "35",
		"cyan":    "36",
		"white":   "37",
	}

	for _, key := range keys {
		switch key {
		case "bold":
			style.bold = true
		case "reverse":
			style.reverse = true
		case "underline":
			style.underline = true
		default:
			if foreground, ok := foregrounds[key]; ok {
				style.foreground = foreground
			} else if utils.IsValidHexValue(key) {
				style.foreground = hexForegroundSequence(key)
			}
		}
	}
	return style
}

func hexForegroundSequence(value string) string {
	hexValue := value[1:]
	if len(hexValue) == 3 {
		hexValue = fmt.Sprintf("%c%c%c%c%c%c", hexValue[0], hexValue[0], hexValue[1], hexValue[1], hexValue[2], hexValue[2])
	}
	red, _ := strconv.ParseUint(hexValue[0:2], 16, 8)
	green, _ := strconv.ParseUint(hexValue[2:4], 16, 8)
	blue, _ := strconv.ParseUint(hexValue[4:6], 16, 8)
	return fmt.Sprintf("38;2;%d;%d;%d", red, green, blue)
}

func (style filterMatchStyle) isEmpty() bool {
	return style.foreground == "" && !style.bold && !style.reverse && !style.underline
}

func (style filterMatchStyle) startSequence() string {
	params := []string{}
	if style.foreground != "" {
		params = append(params, style.foreground)
	}
	if style.bold {
		params = append(params, "1")
	}
	if style.reverse {
		params = append(params, "7")
	}
	if style.underline {
		params = append(params, "4")
	}
	return sgrSequence(params)
}

func (style filterMatchStyle) restoreSequence(base ansiStyleState) string {
	params := []string{}
	if style.foreground != "" {
		params = append(params, base.foreground)
	}
	if style.bold {
		// gocui maps SGR 21 to "bold off" (see its escape interpreter).
		params = append(params, enabledSequence(base.bold, "1", "21"))
	}
	if style.reverse {
		params = append(params, enabledSequence(base.reverse, "7", "27"))
	}
	if style.underline {
		params = append(params, enabledSequence(base.underline, "4", "24"))
	}
	return sgrSequence(params)
}

func enabledSequence(enabled bool, on, off string) string {
	if enabled {
		return on
	}
	return off
}

func sgrSequence(params []string) string {
	if len(params) == 0 {
		return ""
	}
	return "\x1b[" + strings.Join(params, ";") + "m"
}

func sgrSequenceAt(value string, byteIndex int) (sequence, params string, nextIndex int, ok bool) {
	if byteIndex+2 > len(value) || value[byteIndex] != '\x1b' || value[byteIndex+1] != '[' {
		return "", "", byteIndex, false
	}
	for index := byteIndex + 2; index < len(value); index++ {
		if value[index] < 0x40 || value[index] > 0x7e {
			continue
		}
		if value[index] != 'm' {
			return "", "", byteIndex, false
		}
		return value[byteIndex : index+1], value[byteIndex+2 : index], index + 1, true
	}
	return "", "", byteIndex, false
}

func (state *ansiStyleState) apply(params string) {
	if params == "" {
		params = "0"
	}
	values := strings.Split(params, ";")
	for index := 0; index < len(values); index++ {
		value, err := strconv.Atoi(values[index])
		if err != nil {
			continue
		}
		switch {
		case value == 0:
			*state = ansiStyleState{foreground: "39"}
		case value == 1:
			state.bold = true
		case value == 4:
			state.underline = true
		case value == 7:
			state.reverse = true
		case value == 21:
			state.bold = false
		case value == 24:
			state.underline = false
		case value == 27:
			state.reverse = false
		case value >= 30 && value <= 37, value >= 90 && value <= 97:
			state.foreground = values[index]
		case value == 38 && index+1 < len(values):
			parameterCount := 0
			switch values[index+1] {
			case "2":
				parameterCount = 5
			case "5":
				parameterCount = 3
			}
			if parameterCount > 0 && index+parameterCount <= len(values) {
				state.foreground = strings.Join(values[index:index+parameterCount], ";")
				index += parameterCount - 1
			}
		case value == 39:
			state.foreground = "39"
		}
	}
}
