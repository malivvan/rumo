package cui

import (
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
)

// Theme defines the colors used when widgets are initialized.
type Theme struct {
	// Title, border and other lines
	TitleColor    tcell.Color // Box titles.
	BorderColor   tcell.Color // Box borders.
	GraphicsColor tcell.Color // Graphics.

	// Text
	PrimaryTextColor           tcell.Color // Primary text.
	SecondaryTextColor         tcell.Color // Secondary text (e.g. labels).
	TertiaryTextColor          tcell.Color // Tertiary text (e.g. subtitles, notes).
	InverseTextColor           tcell.Color // Text on primary-colored backgrounds.
	ContrastPrimaryTextColor   tcell.Color // Primary text for contrasting elements.
	ContrastSecondaryTextColor tcell.Color // Secondary text on ContrastBackgroundColor-colored backgrounds.

	// Background
	WidgetBackgroundColor       tcell.Color // Main background color for widgets.
	ContrastBackgroundColor     tcell.Color // Background color for contrasting elements.
	MoreContrastBackgroundColor tcell.Color // Background color for even more contrasting elements.

	// Button
	ButtonCursorRune rune // The symbol to draw at the end of button labels when focused.

	// Check box
	CheckBoxCheckedRune rune
	CheckBoxCursorRune  rune // The symbol to draw within the checkbox when focused.

	// Context menu
	ContextMenuPaddingTop    int
	ContextMenuPaddingBottom int
	ContextMenuPaddingLeft   int
	ContextMenuPaddingRight  int

	// Drop down
	DropDownAbbreviationChars string // The chars to show when the option's text gets shortened.
	DropDownSymbol            rune   // The symbol to draw at the end of the field when closed.
	DropDownOpenSymbol        rune   // The symbol to draw at the end of the field when opened.
	DropDownSelectedSymbol    rune   // The symbol to draw to indicate the selected list item.

	// Scroll bar
	ScrollBarColor tcell.Color

	// Window
	WindowMinWidth  int
	WindowMinHeight int
}

// Styles defines the appearance of an application. The default is for a black
// background and some basic colors: black, white, yellow, green, cyan, and
// blue.
var Styles = Theme{
	TitleColor:    color.White,
	BorderColor:   color.White,
	GraphicsColor: color.White,

	PrimaryTextColor:           color.White,
	SecondaryTextColor:         color.Yellow,
	TertiaryTextColor:          color.LimeGreen,
	InverseTextColor:           color.Black,
	ContrastPrimaryTextColor:   color.Black,
	ContrastSecondaryTextColor: color.LightSlateGray,

	WidgetBackgroundColor:       color.Black,
	ContrastBackgroundColor:     color.Green,
	MoreContrastBackgroundColor: color.DarkGreen,

	ButtonCursorRune: '◀',

	CheckBoxCheckedRune: 'X',
	CheckBoxCursorRune:  '◀',

	ContextMenuPaddingTop:    0,
	ContextMenuPaddingBottom: 0,
	ContextMenuPaddingLeft:   1,
	ContextMenuPaddingRight:  1,

	DropDownAbbreviationChars: "...",
	DropDownSymbol:            '◀',
	DropDownOpenSymbol:        '▼',
	DropDownSelectedSymbol:    '▶',

	ScrollBarColor: color.White,

	WindowMinWidth:  4,
	WindowMinHeight: 3,
}
