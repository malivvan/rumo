package module

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

// Export of a Module Value for documentation.
type Export struct {
	Name    string     // Name of the function.
	Type    int        // Type of the value.
	Inputs  [][]string // (Function only) Inputs holds the list of inputs. Each entry holds a name of the argument followed by its possible types.
	Output  []string   // (Function only) Output holds the name of the output value followed by its possible types.
	Comment string     // Comment holds the comment for the value, if any.
}

func (export *Export) String() string {
	return fmt.Sprintf("%s(%v) (%v) %s", export.Name, export.Inputs, export.Output, export.Comment)
}

// ParseExport parses a string into an Export struct. The expected format is:
//
//	<name>(<input1>, <input2>, ...) (<output>) <comment>
//
// where:
//   - <name> is the name of the function.
//   - <input> is in the format "<argName> <type1|type2|...>".
//   - <output> is in the format "<argName> <type1|type2|...>".
//   - <comment> is any additional information about the function.
func ParseExport(s string) *Export {
	if matches := funcInfoRe.FindStringSubmatch(s); matches != nil && len(matches) > 0 {
		return &Export{Name: matches[1], Inputs: parseParams(matches[2]), Output: parseParam(matches[4]), Comment: matches[5]}
	} else if matches := constInfoRe.FindStringSubmatch(s); matches != nil && len(matches) > 0 {
		switch matches[2] {
		case "int":
			return &Export{Name: matches[1], Comment: matches[3]}
		case "float":
			return &Export{Name: matches[1], Comment: matches[3]}
		case "string":
			return &Export{Name: matches[1], Comment: matches[3]}
		case "bool":
			return &Export{Name: matches[1], Comment: matches[3]}
		case "bytes":
			return &Export{Name: matches[1], Comment: matches[3]}
		case "char":
			return &Export{Name: matches[1], Comment: matches[3]}
		case "time":
			return &Export{Name: matches[1], Comment: matches[3]}
		default:
			panic(fmt.Errorf("unexpected type: %s", matches[2]))
		}
	} else if matches := bareNameRe.FindStringSubmatch(s); matches != nil && len(matches) > 0 {
		return &Export{Name: matches[1], Comment: strings.TrimSpace(matches[2])}
	}
	panic(fmt.Errorf("unexpected export format: %s", s))
}

// ParseExports parses a string containing multiple exports into a slice of Export structs.
// The function handles both single-line comments (starting with "//") and multi-line comments (enclosed in "/* ... */").
// The expected format for each export is the same as described in ParseExport, and comments can be associated with the export either before or after the export definition.
// For example:
//
//	// This is a single-line comment for the export
//	myFunc(arg1 int, arg2 string) (result int) This function does something.
//
//	/* This is a multi-line comment for the export
//	   It can span multiple lines.
//	*/
//	anotherFunc(arg1 float) (result float) This function does something else.
func ParseExports(s string) (fns []*Export, err error) {
	var info *Export
	state := 0
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		startCommentIdx := strings.Index(line, "/*")
		if startCommentIdx != -1 {
			line = strings.TrimSpace(line[startCommentIdx+2:])
			state = 1
		}
		endCommentIdx := strings.Index(line, "*/")
		if endCommentIdx != -1 {
			line = strings.TrimSpace(line[:endCommentIdx])
			state = 2
		}
		if state > 0 {
			if info == nil {
				info = ParseExport(line[:])
			} else {
				info.Comment += "\n" + strings.TrimSpace(line[:])
			}
			if state == 2 {
				fns = append(fns, info)
				info = nil
				state = 0
			}
		} else if strings.HasPrefix(line, "//") {
			if info == nil {
				info = ParseExport(line[2:])
			} else {
				info.Comment += "\n" + strings.TrimSpace(line[2:])
			}
		} else {
			if info != nil {
				fns = append(fns, info)
				info = nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return fns, nil
}

var constInfoRe = regexp.MustCompile(`(?m)^\s*([[:word:]]+)\s*(int|float|string|bool|bytes|char|time)\s*([[:print:]]*?)\s*$`)
var funcInfoRe = regexp.MustCompile(`(?m)^\s*([[:word:]]+)\s*\(([[:print:]]*?)\)\s*(\(([[:print:]]*?)\))?\s*([[:print:]]*?)\s*$`)
var bareNameRe = regexp.MustCompile(`(?m)^\s*([[:word:]]+)\s*([[:print:]]*?)\s*$`)

func parseTypes(s string) (types []string) {
	for _, t := range strings.Split(s, "|") {
		types = append(types, strings.ToLower(strings.TrimSpace(t)))
	}
	return types
}

func parseParams(s string) (params [][]string) {
	for _, input := range strings.Split(s, ",") {
		param := parseParam(input)
		if param != nil {
			params = append(params, parseParam(input))
		}
	}
	return params
}

func parseParam(s string) []string {
	s = strings.TrimSpace(s)
	idx := strings.Index(s, " ")
	if idx == -1 {
		return nil
	}
	return append([]string{s[:idx]}, parseTypes(s[idx+1:])...)
}
