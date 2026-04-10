package vte

import (
	"strings"
)

func (vt *VT) osc(data string) {
	selector, val, found := cutString(data, ";")
	if !found {
		return
	}
	switch selector {
	case "0", "1", "2":
		ev := &EventTitle{
			EventTerminal: newEventTerminal(vt),
			title:         val,
		}
		vt.postEvent(ev)
	case "4":
		// OSC 4 — Change/Query Color Number
		// Silently consumed; query responses not implemented.
	case "8":
		if vt.OSC8 {
			url, id := osc8(val)
			vt.cursor.attrs = vt.cursor.attrs.Url(url)
			vt.cursor.attrs = vt.cursor.attrs.UrlId(id)
		}
	case "10":
		// OSC 10 — Set/Query Default Foreground Color
		// Silently consumed; query responses not implemented.
	case "11":
		// OSC 11 — Set/Query Default Background Color
		// Silently consumed; query responses not implemented.
	case "12":
		// OSC 12 — Set/Query Default Cursor Color
		// Silently consumed; query responses not implemented.
	case "52":
		// OSC 52 — Clipboard Access
		// Format: OSC 52 ; <selection> ; <base64-data> ST
		sel, b64data, ok := cutString(val, ";")
		if !ok {
			return
		}
		vt.postEvent(&EventClipboard{
			EventTerminal: newEventTerminal(vt),
			selection:     sel,
			data:          b64data,
		})
	case "104":
		// OSC 104 — Reset Color Number
		// Silently consumed.
	}
}

// parses an osc8 payload into the URL and optional ID
func osc8(val string) (string, string) {
	// OSC 8 ; params ; url ST
	// params: key1=value1:key2=value2
	var id string
	params, url, found := cutString(val, ";")
	if !found {
		return "", ""
	}
	for _, param := range strings.Split(params, ":") {
		key, val, found := cutString(param, "=")
		if !found {
			continue
		}
		switch key {
		case "id":
			id = val
		}
	}
	return url, id
}

// Copied from stdlib to here for go 1.16 compat
func cutString(s string, sep string) (before string, after string, found bool) {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}
