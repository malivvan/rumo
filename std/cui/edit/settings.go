package edit

// DefaultLocalSettings returns the default local settings
// Note that filetype is a local only option
func DefaultLocalSettings() map[string]interface{} {
	return map[string]interface{}{
		"autoindent":       true,
		"autosave":         false,
		"basename":         false,
		"colorcolumn":      float64(0),
		"cursorline":       true,
		"eofnewline":       false,
		"fastdirty":        true,
		"fileformat":       "unix",
		"filetype":         "Unknown",
		"hidecursoronblur": false,
		"hidehelp":         false,
		"indentchar":       " ",
		"keepautoindent":   false,
		"matchbrace":       false,
		"matchbraceleft":   false,
		"rmtrailingws":     false,
		"ruler":            true,
		"savecursor":       false,
		"saveundo":         false,
		"scrollbar":        false,
		"scrollmargin":     float64(3),
		"scrollspeed":      float64(2),
		"showwhitespace":   false,
		"smartpaste":       true,
		"softwrap":         false,
		"splitbottom":      true,
		"splitright":       true,
		"statusline":       true,
		"syntax":           true,
		"tabmovement":      false,
		"tabsize":          float64(4),
		"tabstospaces":     false,
		"useprimary":       true,
	}
}
