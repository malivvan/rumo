package shell

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"

	"github.com/malivvan/rumo/std/shell/platform"
	"github.com/malivvan/rumo/std/shell/runes"
)

var (
	ErrInterrupt = errors.New("Interrupt")
)

type operation struct {
	m       sync.Mutex
	t       *terminal
	buf     *runeBuffer
	wrapOut atomic.Pointer[wrapWriter]
	wrapErr atomic.Pointer[wrapWriter]

	isPrompting bool // true when prompt written and waiting for input

	history   *opHistory
	search    *opSearch
	completer *opCompleter
	vim       *opVim
	undo      *opUndo
}

func (op *operation) SetBuffer(what string) {
	op.buf.SetNoRefresh([]rune(what))
}

type wrapWriter struct {
	o      *operation
	target io.Writer
}

func (w *wrapWriter) Write(b []byte) (int, error) {
	return w.o.write(w.target, b)
}

func (op *operation) write(target io.Writer, b []byte) (int, error) {
	op.m.Lock()
	defer op.m.Unlock()

	if !op.isPrompting {
		return target.Write(b)
	}

	var (
		n   int
		err error
	)
	op.buf.Refresh(func() {
		n, err = target.Write(b)
		// Adjust the prompt start position by b
		rout := runes.ColorFilter([]rune(string(b[:])))
		tWidth, _ := op.t.GetWidthHeight()
		sp := runes.SplitByLine(rout, []rune{}, op.buf.ppos, tWidth, 1)
		if len(sp) > 1 {
			op.buf.ppos = len(sp[len(sp)-1])
		} else {
			op.buf.ppos += len(rout)
		}
	})

	op.search.RefreshIfNeeded()
	if op.completer.IsInCompleteMode() {
		op.completer.CompleteRefresh()
	}
	return n, err
}

func newOperation(t *terminal) *operation {
	cfg := t.GetConfig()
	op := &operation{
		t:   t,
		buf: newRuneBuffer(t),
	}
	_, _ = op.SetConfig(cfg)
	op.vim = newVimMode(op)
	op.completer = newOpCompleter(op.buf.w, op)
	cfg.FuncOnWidthChanged(t.OnSizeChange)
	return op
}

func (op *operation) GetConfig() *Config {
	return op.t.GetConfig()
}

func (op *operation) readline(deadline chan struct{}) ([]rune, error) {
	isTyping := false // don't add new undo entries during normal typing

	for {
		keepInSearchMode := false
		keepInCompleteMode := false
		r, err := op.t.GetRune(deadline)

		if cfg := op.GetConfig(); cfg.FuncFilterInputRune != nil && err == nil {
			var process bool
			r, process = cfg.FuncFilterInputRune(r)
			if !process {
				op.buf.Refresh(nil) // to refresh the line
				continue            // ignore this rune
			}
		}

		if err == io.EOF {
			if op.buf.Len() == 0 {
				op.buf.Clean()
				return nil, io.EOF
			} else {
				// if stdin got io.EOF and there is something left in buffer,
				// let's flush them by sending CharEnter.
				// And we will got io.EOF int next loop.
				r = CharEnter
			}
		} else if err != nil {
			return nil, err
		}
		isUpdateHistory := true

		if op.completer.IsInCompleteSelectMode() {
			keepInCompleteMode = op.completer.HandleCompleteSelect(r)
			if keepInCompleteMode {
				continue
			}

			op.buf.Refresh(nil)
			switch r {
			case CharEnter, CharCtrlJ:
				_ = op.history.Update(op.buf.Runes(), false)
				fallthrough
			case CharInterrupt:
				fallthrough
			case CharBell:
				continue
			}
		}

		if op.vim.IsEnableVimMode() {
			r = op.vim.HandleVim(r, func() rune {
				r, err := op.t.GetRune(deadline)
				if err == nil {
					return r
				} else {
					return 0
				}
			})
			if r == 0 {
				continue
			}
		}

		var result []rune

		isTypingRune := false

		switch r {
		case CharBell:
			if op.search.IsSearchMode() {
				op.search.ExitSearchMode(true)
				op.buf.Refresh(nil)
			}
			if op.completer.IsInCompleteMode() {
				op.completer.ExitCompleteMode(true)
				op.buf.Refresh(nil)
			}
		case CharBckSearch:
			if !op.search.SearchMode(searchDirectionBackward) {
				op.t.Bell()
				break
			}
			keepInSearchMode = true
		case CharCtrlU:
			op.undo.add()
			op.buf.KillFront()
		case CharFwdSearch:
			if !op.search.SearchMode(searchDirectionForward) {
				op.t.Bell()
				break
			}
			keepInSearchMode = true
		case CharKill:
			op.undo.add()
			op.buf.Kill()
			keepInCompleteMode = true
		case MetaForward:
			op.buf.MoveToNextWord()
		case CharTranspose:
			op.undo.add()
			op.buf.Transpose()
		case MetaBackward:
			op.buf.MoveToPrevWord()
		case MetaDelete:
			op.undo.add()
			op.buf.DeleteWord()
		case CharLineStart:
			op.buf.MoveToLineStart()
		case CharLineEnd:
			op.buf.MoveToLineEnd()
		case CharBackspace, CharCtrlH:
			op.undo.add()
			if op.search.IsSearchMode() {
				op.search.SearchBackspace()
				keepInSearchMode = true
				break
			}

			if op.buf.Len() == 0 {
				op.t.Bell()
				break
			}
			op.buf.Backspace()
		case CharCtrlZ:
			if !platform.IsWindows {
				op.buf.Clean()
				op.t.SleepToResume()
				op.Refresh()
			}
		case CharCtrlL:
			_ = clearScreen(op.t)
			op.buf.SetOffset(cursorPosition{1, 1})
			op.Refresh()
		case MetaBackspace, CharCtrlW:
			op.undo.add()
			op.buf.BackEscapeWord()
		case MetaShiftTab:
			// no-op
		case CharCtrlY:
			op.buf.Yank()
		case CharCtrl_:
			op.undo.undo()
		case CharEnter, CharCtrlJ:
			if op.search.IsSearchMode() {
				op.search.ExitSearchMode(false)
			}
			if op.completer.IsInCompleteMode() {
				op.completer.ExitCompleteMode(true)
				op.buf.Refresh(nil)
			}
			op.buf.MoveToLineEnd()
			var data []rune
			op.buf.WriteRune('\n')
			data = op.buf.Reset()
			data = data[:len(data)-1] // trim \n
			result = data
			if !op.GetConfig().DisableAutoSaveHistory {
				// ignore IO error
				_ = op.history.New(data)
			} else {
				isUpdateHistory = false
			}
			op.undo.init()
		case CharBackward:
			op.buf.MoveBackward()
		case CharForward:
			op.buf.MoveForward()
		case CharPrev:
			buf := op.history.Prev()
			if buf != nil {
				op.buf.Set(buf)
				op.undo.init()
			} else {
				op.t.Bell()
			}
		case CharNext:
			buf, ok := op.history.Next()
			if ok {
				op.buf.Set(buf)
				op.undo.init()
			} else {
				op.t.Bell()
			}
		case MetaDeleteKey, CharEOT:
			op.undo.add()
			// on Delete key or Ctrl-D, attempt to delete a character:
			if op.buf.Len() > 0 || !op.IsNormalMode() {
				if !op.buf.Delete() {
					op.t.Bell()
				}
				break
			}
			if r != CharEOT {
				break
			}
			// Ctrl-D on an empty buffer: treated as EOF
			op.buf.WriteString(op.GetConfig().EOFPrompt + "\n")
			op.buf.Reset()
			isUpdateHistory = false
			op.history.Revert()
			op.buf.Clean()
			return nil, io.EOF
		case CharInterrupt:
			if op.search.IsSearchMode() {
				op.search.ExitSearchMode(true)
				break
			}
			if op.completer.IsInCompleteMode() {
				op.completer.ExitCompleteMode(true)
				op.buf.Refresh(nil)
				break
			}
			op.buf.MoveToLineEnd()
			op.buf.Refresh(nil)
			hint := op.GetConfig().InterruptPrompt + "\n"
			op.buf.WriteString(hint)
			remain := op.buf.Reset()
			remain = remain[:len(remain)-len([]rune(hint))]
			isUpdateHistory = false
			op.history.Revert()
			return nil, ErrInterrupt
		case CharTab:
			if op.GetConfig().AutoComplete != nil {
				if op.completer.OnComplete() {
					if op.completer.IsInCompleteMode() {
						keepInCompleteMode = true
						continue // redraw is done, loop
					}
				} else {
					op.t.Bell()
				}
				op.buf.Refresh(nil)
				break
			} // else: process as a normal input character
			fallthrough
		default:
			isTypingRune = true
			if !isTyping {
				op.undo.add()
			}
			if op.search.IsSearchMode() {
				op.search.SearchChar(r)
				keepInSearchMode = true
				break
			}
			op.buf.WriteRune(r)
			if op.completer.IsInCompleteMode() {
				op.completer.OnComplete()
				if op.completer.IsInCompleteMode() {
					keepInCompleteMode = true
				} else {
					op.buf.Refresh(nil)
				}
			}
		}

		isTyping = isTypingRune

		// suppress the Listener callback if we received Enter or similar and are
		// submitting the result, since the buffer has already been cleared:
		if result == nil {
			if listener := op.GetConfig().Listener; listener != nil {
				newLine, newPos, ok := listener(op.buf.Runes(), op.buf.Pos(), r)
				if ok {
					op.buf.SetWithIdx(newPos, newLine)
				}
			}
		}

		op.m.Lock()
		if !keepInSearchMode && op.search.IsSearchMode() {
			op.search.ExitSearchMode(false)
			op.buf.Refresh(nil)
			op.undo.init()
		} else if op.completer.IsInCompleteMode() {
			if !keepInCompleteMode {
				op.completer.ExitCompleteMode(false)
				op.refresh()
				op.undo.init()
			} else {
				op.buf.Refresh(nil)
				op.completer.CompleteRefresh()
			}
		}
		if isUpdateHistory && !op.search.IsSearchMode() {
			// it will cause null history
			_ = op.history.Update(op.buf.Runes(), false)
		}
		op.m.Unlock()

		if result != nil {
			return result, nil
		}
	}
}

func (op *operation) Stderr() io.Writer {
	return op.wrapErr.Load()
}

func (op *operation) Stdout() io.Writer {
	return op.wrapOut.Load()
}

func (op *operation) String() (string, error) {
	r, err := op.Runes()
	return string(r), err
}

func (op *operation) Runes() ([]rune, error) {
	_ = op.t.EnterRawMode()
	defer func() { _ = op.t.ExitRawMode() }()

	cfg := op.GetConfig()
	listener := cfg.Listener
	if listener != nil {
		listener(nil, 0, 0)
	}

	// Before writing the prompt and starting to read, get a lock
	// so we don't race with wrapWriter trying to write and refresh.
	op.m.Lock()
	op.isPrompting = true
	// Query cursor position before printing the prompt as there
	// may be existing text on the same line that ideally we don't
	// want to overwrite and cause prompt to jump left.
	op.getAndSetOffset(nil)
	op.buf.Print() // print prompt & buffer contents
	// Prompt written safely, unlock until read completes and then
	// lock again to unset.
	op.m.Unlock()

	if cfg.Undo {
		op.undo = newOpUndo(op)
	}

	defer func() {
		op.m.Lock()
		op.isPrompting = false
		op.buf.SetOffset(cursorPosition{1, 1})
		op.m.Unlock()
	}()

	return op.readline(nil)
}

func (op *operation) getAndSetOffset(deadline chan struct{}) {
	if !op.GetConfig().isInteractive {
		return
	}

	// Handle lineedge cases where existing text before before
	// the prompt is printed would leave us at the right edge of
	// the screen but the next character would actually be printed
	// at the beginning of the next line.
	// TODO ???
	_, _ = op.t.Write([]byte(" \b"))

	if offset, err := op.t.GetCursorPosition(deadline); err == nil {
		op.buf.SetOffset(offset)
	}
}

func (op *operation) GenPasswordConfig() *Config {
	baseConfig := op.GetConfig()
	return &Config{
		EnableMask:      true,
		InterruptPrompt: "\n",
		EOFPrompt:       "\n",
		HistoryLimit:    -1,

		Stdin:  baseConfig.Stdin,
		Stdout: baseConfig.Stdout,
		Stderr: baseConfig.Stderr,

		FuncIsTerminal:     baseConfig.FuncIsTerminal,
		FuncMakeRaw:        baseConfig.FuncMakeRaw,
		FuncExitRaw:        baseConfig.FuncExitRaw,
		FuncOnWidthChanged: baseConfig.FuncOnWidthChanged,
	}
}

func (op *operation) ReadLineWithConfig(cfg *Config) (string, error) {
	backupCfg, err := op.SetConfig(cfg)
	if err != nil {
		return "", err
	}
	defer func() {
		_, _ = op.SetConfig(backupCfg)
	}()
	return op.String()
}

func (op *operation) SetTitle(t string) {
	_, _ = op.t.Write([]byte("\033[2;" + t + "\007"))
}

func (op *operation) Slice() ([]byte, error) {
	r, err := op.Runes()
	if err != nil {
		return nil, err
	}
	return []byte(string(r)), nil
}

func (op *operation) Close() {
	op.history.Close()
}

func (op *operation) IsNormalMode() bool {
	return !op.completer.IsInCompleteMode() && !op.search.IsSearchMode()
}

func (op *operation) SetConfig(cfg *Config) (*Config, error) {
	op.m.Lock()
	defer op.m.Unlock()
	old := op.t.GetConfig()
	if err := cfg.init(); err != nil {
		return old, err
	}

	// install the config in its canonical location (inside terminal):
	_ = op.t.SetConfig(cfg)

	op.wrapOut.Store(&wrapWriter{target: cfg.Stdout, o: op})
	op.wrapErr.Store(&wrapWriter{target: cfg.Stderr, o: op})

	if op.history == nil {
		op.history = newOpHistory(op)
	}
	if op.search == nil {
		op.search = newOpSearch(op.buf.w, op.buf, op.history)
	}

	if cfg.AutoComplete != nil && op.completer == nil {
		op.completer = newOpCompleter(op.buf.w, op)
	}

	return old, nil
}

func (op *operation) ResetHistory() {
	op.history.Reset()
}

func (op *operation) SaveToHistory(content string) error {
	return op.history.New([]rune(content))
}

func (op *operation) Refresh() {
	op.m.Lock()
	defer op.m.Unlock()
	op.refresh()
}

func (op *operation) refresh() {
	if op.isPrompting {
		op.buf.Refresh(nil)
	}
}
