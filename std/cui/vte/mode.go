package vte

type mode int

const (
	// ANSI-Standardized modes
	//
	// Keyboard Action mode
	kam mode = 1 << iota
	// Insert/Replace mode
	irm
	// Send/Receive mode
	srm
	// Line feed/new line mode
	lnm

	// ANSI-Compatible DEC Private Modes
	//
	// Cursor Key mode
	decckm
	// ANSI/VT52 mode
	decanm
	// Column mode
	deccolm
	// Scroll mode
	decsclm
	// Origin mode
	decom
	// Autowrap mode
	decawm
	// Autorepeat mode
	decarm
	// Printer form feed mode
	decpff
	// Printer extent mode
	decpex
	// Text Cursor Enable mode
	dectcem
	// National replacement character sets
	decnrcm

	// xterm
	//
	// Use alternate screen
	smcup
	// Bracketed paste
	paste
	// vt220 mouse
	mouseButtons
	// vt220 + drag
	mouseDrag
	// vt220 + all motion
	mouseMotion
	// Mouse SGR mode
	mouseSGR
	// Alternate scroll
	altScroll
	// Focus event reporting (DECSET 1004)
	focusEvents
	// Synchronized output (DECSET 2026)
	syncOutput
	// Reverse video (DECSCNM, DECSET 5)
	decscnm
	// X10 mouse compatibility (DECSET 9)
	mouseX10
	// UTF-8 mouse encoding (DECSET 1005)
	mouseUTF8
)

func (vt *VT) enterAltScreen(saveCursor bool) {
	if saveCursor {
		vt.decsc()
	}
	vt.activeScreen = vt.altScreen
	vt.mode |= smcup
	// Enable altScroll in the alt screen. This is only used
	// if the application doesn't enable mouse.
	vt.mode |= altScroll
}

func (vt *VT) exitAltScreen(restoreCursor bool) {
	if vt.mode&smcup != 0 {
		// Only clear if we were in the alternate screen.
		vt.ed(2)
	}
	vt.activeScreen = vt.primaryScreen
	vt.mode &^= smcup
	vt.mode &^= altScroll
	if restoreCursor {
		vt.decrc()
	}
}

func (vt *VT) sm(params []int) {
	for _, param := range params {
		switch param {
		case 2:
			vt.mode |= kam
		case 4:
			vt.mode |= irm
		case 12:
			vt.mode |= srm
		case 20:
			vt.mode |= lnm
		}
	}
}

func (vt *VT) rm(params []int) {
	for _, param := range params {
		switch param {
		case 2:
			vt.mode &^= kam
		case 4:
			vt.mode &^= irm
		case 12:
			vt.mode &^= srm
		case 20:
			vt.mode &^= lnm
		}
	}
}

func (vt *VT) decset(params []int) {
	for _, param := range params {
		switch param {
		case 1:
			vt.mode |= decckm
		case 2:
			vt.mode |= decanm
		case 3:
			vt.mode |= deccolm
		case 4:
			vt.mode |= decsclm
		case 5:
			vt.mode |= decscnm
		case 6:
			vt.mode |= decom
			vt.homeCursor()
		case 7:
			vt.mode |= decawm
			vt.lastCol = false
		case 8:
			vt.mode |= decarm
		case 9:
			vt.mode |= mouseX10
		case 25:
			vt.mode |= dectcem
		case 47, 1047:
			vt.enterAltScreen(false)
		case 1048:
			vt.decsc()
		case 1000:
			vt.mode |= mouseButtons
		case 1002:
			vt.mode |= mouseDrag
		case 1003:
			vt.mode |= mouseMotion
		case 1004:
			vt.mode |= focusEvents
		case 1005:
			vt.mode |= mouseUTF8
		case 1006:
			vt.mode |= mouseSGR
		case 1007:
			vt.mode |= altScroll
		case 1049:
			vt.enterAltScreen(true)
		case 2004:
			vt.mode |= paste
		case 2026:
			vt.mode |= syncOutput
		}
	}
}

func (vt *VT) decrst(params []int) {
	for _, param := range params {
		switch param {
		case 1:
			vt.mode &^= decckm
		case 2:
			vt.mode &^= decanm
		case 3:
			vt.mode &^= deccolm
		case 4:
			vt.mode &^= decsclm
		case 5:
			vt.mode &^= decscnm
		case 6:
			vt.mode &^= decom
			vt.homeCursor()
		case 7:
			vt.mode &^= decawm
			vt.lastCol = false
		case 8:
			vt.mode &^= decarm
		case 9:
			vt.mode &^= mouseX10
		case 25:
			vt.mode &^= dectcem
		case 47, 1047:
			vt.exitAltScreen(false)
		case 1048:
			vt.decrc()
		case 1000:
			vt.mode &^= mouseButtons
		case 1002:
			vt.mode &^= mouseDrag
		case 1003:
			vt.mode &^= mouseMotion
		case 1004:
			vt.mode &^= focusEvents
		case 1005:
			vt.mode &^= mouseUTF8
		case 1006:
			vt.mode &^= mouseSGR
		case 1007:
			vt.mode &^= altScroll
		case 1049:
			vt.exitAltScreen(true)
		case 2004:
			vt.mode &^= paste
		case 2026:
			vt.mode &^= syncOutput
		}
	}
}
