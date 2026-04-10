# VTE Changelog

All notable changes to the `vte` package are documented below.

## Bug Fixes

### Character Set Handling

- **SI (Shift In) now correctly invokes G0 into GL.** The SI handler (0x0F) was incorrectly selecting the G2 charset designator instead of G0, causing broken character set rendering after any Shift-In control code.

### Backspace Reverse Wrap

- **Backspace reverse-wrap is now gated on DECAWM (mode 7).** Previously, pressing backspace at the left margin would unconditionally reverse-wrap to the end of the previous line. Per spec, reverse wrap should only occur when Auto Wrap Mode is enabled. A mode check is now enforced.

### Cell Erase

- **`cell.erase()` now fully resets cell state.** The erase method previously left `combining`, `width`, and `wrapped` fields untouched. Stale combining characters and width values could persist after erase operations, causing ghost diacritics or rendering artifacts. All fields are now cleared.

### Cursor Movement

- **CHT (Cursor Forward Tabulation) no longer counts the current tab stop position.** When the cursor was already at a tab stop, the comparison used `>` instead of `>=`, causing a no-op that consumed a tab advance without actually moving the cursor.

- **CBT (Cursor Backward Tabulation) no longer counts the current tab stop position.** Same off-by-one as CHT but for backward tabulation — the loop now correctly continues past the current position instead of breaking.

- **CPL (Cursor Preceding Line) comment corrected from "down" to "up".**

### Erase Characters (ECH)

- **ECH no longer skips the last screen column.** An off-by-one in the boundary check (`==` instead of `>=`) caused the rightmost column to never be erased when targeted.

### Repeat (REP)

- **REP now correctly advances the cursor, copies attributes and width, and writes to the right margin.** The previous implementation only copied cell content without advancing `cursor.col`, did not propagate `attrs` or `width`, and skipped the right margin boundary. REP now behaves like normal character printing repeated Ps times, per ECMA-48.

### Tab Stop Management (HTS)

- **HTS now inserts tab stops in sorted order with deduplication.** Previously, `hts()` appended the current column to the tab stop list without maintaining sorted order. Since both CHT and CBT iterate the list assuming it is sorted, setting tab stops in non-sequential order would cause tabulation to malfunction.

### Scroll Region Margins

- **`scrollUp` and `scrollDown` now copy only within left/right margins.** Previously, `scrollUp` would copy the full row (including cells outside the left-right margins) during a scroll, even though the erase loop correctly respected margins. Both functions now consistently operate within the scrolling region.

### Key Mappings

- **Alt+F4 through Alt+F12 key codes corrected.** Alt+F4 was mapped to Alt+F5's escape sequence (KeyF53 instead of KeyF52), causing a cascading off-by-one error for all Alt+function key combinations from F4 through F12.

- **Meta+Shift+Left and Meta+Shift+Right escape sequences swapped.** `KeyMetaShfLeft` was set to the cursor-forward sequence (`\x1b[1;10C`) and `KeyMetaShfRight` to cursor-backward (`\x1b[1;10D`). These are now correctly assigned.

### Sixel Capability Advertisement

- **Removed sixel capability flag from DA1 response.** The Device Attributes response advertised sixel graphics support (`4;`), but no sixel handling is implemented. Applications that rely on the advertised capability would send sixel data that was silently discarded.

## Stability Fixes

- **`VT.Resize()` no longer panics when pty is nil.** `Resize` can be called before `Start` (e.g., from `Terminal.Draw` on the first frame), at which point `vt.pty` is nil. A nil guard now prevents the nil-pointer dereference.

- **`VT.Close()` no longer panics when pty is nil.** Similarly, calling `Close()` on a VT that was never started would dereference a nil pointer. A nil guard was added.

## Concurrency Fixes

- **`VT.Resize()` now acquires `vt.mu` before modifying shared state.** `Resize` modifies `activeScreen`, `margin`, `cursor`, and other fields also accessed by the parser goroutine under `vt.mu`. Without the lock, concurrent calls from `Terminal.Draw` during parsing caused data races.

## New Features

### Key Handling

- **Added Alt+Shift+F5 through Alt+Shift+F12 mappings.** The `Alt|Shift` modifier case previously only handled F1–F4. Function keys F5–F12 with Alt+Shift are now correctly mapped.

### OSC (Operating System Command) Sequences

- **Added OSC 1 (Set Icon Name) handler.** Some applications use OSC 1 separately from OSC 2 to set only the icon name.

- **Added OSC 4 (Change/Query Color Number) — silently consumed.**

- **Added OSC 10/11/12 (Default Foreground/Background/Cursor Color) — silently consumed.**

- **Added OSC 52 (Clipboard Access) — emits `EventClipboard` event.** Applications can now request clipboard operations via OSC 52, which are surfaced as events to the host.

- **Added OSC 104 (Reset Color Number) — silently consumed.**

### Selective Erase

- **Implemented DECSED (`CSI ? J`) and DECSEL (`CSI ? K`).** Selective Erase in Display and Selective Erase in Line now erase only characters that do not have the protected attribute (DECSCA) set, using the existing `selectiveErase()` cell method.

### SGR (Select Graphic Rendition)

- **Added SGR 6 (Rapid Blink) support.** Rapid blink is now treated identically to slow blink (SGR 5), mapping to `Blink(true)`.

- **Added SGR 26 (Rapid Blink Off) support.** Turns off blink, corresponding to SGR 6.

### 8-bit C1 Control Codes

- **Added 8-bit C1 control code handling in the parser.** The `anywhere()` function now handles 0x9B (CSI), 0x9D (OSC), 0x90 (DCS), and 0x9C (ST) transitions. Previously, 8-bit control sequences were misinterpreted as printable characters.

### DECALN (Screen Alignment Pattern)

- **Implemented ESC `#8` (DECALN).** Fills the entire screen with 'E' characters and resets margins and cursor position, as used by vttest and other terminal test suites.

### DEC Line Attributes

- **Added silent stubs for ESC `#3`, `#4`, `#5`, `#6`.** DECDHL (top/bottom half), DECSWL (single-width), and DECDWL (double-width) sequences are now silently consumed instead of falling through unhandled.

### DECSCNM (Reverse Video)

- **Mode 5 (DECSCNM) is now explicitly recognized** in both DECSET and DECRST handlers with a documented note that it is intentionally not rendered.

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

