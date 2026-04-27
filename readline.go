//go:build readline

package rumo

import (
	"io"
	"os"

	"github.com/malivvan/readline"
	"github.com/malivvan/readline/term"
)

func init() {
	newReadline = func(prompt string, stdin io.Reader, stdout, stderr io.Writer) func(completer *Completer) (ReadLine, error) {
		return func(completer *Completer) (ReadLine, error) {
			interactive := false
			if fin, ok := stdin.(*os.File); ok {
				if fOut, ok := stdout.(*os.File); ok {
					interactive = term.IsTerminal(int(fin.Fd())) && term.IsTerminal(int(fOut.Fd()))
				}
			}
			rl, err := readline.NewFromConfig(&readline.Config{
				Prompt:          prompt,
				Stdin:           stdin,
				Stdout:          stdout,
				Stderr:          stderr,
				InterruptPrompt: "\n",
				EOFPrompt:       "\n",
				HistoryLimit:    1000,
				Undo:            true,
				FuncIsTerminal:  func() bool { return interactive },
				AutoComplete:    completer,
			})
			if err != nil {
				return nil, err
			}
			return rl, nil
		}
	}

}
