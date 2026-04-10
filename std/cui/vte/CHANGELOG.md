# VTE Changelog

All notable changes to the `vte` package are documented below.

## Bug Fixes

### Insert Character (ICH)

- **ICH boundary check no longer uses wrong expression.** The shift loop guard in `ich()` added two absolute column indices (`col` + `i`) which produced a meaningless value, causing ICH to prematurely stop shifting characters when the cursor was not at column 0. The redundant check was removed since `i` is already bounded by the screen.

- **ICH blank-fill no longer skips the last column.** The blank-fill loop used `>= (vt.width() - 1)` which prevented writing a blank to the last screen column. The condition is now `>= vt.width()`.

- **ICH no longer copies from before the cursor position.** The source validity check only prevented negative indices but did not stop copying characters from positions before the cursor, violating the ICH spec. The check now correctly bounds against the cursor column.

### Character Set Handling

- **SI (Shift In) now correctly invokes G0 into GL.** The SI handler (0x0F) was incorrectly selecting the G2 charset designator instead of G0, causing broken character set rendering after any Shift-In control code.

- **SS2/SS3 now save the previous charset designator.** ESC N (SS2) and ESC O (SS3) set `charsets.selected` to G2/G3 but never saved the current value to `charsets.saved`. When `print()` reverted the single shift, it restored `saved` which defaulted to G0. If the terminal was in G1 (after SO), the single shift incorrectly reverted to G0 instead of G1.

- **Single shift flag is now cleared after use.** After reverting the charset from a single shift, the `singleShift` flag was never set to `false`. Every subsequent `print()` call continued to overwrite `selected` with `saved`, which could undo later charset switches (e.g., SO setting G1 would be immediately reverted by the stale flag on the next character print).

### Backspace Reverse Wrap

- **Backspace reverse-wrap is now gated on DECAWM (mode 7).** Previously, pressing backspace at the left margin would unconditionally reverse-wrap to the end of the previous line. Per spec, reverse wrap should only occur when Auto Wrap Mode is enabled. A mode check is now enforced.

### Cell Erase

- **`cell.erase()` now fully resets cell state.** The erase method previously left `combining`, `width`, and `wrapped` fields untouched. Stale combining characters and width values could persist after erase operations, causing ghost diacritics or rendering artifacts. All fields are now cleared.

### Cursor Movement

- **CUD/CUF/CUB no longer clamp to margin when the cursor is outside the scroll region.** Per the DEC VT510 spec, if the cursor is already below the scroll region, CUD should stop at the last screen line, not the bottom margin. The same issue affected CUF (clamped to `margin.right` instead of last column) and CUB (clamped to `margin.left` instead of column 0). CUU already handled this correctly. The incorrect clamping could cause the cursor to snap backward into the scroll region.

- **CNL (Cursor Next Line) no longer scrolls at the bottom margin.** Each iteration called `nel()` → `ind()`, which scrolled the screen up at the bottom margin. Per ECMA-48 §8.3.20, CNL should move the cursor down Ps lines to column 1, stopping at the bottom margin without scrolling.

- **CPL (Cursor Preceding Line) no longer scrolls at the top margin.** Each iteration called `ri()`, which scrolled the screen down at the top margin. Per ECMA-48 §8.3.13, CPL should move the cursor up Ps lines to column 1, stopping at the top margin without scrolling.

- **CHT (Cursor Forward Tabulation) no longer counts the current tab stop position.** When the cursor was already at a tab stop, the comparison used `>` instead of `>=`, causing a no-op that consumed a tab advance without actually moving the cursor.

- **CBT (Cursor Backward Tabulation) no longer counts the current tab stop position.** Same off-by-one as CHT but for backward tabulation — the loop now correctly continues past the current position instead of breaking.

- **CPL (Cursor Preceding Line) comment corrected from "down" to "up".**

### Device Status Report (DSR)

- **Private DSR 5 now returns the correct response code.** The handler for `CSI ? 5 n` responded with `\x1b[?13n` ("no printer" — the response for `CSI ? 15 n`). The correct response for a general DEC status query is `\x1b[?0n` ("no malfunction detected").

### Erase Characters (ECH)

- **ECH no longer skips the last screen column.** An off-by-one in the boundary check (`==` instead of `>=`) caused the rightmost column to never be erased when targeted.

### Repeat (REP)

- **REP now correctly advances the cursor, copies attributes and width, and writes to the right margin.** The previous implementation only copied cell content without advancing `cursor.col`, did not propagate `attrs` or `width`, and skipped the right margin boundary. REP now behaves like normal character printing repeated Ps times, per ECMA-48.

### Tab Stop Management (HTS)

- **HTS now inserts tab stops in sorted order with deduplication.** Previously, `hts()` appended the current column to the tab stop list without maintaining sorted order. Since both CHT and CBT iterate the list assuming it is sorted, setting tab stops in non-sequential order would cause tabulation to malfunction.

### Scroll Region Margins

- **`scrollUp` and `scrollDown` now copy only within left/right margins.** Previously, `scrollUp` would copy the full row (including cells outside the left-right margins) during a scroll, even though the erase loop correctly respected margins. Both functions now consistently operate within the scrolling region.

### Resize

- **`Resize()` no longer discards content on the cursor row.** The screen-replay loop broke at the cursor row instead of including it, meaning the cursor row's content was never replayed to the new screen buffer. Any text on that row was lost after a resize.

### Key Mappings

- **Alt+F4 through Alt+F12 key codes corrected.** Alt+F4 was mapped to Alt+F5's escape sequence (KeyF53 instead of KeyF52), causing a cascading off-by-one error for all Alt+function key combinations from F4 through F12.

- **Meta+Shift+Left and Meta+Shift+Right escape sequences swapped.** `KeyMetaShfLeft` was set to the cursor-forward sequence (`\x1b[1;10C`) and `KeyMetaShfRight` to cursor-backward (`\x1b[1;10D`). These are now correctly assigned.

### Sixel Capability Advertisement

- **Removed sixel capability flag from DA1 response.** The Device Attributes response advertised sixel graphics support (`4;`), but no sixel handling is implemented. Applications that rely on the advertised capability would send sixel data that was silently discarded.

## Stability Fixes

- **Panic recovery no longer deadlocks on the event channel.** When a panic occurred in the parser goroutine, the recovery function called `postEvent()` which sent to the `events` channel (buffered to 2). Since the same goroutine was the only consumer, and up to 2 events may already have been buffered, `postEvent` would block forever. The `postEvent` function now uses a non-blocking send that drops the event if the channel is full.

- **`VT.Resize()` no longer panics when pty is nil.** `Resize` can be called before `Start` (e.g., from `Terminal.Draw` on the first frame), at which point `vt.pty` is nil. A nil guard now prevents the nil-pointer dereference.

- **`VT.Close()` no longer panics when pty is nil.** Similarly, calling `Close()` on a VT that was never started would dereference a nil pointer. A nil guard was added.

- **Event loop now drains all pending events before parsing the next sequence.** The main parser goroutine previously alternated between draining a single event and parsing via `select`/`default`. Under rapid output, this caused oscillation between producing and consuming events without making forward progress on the input stream. The loop now drains all buffered events in a batch before proceeding to parse.

## Concurrency Fixes

- **`VT.Resize()` now acquires `vt.mu` before modifying shared state.** `Resize` modifies `activeScreen`, `margin`, `cursor`, and other fields also accessed by the parser goroutine under `vt.mu`. Without the lock, concurrent calls from `Terminal.Draw` during parsing caused data races.

## New Features

### C0 Control Codes

- **Added ENQ (0x05) answerback handler.** Per VT100 spec, ENQ triggers an answerback response. Applications that send ENQ and wait for a response would previously hang. The handler now responds with an empty string.

### CSI Sequences

- **Added ED 3 (erase scrollback buffer).** The `ed()` function previously only handled Ps=0, 1, 2. ED 3 is an xterm extension used by the `clear` command and `printf '\e[3J'` to erase the scrollback buffer. Since this VT has no scrollback, it is treated equivalently to ED 2.

- **Implemented DECSCA (`CSI Ps " q`) — Select Character Protection Attribute.** Characters printed while DECSCA is active are marked as protected. DECSED and DECSEL now correctly skip protected cells during selective erase.

- **Implemented XTWINOPS (`CSI Ps t`) — Window Manipulation.** Handles Ps=14 (report window size in pixels, approximated using an 8×16 cell size), Ps=18 (report terminal size in characters), and Ps=22/23 (save/restore title, silently consumed).

### Key Handling

- **Added Alt+Shift+F5 through Alt+Shift+F12 mappings.** The `Alt|Shift` modifier case previously only handled F1–F4. Function keys F5–F12 with Alt+Shift are now correctly mapped.

### OSC (Operating System Command) Sequences

- **Added OSC 1 (Set Icon Name) handler.** Some applications use OSC 1 separately from OSC 2 to set only the icon name.

- **Added OSC 4 (Change/Query Color Number) — silently consumed.**

- **Added OSC 10/11/12 (Default Foreground/Background/Cursor Color) — silently consumed.**

- **Added OSC 52 (Clipboard Access) — emits `EventClipboard` event.** Applications can now request clipboard operations via OSC 52, which are surfaced as events to the host.

- **Added OSC 104 (Reset Color Number) — silently consumed.**

### ESC Sequences

- **Implemented DECBI (ESC 6) — Back Index.** If the cursor is at the left margin, scrolls content within the margins to the right by one column. Otherwise, moves the cursor left one column.

- **Implemented DECFI (ESC 9) — Forward Index.** If the cursor is at the right margin, scrolls content within the margins to the left by one column. Otherwise, moves the cursor right one column.

### Selective Erase

- **Implemented DECSED (`CSI ? J`) and DECSEL (`CSI ? K`).** Selective Erase in Display and Selective Erase in Line now erase only characters that do not have the protected attribute (DECSCA) set, using the existing `selectiveErase()` cell method.

### SGR (Select Graphic Rendition)

- **Added SGR 6 (Rapid Blink) support.** Rapid blink is now treated identically to slow blink (SGR 5), mapping to `Blink(true)`.

- **Added SGR 26 (Rapid Blink Off) support.** Turns off blink, corresponding to SGR 6.

- **Added SGR 53/55 (Overline) support.** SGR 53 enables the overline attribute and SGR 55 disables it. The attribute is tracked per-cell and per-cursor.

### DEC Private Modes

- **Added focus event reporting mode (DECSET/DECRST 1004).** When enabled, the terminal sends `ESC [ I` on focus-in and `ESC [ O` on focus-out to the application. Many modern TUI applications (neovim, tmux, etc.) use this to react to focus changes.

- **Added synchronized output mode (DECSET/DECRST 2026).** Modern terminals use this to batch screen updates and prevent tearing. The mode flag is now tracked so applications that query it see the expected state.

- **DECSCNM (Reverse Video) now has a rendering effect.** DECSET/DECRST 5 was previously recognized but had no visual effect. `Draw()` now applies `Reverse(true)` to all cell styles when DECSCNM is active, inverting the screen as applications expect.

- **Added X10 mouse compatibility mode (DECSET/DECRST 9).** Reports button presses only (not releases or motion) using the legacy X10 encoding. Some older applications still use this protocol.

- **Added UTF-8 mouse encoding mode (DECSET/DECRST 1005).** Encodes mouse coordinates as UTF-8 characters, allowing positions beyond column/row 223. Used by some applications as an alternative to SGR mouse encoding.

### 8-bit C1 Control Codes

- **Added 8-bit C1 control code handling in the parser.** The `anywhere()` function now handles 0x9B (CSI), 0x9D (OSC), 0x90 (DCS), and 0x9C (ST) transitions. Previously, 8-bit control sequences were misinterpreted as printable characters.

### DECALN (Screen Alignment Pattern)

- **Implemented ESC `#8` (DECALN).** Fills the entire screen with 'E' characters and resets margins and cursor position, as used by vttest and other terminal test suites.

### DEC Line Attributes

- **Added silent stubs for ESC `#3`, `#4`, `#5`, `#6`.** DECDHL (top/bottom half), DECSWL (single-width), and DECDWL (double-width) sequences are now silently consumed instead of falling through unhandled.


## Code Quality

- **Removed dead `ps(params)` call in DECSCUSR handler.** The CSI `" q"` handler called `ps(params)` twice, discarding the first result. Reduced to a single call.

- **Removed redundant `setExit` call in DCS passthrough state.** `dcsPassthrough` was calling `p.setExit(p.unhook)` on every character received, acquiring a mutex each time. The exit function is already set during `hook()`.

- **Removed unreachable EOF case from `csiIntermediate`.** EOF is handled by the `anywhere()` wrapper before the state function runs, making the explicit case dead code.

## Build Fixes

- **Added missing `syscall` import to `proc_windows.go`.** The file references `syscall.SysProcAttr{}` but did not import the `syscall` package, causing a compilation failure on Windows.

## Documentation Fixes

- **Fixed incorrect hex codes in C0 control code comments.** Comments for LF, VT, FF, and CR listed hex values offset by +6 (e.g., LF showed `0x10` instead of `0x0A`).

- **Fixed copy-paste errors in terminfo field comments.** Several fields (`KeyCtrlDown`, `KeyMetaDown`, `KeyAltDown`, and others) had comments saying "left" instead of "down".

- **Fixed typo "unexpected characted" → "unexpected character"** in five parser error messages.

- **Fixed typo "licensed undo" → "licensed under"** in parser doc comment.

