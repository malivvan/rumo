vte/csi.go
Fixed CUP (CSI H / CSI f) to treat 0 params as defaults instead of producing negative coordinates.
Made CUP respect DECOM/scroll-region origin when positioning the cursor.
Fixed DECSTBM (CSI r) default parameter handling, clamping, and cursor homing.
Added DEC private status report handling for CSI ? n, including:
CSI ? 5 n → CSI ? 13 n
CSI ? 6 n → CSI ? row ; col R
Made standard CPR (CSI 6 n) report cursor position relative to origin mode when DECOM is set.
Added secondary device attributes response for CSI > c.
vte/mode.go
Added missing private mode handling for:
?47h / ?47l
?1047h / ?1047l
?1048h / ?1048l
existing ?1049h / ?1049l now routes through shared helpers
Made DECOM set/reset home the cursor, which matches DEC behavior and is important when apps switch into progress/status regions.
vte/vt.go
Added shared helpers for:
homing the cursor with origin mode awareness
reporting cursor position relative to the active origin
Added vte/mode_test.go
Covers:
CUP defaults and origin-mode behavior
DECSTBM defaults and cursor homing
private DSR/CPR responses
DECOM set/reset homing
alternate-screen variants 47, 1047, 1048
Why this likely fixes the apt issue:
Progress/status UIs commonly rely on:
scroll-region setup (CSI ... r)
origin mode (?6h)
cursor position reports (CSI 6 n or CSI ? 6 n)
alternate-screen variants (?47h, ?1047h, ?1048h, ?1049h)
Before the fix:
0 parameters in cursor/margin sequences were handled incorrectly
private CPR requests were ignored
origin mode wasn’t applied to cursor addressing/reporting
several alternate-screen sequences were missing
Any of those can cause an app to appear to “freeze” right when it switches to a bottom progress/status bar.