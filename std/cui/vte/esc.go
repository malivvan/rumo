package vte

import "github.com/gdamore/tcell/v3"

func (vt *VT) esc(esc string) {
	switch esc {
	case "7":
		vt.decsc()
	case "8":
		vt.decrc()
	case "D":
		vt.ind()
	case "E":
		vt.nel()
	case "H":
		vt.hts()
	case "M":
		vt.ri()
	case "N":
		vt.charsets.saved = vt.charsets.selected
		vt.charsets.singleShift = true
		vt.charsets.selected = g2
	case "O":
		vt.charsets.saved = vt.charsets.selected
		vt.charsets.singleShift = true
		vt.charsets.selected = g3
	case "=":
		// DECKPAM
	case ">":
		// DECKPNM
	case "6":
		// DECBI — Back Index
		vt.decbi()
	case "9":
		// DECFI — Forward Index
		vt.decfi()
	case "c":
		vt.ris()
	case "(0":
		vt.charsets.designations[g0] = decSpecialAndLineDrawing
	case ")0":
		vt.charsets.designations[g1] = decSpecialAndLineDrawing
	case "*0":
		vt.charsets.designations[g2] = decSpecialAndLineDrawing
	case "+0":
		vt.charsets.designations[g3] = decSpecialAndLineDrawing
	case "(B":
		vt.charsets.designations[g0] = ascii
	case ")B":
		vt.charsets.designations[g1] = ascii
	case "*B":
		vt.charsets.designations[g2] = ascii
	case "+B":
		vt.charsets.designations[g3] = ascii
	case "#3":
		// DECDHL - Double-Height Line (Top Half) - not supported
	case "#4":
		// DECDHL - Double-Height Line (Bottom Half) - not supported
	case "#5":
		// DECSWL - Single-Width Line - not supported (default behavior)
	case "#6":
		// DECDWL - Double-Width Line - not supported
	case "#8":
		// DECALN - Fill the screen with capital Es
		for r := row(0); r < row(vt.height()); r++ {
			for c := column(0); c < column(vt.width()); c++ {
				vt.activeScreen[r][c].content = 'E'
				vt.activeScreen[r][c].width = 1
				vt.activeScreen[r][c].attrs = tcell.StyleDefault
				vt.activeScreen[r][c].combining = nil
				vt.activeScreen[r][c].wrapped = false
			}
		}
		// Reset margins and cursor per spec
		vt.margin.top = 0
		vt.margin.bottom = row(vt.height()) - 1
		vt.margin.left = 0
		vt.margin.right = column(vt.width()) - 1
		vt.cursor.row = 0
		vt.cursor.col = 0
	}
}

// Index ESC-D
func (vt *VT) ind() {
	vt.lastCol = false
	if vt.cursor.row == vt.margin.bottom {
		vt.scrollUp(1)
		return
	}
	if vt.cursor.row >= row(vt.height()-1) {
		// don't let row go beyond the height

		return
	}
	vt.cursor.row += 1
}

// Next line ESC-E
// Moves cursor to the left margin of the next line, scrolling if necessary
func (vt *VT) nel() {
	vt.ind()
	vt.cursor.col = vt.margin.left
}

// Horizontal tab set ESC-H
func (vt *VT) hts() {
	col := vt.cursor.col
	// Find sorted insertion point
	i := 0
	for i < len(vt.tabStop) {
		if vt.tabStop[i] == col {
			return // already exists
		}
		if vt.tabStop[i] > col {
			break
		}
		i++
	}
	// Insert at position i
	vt.tabStop = append(vt.tabStop, 0)
	copy(vt.tabStop[i+1:], vt.tabStop[i:])
	vt.tabStop[i] = col
}

// Reverse Index ESC-M
func (vt *VT) ri() {
	vt.lastCol = false
	if vt.cursor.row < 0 {
		return
	}
	if vt.cursor.row == vt.margin.top {
		vt.scrollDown(1)
		return
	}
	vt.cursor.row -= 1
}

// Save Cursor DECSC ESC-7
func (vt *VT) decsc() {
	state := cursorState{
		cursor: vt.cursor,
		decawm: vt.mode&decawm != 0,
		decom:  vt.mode&decom != 0,
		charsets: charsets{
			selected: vt.charsets.selected,
			saved:    vt.charsets.saved,
			designations: map[charsetDesignator]charset{
				g0: vt.charsets.designations[g0],
				g1: vt.charsets.designations[g1],
				g2: vt.charsets.designations[g2],
				g3: vt.charsets.designations[g3],
			},
		},
	}
	switch {
	case vt.mode&smcup != 0:
		// We are in alt screen
		vt.altState = state
	default:
		vt.primaryState = state
	}
}

// Restore Cursor DECRC ESC-8
func (vt *VT) decrc() {
	var state cursorState
	switch {
	case vt.mode&smcup != 0:
		// In the alt screen
		state = vt.altState
	default:
		state = vt.primaryState
	}

	vt.cursor = state.cursor
	vt.charsets = charsets{
		selected: state.charsets.selected,
		saved:    state.charsets.saved,
		designations: map[charsetDesignator]charset{
			g0: state.charsets.designations[g0],
			g1: state.charsets.designations[g1],
			g2: state.charsets.designations[g2],
			g3: state.charsets.designations[g3],
		},
	}

	switch state.decawm {
	case true:
		vt.mode |= decawm
	case false:
		vt.mode &^= decawm
	}

	switch state.decom {
	case true:
		vt.mode |= decom
	case false:
		vt.mode &^= decom
	}
}

// Back Index (DECBI) ESC-6
// If the cursor is at the left margin, scroll the content within the margins
// to the right by one column. Otherwise, move the cursor left one column.
func (vt *VT) decbi() {
	if vt.cursor.col == vt.margin.left {
		// Scroll right: shift columns right within margins, blank the left margin column
		for r := vt.margin.top; r <= vt.margin.bottom; r++ {
			for c := vt.margin.right; c > vt.margin.left; c-- {
				vt.activeScreen[r][c] = vt.activeScreen[r][c-1]
			}
			vt.activeScreen[r][vt.margin.left].erase(vt.cursor.attrs)
		}
	} else if vt.cursor.col > 0 {
		vt.cursor.col--
	}
}

// Forward Index (DECFI) ESC-9
// If the cursor is at the right margin, scroll the content within the margins
// to the left by one column. Otherwise, move the cursor right one column.
func (vt *VT) decfi() {
	if vt.cursor.col == vt.margin.right {
		// Scroll left: shift columns left within margins, blank the right margin column
		for r := vt.margin.top; r <= vt.margin.bottom; r++ {
			for c := vt.margin.left; c < vt.margin.right; c++ {
				vt.activeScreen[r][c] = vt.activeScreen[r][c+1]
			}
			vt.activeScreen[r][vt.margin.right].erase(vt.cursor.attrs)
		}
	} else if vt.cursor.col < column(vt.width()-1) {
		vt.cursor.col++
	}
}

// Reset Initial State (RIS) ESC-c
func (vt *VT) ris() {
	w := vt.width()
	h := vt.height()
	vt.altScreen = make([][]cell, h)
	vt.primaryScreen = make([][]cell, h)
	for i := range vt.altScreen {
		vt.altScreen[i] = make([]cell, w)
		vt.primaryScreen[i] = make([]cell, w)
	}
	vt.margin.bottom = row(h) - 1
	vt.margin.right = column(w) - 1
	vt.cursor.row = 0
	vt.cursor.col = 0
	vt.lastCol = false
	vt.activeScreen = vt.primaryScreen
	vt.charsets = charsets{
		selected: 0,
		saved:    0,
		designations: map[charsetDesignator]charset{
			g0: ascii,
			g1: ascii,
			g2: ascii,
			g3: ascii,
		},
	}
	vt.mode = decawm | dectcem
	vt.tabStop = []column{}
	for i := 7; i < (50 * 7); i += 8 {
		vt.tabStop = append(vt.tabStop, column(i))
	}
}
