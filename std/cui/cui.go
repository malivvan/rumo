package cui

import (
	"context"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/module"
)

// -- Module registration (constructors) --------------------------------

// Module provides a terminal-based UI toolkit built on top of tview.
var Module = module.NewBuiltin().
	Func("new_app() (app *App)", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		return wrapApp(NewApp()), nil
	}).
	Func("new_box() (box *Box)", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		return wrapBox(), nil
	}).
	Func("new_text_view() (tv *TextView)", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		return wrapTextView(NewTextView()), nil
	}).
	Func("new_button(label string) (btn *Button)", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		label, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
		}
		return wrapButton(NewButton(label)), nil
	}).
	Func("new_check_box() (cb *CheckBox)", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		return wrapCheckBox(NewCheckBox()), nil
	}).
	Func("new_input_field() (i *InputField)", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		return wrapInputField(NewInputField()), nil
	}).
	Func("new_list() (l *List)", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		return wrapList(NewList()), nil
	}).
	Func("new_table() (t *Table)", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		return wrapTable(NewTable()), nil
	}).
	Func("new_flex() (f *Flex)", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		return wrapFlex(NewFlex()), nil
	}).
	Func("new_grid() (g *Grid)", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		return wrapGrid(NewGrid()), nil
	}).
	Func("new_form() (f *Form)", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		return wrapForm(NewForm()), nil
	}).
	Func("new_modal() (m *Modal)", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		return wrapModal(NewModal()), nil
	}).
	Func("new_panels() (p *Panels)", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		return wrapPanels(NewPanels()), nil
	}).
	Func("new_tabbed_panels() (tp *TabbedPanels)", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		return wrapTabbedPanels(NewTabbedPanels()), nil
	}).
	Func("new_tree_view() (tv *TreeView)", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		return wrapTreeView(NewTreeView()), nil
	}).
	Func("new_tree_node(text string) (node *TreeNode)", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		text, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
		}
		return wrapTreeNode(NewTreeNode(text)), nil
	}).
	Func("new_progress_bar() (pb *ProgressBar)", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		return wrapProgressBar(NewProgressBar()), nil
	}).
	Func("new_drop_down() (dd *DropDown)", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		return wrapDropDown(NewDropDown()), nil
	}).
	Func("new_frame(widget *Widget) (fr *Frame)", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		w := toWidget(args[0])
		if w == nil {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "widget", Found: args[0].TypeName()}
		}
		return wrapFrame(NewFrame(w)), nil
	}).
	Const("flex_row int", FlexRow).
	Const("flex_column int", FlexColumn).
	Const("align_left int", AlignLeft).
	Const("align_center int", AlignCenter).
	Const("align_right int", AlignRight)

// -- App wrapper -------------------------------------------------------

func wrapApp(a *App) *vm.ImmutableMap {
	return &vm.ImmutableMap{Value: map[string]vm.Object{
		"run": fn("run", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			if err := a.Run(); err != nil {
				return module.WrapError(err), nil
			}
			return vm.UndefinedValue, nil
		}),
		"stop": fn("stop", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			a.Stop()
			return vm.UndefinedValue, nil
		}),
		"draw": fn("draw", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			a.Draw()
			return vm.UndefinedValue, nil
		}),
		"enable_mouse": fn("enable_mouse", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			if len(args) != 1 {
				return nil, vm.ErrWrongNumArguments
			}
			a.EnableMouse(!args[0].IsFalsy())
			return vm.UndefinedValue, nil
		}),
		"set_root": fn("set_root", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			if len(args) < 1 || len(args) > 2 {
				return nil, vm.ErrWrongNumArguments
			}
			w := toWidget(args[0])
			if w == nil {
				return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "widget", Found: args[0].TypeName()}
			}
			fullscreen := true
			if len(args) == 2 {
				fullscreen = !args[1].IsFalsy()
			}
			a.SetRoot(w, fullscreen)
			return vm.UndefinedValue, nil
		}),
		"set_focus": fn("set_focus", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			if len(args) != 1 {
				return nil, vm.ErrWrongNumArguments
			}
			w := toWidget(args[0])
			if w == nil {
				return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "widget", Found: args[0].TypeName()}
			}
			a.SetFocus(w)
			return vm.UndefinedValue, nil
		}),
		"queue_update": fn("queue_update", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			if len(args) != 1 {
				return nil, vm.ErrWrongNumArguments
			}
			if !args[0].CanCall() {
				return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "callable", Found: args[0].TypeName()}
			}
			a.QueueUpdate(func() {
				args[0].Call(ctx)
			})
			return vm.UndefinedValue, nil
		}),
		"queue_update_draw": fn("queue_update_draw", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			if len(args) != 1 {
				return nil, vm.ErrWrongNumArguments
			}
			if !args[0].CanCall() {
				return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "callable", Found: args[0].TypeName()}
			}
			a.QueueUpdateDraw(func() {
				args[0].Call(ctx)
			})
			return vm.UndefinedValue, nil
		}),
	}}
}

// -- Box (base) wrapper ------------------------------------------------

func addBoxMethods(m map[string]vm.Object, b *Box) {
	m["set_border"] = fn("set_border", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		b.SetBorder(!args[0].IsFalsy())
		return vm.UndefinedValue, nil
	})
	m["set_title"] = fn("set_title", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		s, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
		}
		b.SetTitle(s)
		return vm.UndefinedValue, nil
	})
	m["get_title"] = fn("get_title", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		return &vm.String{Value: b.GetTitle()}, nil
	})
	m["set_rect"] = fn("set_rect", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 4 {
			return nil, vm.ErrWrongNumArguments
		}
		x, ok := vm.ToInt(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "x", Expected: "int", Found: args[0].TypeName()}
		}
		y, ok := vm.ToInt(args[1])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "y", Expected: "int", Found: args[1].TypeName()}
		}
		w, ok := vm.ToInt(args[2])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "width", Expected: "int", Found: args[2].TypeName()}
		}
		h, ok := vm.ToInt(args[3])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "height", Expected: "int", Found: args[3].TypeName()}
		}
		b.SetRect(x, y, w, h)
		return vm.UndefinedValue, nil
	})
	m["get_rect"] = fn("get_rect", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		x, y, w, h := b.GetRect()
		return &vm.ImmutableMap{Value: map[string]vm.Object{
			"x": &vm.Int{Value: int64(x)}, "y": &vm.Int{Value: int64(y)},
			"width": &vm.Int{Value: int64(w)}, "height": &vm.Int{Value: int64(h)},
		}}, nil
	})
	m["set_visible"] = fn("set_visible", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		b.SetVisible(!args[0].IsFalsy())
		return vm.UndefinedValue, nil
	})
	m["get_visible"] = fn("get_visible", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		return boolVal(b.GetVisible()), nil
	})
	m["set_padding"] = fn("set_padding", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 4 {
			return nil, vm.ErrWrongNumArguments
		}
		t, _ := vm.ToInt(args[0])
		bot, _ := vm.ToInt(args[1])
		l, _ := vm.ToInt(args[2])
		r, _ := vm.ToInt(args[3])
		b.SetPadding(t, bot, l, r)
		return vm.UndefinedValue, nil
	})
}

// -- Box widget wrapper -----------------------------------------------

func wrapBox() *vm.ImmutableMap {
	b := NewBox()
	m := make(map[string]vm.Object)
	m["__widget"] = &widgetPtr{w: b}
	addBoxMethods(m, b)
	return &vm.ImmutableMap{Value: m}
}

// -- TextView wrapper -------------------------------------------------

func wrapTextView(tv *TextView) *vm.ImmutableMap {
	m := make(map[string]vm.Object)
	m["__widget"] = &widgetPtr{w: tv}
	addBoxMethods(m, tv.Box)

	m["set_text"] = fn("set_text", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		s, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
		}
		tv.SetText(s)
		return vm.UndefinedValue, nil
	})
	m["get_text"] = fn("get_text", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		strip := false
		if len(args) > 0 {
			strip = !args[0].IsFalsy()
		}
		return &vm.String{Value: tv.GetText(strip)}, nil
	})
	m["set_text_align"] = fn("set_text_align", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		a, ok := vm.ToInt(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "int", Found: args[0].TypeName()}
		}
		tv.SetTextAlign(a)
		return vm.UndefinedValue, nil
	})
	m["set_dynamic_colors"] = fn("set_dynamic_colors", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		tv.SetDynamicColors(!args[0].IsFalsy())
		return vm.UndefinedValue, nil
	})
	m["set_regions"] = fn("set_regions", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		tv.SetRegions(!args[0].IsFalsy())
		return vm.UndefinedValue, nil
	})
	m["set_scrollable"] = fn("set_scrollable", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		tv.SetScrollable(!args[0].IsFalsy())
		return vm.UndefinedValue, nil
	})
	m["set_wrap"] = fn("set_wrap", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		tv.SetWrap(!args[0].IsFalsy())
		return vm.UndefinedValue, nil
	})
	m["set_word_wrap"] = fn("set_word_wrap", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		tv.SetWordWrap(!args[0].IsFalsy())
		return vm.UndefinedValue, nil
	})
	m["clear"] = fn("clear", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		tv.Clear()
		return vm.UndefinedValue, nil
	})
	m["set_max_lines"] = fn("set_max_lines", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		n, ok := vm.ToInt(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "int", Found: args[0].TypeName()}
		}
		tv.SetMaxLines(n)
		return vm.UndefinedValue, nil
	})
	m["scroll_to_end"] = fn("scroll_to_end", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		tv.ScrollToEnd()
		return vm.UndefinedValue, nil
	})
	m["scroll_to_beginning"] = fn("scroll_to_beginning", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		tv.ScrollToBeginning()
		return vm.UndefinedValue, nil
	})
	return &vm.ImmutableMap{Value: m}
}

// -- Button wrapper ---------------------------------------------------

func wrapButton(b *Button) *vm.ImmutableMap {
	m := make(map[string]vm.Object)
	m["__widget"] = &widgetPtr{w: b}
	addBoxMethods(m, b.Box)

	m["set_label"] = fn("set_label", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		s, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
		}
		b.SetLabel(s)
		return vm.UndefinedValue, nil
	})
	m["get_label"] = fn("get_label", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		return &vm.String{Value: b.GetLabel()}, nil
	})
	m["set_selected_func"] = fn("set_selected_func", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		if !args[0].CanCall() {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "callable", Found: args[0].TypeName()}
		}
		cb := args[0]
		b.SetSelectedFunc(func() {
			cb.Call(ctx)
		})
		return vm.UndefinedValue, nil
	})
	return &vm.ImmutableMap{Value: m}
}

// -- CheckBox wrapper -------------------------------------------------

func wrapCheckBox(c *CheckBox) *vm.ImmutableMap {
	m := make(map[string]vm.Object)
	m["__widget"] = &widgetPtr{w: c}
	addBoxMethods(m, c.Box)

	m["set_label"] = fn("set_label", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		s, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
		}
		c.SetLabel(s)
		return vm.UndefinedValue, nil
	})
	m["get_label"] = fn("get_label", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		return &vm.String{Value: c.GetLabel()}, nil
	})
	m["set_checked"] = fn("set_checked", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		c.SetChecked(!args[0].IsFalsy())
		return vm.UndefinedValue, nil
	})
	m["is_checked"] = fn("is_checked", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		return boolVal(c.IsChecked()), nil
	})
	m["set_message"] = fn("set_message", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		s, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
		}
		c.SetMessage(s)
		return vm.UndefinedValue, nil
	})
	m["set_changed_func"] = fn("set_changed_func", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		if !args[0].CanCall() {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "callable", Found: args[0].TypeName()}
		}
		cb := args[0]
		c.SetChangedFunc(func(checked bool) {
			cb.Call(ctx, boolVal(checked))
		})
		return vm.UndefinedValue, nil
	})
	return &vm.ImmutableMap{Value: m}
}

// -- InputField wrapper -----------------------------------------------

func wrapInputField(i *InputField) *vm.ImmutableMap {
	m := make(map[string]vm.Object)
	m["__widget"] = &widgetPtr{w: i}
	addBoxMethods(m, i.Box)

	m["set_label"] = fn("set_label", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		s, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
		}
		i.SetLabel(s)
		return vm.UndefinedValue, nil
	})
	m["get_label"] = fn("get_label", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		return &vm.String{Value: i.GetLabel()}, nil
	})
	m["set_text"] = fn("set_text", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		s, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
		}
		i.SetText(s)
		return vm.UndefinedValue, nil
	})
	m["get_text"] = fn("get_text", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		return &vm.String{Value: i.GetText()}, nil
	})
	m["set_placeholder"] = fn("set_placeholder", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		s, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
		}
		i.SetPlaceholder(s)
		return vm.UndefinedValue, nil
	})
	m["set_field_width"] = fn("set_field_width", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		w, ok := vm.ToInt(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "int", Found: args[0].TypeName()}
		}
		i.SetFieldWidth(w)
		return vm.UndefinedValue, nil
	})
	m["set_changed_func"] = fn("set_changed_func", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		if !args[0].CanCall() {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "callable", Found: args[0].TypeName()}
		}
		cb := args[0]
		i.SetChangedFunc(func(text string) {
			cb.Call(ctx, &vm.String{Value: text})
		})
		return vm.UndefinedValue, nil
	})
	return &vm.ImmutableMap{Value: m}
}

// -- List wrapper -----------------------------------------------------

func wrapList(l *List) *vm.ImmutableMap {
	m := make(map[string]vm.Object)
	m["__widget"] = &widgetPtr{w: l}
	addBoxMethods(m, l.Box)

	m["add_item"] = fn("add_item", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) < 1 {
			return nil, vm.ErrWrongNumArguments
		}
		s, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
		}
		item := NewListItem(s)
		if len(args) >= 2 {
			sec, ok := strictStr(args[1])
			if ok {
				item.SetSecondaryText(sec)
			}
		}
		if len(args) >= 3 && args[2].CanCall() {
			cb := args[2]
			item.SetSelectedFunc(func() {
				cb.Call(ctx)
			})
		}
		l.AddItem(item)
		return vm.UndefinedValue, nil
	})
	m["clear"] = fn("clear", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		l.Clear()
		return vm.UndefinedValue, nil
	})
	m["get_item_count"] = fn("get_item_count", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		return &vm.Int{Value: int64(l.GetItemCount())}, nil
	})
	m["get_current_item_index"] = fn("get_current_item_index", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		return &vm.Int{Value: int64(l.GetCurrentItemIndex())}, nil
	})
	m["set_current_item"] = fn("set_current_item", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		idx, ok := vm.ToInt(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "int", Found: args[0].TypeName()}
		}
		l.SetCurrentItem(idx)
		return vm.UndefinedValue, nil
	})
	m["remove_item"] = fn("remove_item", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		idx, ok := vm.ToInt(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "int", Found: args[0].TypeName()}
		}
		l.RemoveItem(idx)
		return vm.UndefinedValue, nil
	})
	m["set_selected_func"] = fn("set_selected_func", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		if !args[0].CanCall() {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "callable", Found: args[0].TypeName()}
		}
		cb := args[0]
		l.SetSelectedFunc(func(index int, item *ListItem) {
			cb.Call(ctx, &vm.Int{Value: int64(index)}, &vm.String{Value: item.GetMainText()})
		})
		return vm.UndefinedValue, nil
	})
	m["set_changed_func"] = fn("set_changed_func", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		if !args[0].CanCall() {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "callable", Found: args[0].TypeName()}
		}
		cb := args[0]
		l.SetChangedFunc(func(index int, item *ListItem) {
			cb.Call(ctx, &vm.Int{Value: int64(index)}, &vm.String{Value: item.GetMainText()})
		})
		return vm.UndefinedValue, nil
	})
	m["show_secondary_text"] = fn("show_secondary_text", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		l.ShowSecondaryText(!args[0].IsFalsy())
		return vm.UndefinedValue, nil
	})
	m["set_highlight_full_line"] = fn("set_highlight_full_line", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		l.SetHighlightFullLine(!args[0].IsFalsy())
		return vm.UndefinedValue, nil
	})
	return &vm.ImmutableMap{Value: m}
}

// -- Table wrapper ----------------------------------------------------

func wrapTable(t *Table) *vm.ImmutableMap {
	m := make(map[string]vm.Object)
	m["__widget"] = &widgetPtr{w: t}
	addBoxMethods(m, t.Box)

	m["set_cell"] = fn("set_cell", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 3 {
			return nil, vm.ErrWrongNumArguments
		}
		row, ok := vm.ToInt(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "row", Expected: "int", Found: args[0].TypeName()}
		}
		col, ok := vm.ToInt(args[1])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "column", Expected: "int", Found: args[1].TypeName()}
		}
		text, ok := strictStr(args[2])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "text", Expected: "string", Found: args[2].TypeName()}
		}
		t.SetCellSimple(row, col, text)
		return vm.UndefinedValue, nil
	})
	m["get_cell_text"] = fn("get_cell_text", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 2 {
			return nil, vm.ErrWrongNumArguments
		}
		row, ok := vm.ToInt(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "row", Expected: "int", Found: args[0].TypeName()}
		}
		col, ok := vm.ToInt(args[1])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "column", Expected: "int", Found: args[1].TypeName()}
		}
		cell := t.GetCell(row, col)
		if cell == nil {
			return vm.UndefinedValue, nil
		}
		return &vm.String{Value: cell.GetText()}, nil
	})
	m["clear"] = fn("clear", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		t.Clear()
		return vm.UndefinedValue, nil
	})
	m["get_row_count"] = fn("get_row_count", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		return &vm.Int{Value: int64(t.GetRowCount())}, nil
	})
	m["get_column_count"] = fn("get_column_count", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		return &vm.Int{Value: int64(t.GetColumnCount())}, nil
	})
	m["set_borders"] = fn("set_borders", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		t.SetBorders(!args[0].IsFalsy())
		return vm.UndefinedValue, nil
	})
	m["set_fixed"] = fn("set_fixed", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 2 {
			return nil, vm.ErrWrongNumArguments
		}
		rows, ok := vm.ToInt(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "rows", Expected: "int", Found: args[0].TypeName()}
		}
		cols, ok := vm.ToInt(args[1])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "columns", Expected: "int", Found: args[1].TypeName()}
		}
		t.SetFixed(rows, cols)
		return vm.UndefinedValue, nil
	})
	m["set_selectable"] = fn("set_selectable", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 2 {
			return nil, vm.ErrWrongNumArguments
		}
		t.SetSelectable(!args[0].IsFalsy(), !args[1].IsFalsy())
		return vm.UndefinedValue, nil
	})
	m["get_selection"] = fn("get_selection", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		row, col := t.GetSelection()
		return &vm.ImmutableMap{Value: map[string]vm.Object{
			"row":    &vm.Int{Value: int64(row)},
			"column": &vm.Int{Value: int64(col)},
		}}, nil
	})
	m["select_row"] = fn("select_row", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		row, ok := vm.ToInt(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "int", Found: args[0].TypeName()}
		}
		t.Select(row, 0)
		return vm.UndefinedValue, nil
	})
	m["set_selected_func"] = fn("set_selected_func", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		if !args[0].CanCall() {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "callable", Found: args[0].TypeName()}
		}
		cb := args[0]
		t.SetSelectedFunc(func(row, column int) {
			cb.Call(ctx, &vm.Int{Value: int64(row)}, &vm.Int{Value: int64(column)})
		})
		return vm.UndefinedValue, nil
	})
	m["set_selection_changed_func"] = fn("set_selection_changed_func", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		if !args[0].CanCall() {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "callable", Found: args[0].TypeName()}
		}
		cb := args[0]
		t.SetSelectionChangedFunc(func(row, column int) {
			cb.Call(ctx, &vm.Int{Value: int64(row)}, &vm.Int{Value: int64(column)})
		})
		return vm.UndefinedValue, nil
	})
	m["scroll_to_beginning"] = fn("scroll_to_beginning", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		t.ScrollToBeginning()
		return vm.UndefinedValue, nil
	})
	m["scroll_to_end"] = fn("scroll_to_end", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		t.ScrollToEnd()
		return vm.UndefinedValue, nil
	})
	return &vm.ImmutableMap{Value: m}
}

// -- Flex wrapper -----------------------------------------------------

func wrapFlex(f *Flex) *vm.ImmutableMap {
	m := make(map[string]vm.Object)
	m["__widget"] = &widgetPtr{w: f}
	addBoxMethods(m, f.Box)

	m["add_item"] = fn("add_item", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) < 1 || len(args) > 4 {
			return nil, vm.ErrWrongNumArguments
		}
		w := toWidget(args[0])
		if w == nil {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "widget", Found: args[0].TypeName()}
		}
		fixedSize := 0
		proportion := 1
		focus := false
		if len(args) >= 2 {
			if v, ok := vm.ToInt(args[1]); ok {
				fixedSize = v
			}
		}
		if len(args) >= 3 {
			if v, ok := vm.ToInt(args[2]); ok {
				proportion = v
			}
		}
		if len(args) >= 4 {
			focus = !args[3].IsFalsy()
		}
		f.AddItem(w, fixedSize, proportion, focus)
		return vm.UndefinedValue, nil
	})
	m["clear"] = fn("clear", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		f.Clear()
		return vm.UndefinedValue, nil
	})
	m["set_direction"] = fn("set_direction", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		d, ok := vm.ToInt(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "int", Found: args[0].TypeName()}
		}
		f.SetDirection(d)
		return vm.UndefinedValue, nil
	})
	m["set_full_screen"] = fn("set_full_screen", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		f.SetFullScreen(!args[0].IsFalsy())
		return vm.UndefinedValue, nil
	})
	m["remove_item"] = fn("remove_item", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		w := toWidget(args[0])
		if w == nil {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "widget", Found: args[0].TypeName()}
		}
		f.RemoveItem(w)
		return vm.UndefinedValue, nil
	})
	return &vm.ImmutableMap{Value: m}
}

// -- Grid wrapper -----------------------------------------------------

func wrapGrid(g *Grid) *vm.ImmutableMap {
	m := make(map[string]vm.Object)
	m["__widget"] = &widgetPtr{w: g}
	addBoxMethods(m, g.Box)

	m["set_rows"] = fn("set_rows", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		var rows []int
		for _, a := range args {
			v, ok := vm.ToInt(a)
			if !ok {
				return nil, vm.ErrInvalidArgumentType{Name: "row", Expected: "int", Found: a.TypeName()}
			}
			rows = append(rows, v)
		}
		g.SetRows(rows...)
		return vm.UndefinedValue, nil
	})
	m["set_columns"] = fn("set_columns", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		var cols []int
		for _, a := range args {
			v, ok := vm.ToInt(a)
			if !ok {
				return nil, vm.ErrInvalidArgumentType{Name: "column", Expected: "int", Found: a.TypeName()}
			}
			cols = append(cols, v)
		}
		g.SetColumns(cols...)
		return vm.UndefinedValue, nil
	})
	m["set_borders"] = fn("set_borders", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		g.SetBorders(!args[0].IsFalsy())
		return vm.UndefinedValue, nil
	})
	m["set_gap"] = fn("set_gap", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 2 {
			return nil, vm.ErrWrongNumArguments
		}
		r, ok := vm.ToInt(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "row", Expected: "int", Found: args[0].TypeName()}
		}
		c, ok := vm.ToInt(args[1])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "column", Expected: "int", Found: args[1].TypeName()}
		}
		g.SetGap(r, c)
		return vm.UndefinedValue, nil
	})
	m["add_item"] = fn("add_item", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		// add_item(widget, row, col, rowSpan, colSpan, minGridWidth, minGridHeight, focus)
		if len(args) != 8 {
			return nil, vm.ErrWrongNumArguments
		}
		w := toWidget(args[0])
		if w == nil {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "widget", Found: args[0].TypeName()}
		}
		row, _ := vm.ToInt(args[1])
		col, _ := vm.ToInt(args[2])
		rowSpan, _ := vm.ToInt(args[3])
		colSpan, _ := vm.ToInt(args[4])
		minW, _ := vm.ToInt(args[5])
		minH, _ := vm.ToInt(args[6])
		focus := !args[7].IsFalsy()
		g.AddItem(w, row, col, rowSpan, colSpan, minW, minH, focus)
		return vm.UndefinedValue, nil
	})
	return &vm.ImmutableMap{Value: m}
}

// -- Form wrapper -----------------------------------------------------

func wrapForm(f *Form) *vm.ImmutableMap {
	m := make(map[string]vm.Object)
	m["__widget"] = &widgetPtr{w: f}
	addBoxMethods(m, f.Box)

	m["add_input_field"] = fn("add_input_field", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) < 2 {
			return nil, vm.ErrWrongNumArguments
		}
		label, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "label", Expected: "string", Found: args[0].TypeName()}
		}
		value, ok := strictStr(args[1])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "value", Expected: "string", Found: args[1].TypeName()}
		}
		fieldWidth := 0
		if len(args) >= 3 {
			if v, ok := vm.ToInt(args[2]); ok {
				fieldWidth = v
			}
		}
		var changedFn func(string)
		if len(args) >= 4 && args[3].CanCall() {
			cb := args[3]
			changedFn = func(text string) {
				cb.Call(ctx, &vm.String{Value: text})
			}
		}
		f.AddInputField(label, value, fieldWidth, nil, changedFn)
		return vm.UndefinedValue, nil
	})
	m["add_password_field"] = fn("add_password_field", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) < 2 {
			return nil, vm.ErrWrongNumArguments
		}
		label, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "label", Expected: "string", Found: args[0].TypeName()}
		}
		value, ok := strictStr(args[1])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "value", Expected: "string", Found: args[1].TypeName()}
		}
		fieldWidth := 0
		if len(args) >= 3 {
			if v, ok := vm.ToInt(args[2]); ok {
				fieldWidth = v
			}
		}
		var changedFn func(string)
		if len(args) >= 4 && args[3].CanCall() {
			cb := args[3]
			changedFn = func(text string) {
				cb.Call(ctx, &vm.String{Value: text})
			}
		}
		f.AddPasswordField(label, value, fieldWidth, '*', changedFn)
		return vm.UndefinedValue, nil
	})
	m["add_check_box"] = fn("add_check_box", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) < 3 {
			return nil, vm.ErrWrongNumArguments
		}
		label, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "label", Expected: "string", Found: args[0].TypeName()}
		}
		message, ok := strictStr(args[1])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "message", Expected: "string", Found: args[1].TypeName()}
		}
		checked := !args[2].IsFalsy()
		var changedFn func(bool)
		if len(args) >= 4 && args[3].CanCall() {
			cb := args[3]
			changedFn = func(checked bool) {
				cb.Call(ctx, boolVal(checked))
			}
		}
		f.AddCheckBox(label, message, checked, changedFn)
		return vm.UndefinedValue, nil
	})
	m["add_button"] = fn("add_button", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) < 1 {
			return nil, vm.ErrWrongNumArguments
		}
		label, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "label", Expected: "string", Found: args[0].TypeName()}
		}
		var handler func()
		if len(args) >= 2 && args[1].CanCall() {
			cb := args[1]
			handler = func() {
				cb.Call(ctx)
			}
		}
		f.AddButton(label, handler)
		return vm.UndefinedValue, nil
	})
	m["clear"] = fn("clear", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		incButtons := true
		if len(args) >= 1 {
			incButtons = !args[0].IsFalsy()
		}
		f.Clear(incButtons)
		return vm.UndefinedValue, nil
	})
	m["set_horizontal"] = fn("set_horizontal", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		f.SetHorizontal(!args[0].IsFalsy())
		return vm.UndefinedValue, nil
	})
	m["set_cancel_func"] = fn("set_cancel_func", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		if !args[0].CanCall() {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "callable", Found: args[0].TypeName()}
		}
		cb := args[0]
		f.SetCancelFunc(func() {
			cb.Call(ctx)
		})
		return vm.UndefinedValue, nil
	})
	m["get_form_item_count"] = fn("get_form_item_count", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		return &vm.Int{Value: int64(f.GetFormItemCount())}, nil
	})
	m["get_button_count"] = fn("get_button_count", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		return &vm.Int{Value: int64(f.GetButtonCount())}, nil
	})
	return &vm.ImmutableMap{Value: m}
}

// -- Modal wrapper ----------------------------------------------------

func wrapModal(mod *Modal) *vm.ImmutableMap {
	m := make(map[string]vm.Object)
	m["__widget"] = &widgetPtr{w: mod}
	addBoxMethods(m, mod.Box)

	m["set_text"] = fn("set_text", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		s, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
		}
		mod.SetText(s)
		return vm.UndefinedValue, nil
	})
	m["add_buttons"] = fn("add_buttons", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		var labels []string
		for _, a := range args {
			s, ok := strictStr(a)
			if !ok {
				return nil, vm.ErrInvalidArgumentType{Name: "button", Expected: "string", Found: a.TypeName()}
			}
			labels = append(labels, s)
		}
		mod.AddButtons(labels)
		return vm.UndefinedValue, nil
	})
	m["set_done_func"] = fn("set_done_func", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		if !args[0].CanCall() {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "callable", Found: args[0].TypeName()}
		}
		cb := args[0]
		mod.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			cb.Call(ctx, &vm.Int{Value: int64(buttonIndex)}, &vm.String{Value: buttonLabel})
		})
		return vm.UndefinedValue, nil
	})
	return &vm.ImmutableMap{Value: m}
}

// -- Panels wrapper ---------------------------------------------------

func wrapPanels(p *Panels) *vm.ImmutableMap {
	m := make(map[string]vm.Object)
	m["__widget"] = &widgetPtr{w: p}
	addBoxMethods(m, p.Box)

	m["add_panel"] = fn("add_panel", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) < 3 {
			return nil, vm.ErrWrongNumArguments
		}
		name, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "name", Expected: "string", Found: args[0].TypeName()}
		}
		w := toWidget(args[1])
		if w == nil {
			return nil, vm.ErrInvalidArgumentType{Name: "widget", Expected: "widget", Found: args[1].TypeName()}
		}
		resize := !args[2].IsFalsy()
		visible := false
		if len(args) >= 4 {
			visible = !args[3].IsFalsy()
		}
		p.AddPanel(name, w, resize, visible)
		return vm.UndefinedValue, nil
	})
	m["remove_panel"] = fn("remove_panel", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		name, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "name", Expected: "string", Found: args[0].TypeName()}
		}
		p.RemovePanel(name)
		return vm.UndefinedValue, nil
	})
	m["show_panel"] = fn("show_panel", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		name, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "name", Expected: "string", Found: args[0].TypeName()}
		}
		p.ShowPanel(name)
		return vm.UndefinedValue, nil
	})
	m["hide_panel"] = fn("hide_panel", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		name, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "name", Expected: "string", Found: args[0].TypeName()}
		}
		p.HidePanel(name)
		return vm.UndefinedValue, nil
	})
	m["set_current_panel"] = fn("set_current_panel", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		name, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "name", Expected: "string", Found: args[0].TypeName()}
		}
		p.SetCurrentPanel(name)
		return vm.UndefinedValue, nil
	})
	m["get_panel_count"] = fn("get_panel_count", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		return &vm.Int{Value: int64(p.GetPanelCount())}, nil
	})
	return &vm.ImmutableMap{Value: m}
}

// -- TabbedPanels wrapper ---------------------------------------------

func wrapTabbedPanels(tp *TabbedPanels) *vm.ImmutableMap {
	m := make(map[string]vm.Object)
	m["__widget"] = &widgetPtr{w: tp}
	addBoxMethods(m, tp.Box)

	m["add_tab"] = fn("add_tab", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) < 3 {
			return nil, vm.ErrWrongNumArguments
		}
		name, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "name", Expected: "string", Found: args[0].TypeName()}
		}
		label, ok := strictStr(args[1])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "label", Expected: "string", Found: args[1].TypeName()}
		}
		w := toWidget(args[2])
		if w == nil {
			return nil, vm.ErrInvalidArgumentType{Name: "widget", Expected: "widget", Found: args[2].TypeName()}
		}
		tp.AddTab(name, label, w)
		return vm.UndefinedValue, nil
	})
	m["set_current_tab"] = fn("set_current_tab", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		name, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "name", Expected: "string", Found: args[0].TypeName()}
		}
		tp.SetCurrentTab(name)
		return vm.UndefinedValue, nil
	})
	return &vm.ImmutableMap{Value: m}
}

// -- TreeView wrapper -------------------------------------------------

func wrapTreeView(tv *TreeView) *vm.ImmutableMap {
	m := make(map[string]vm.Object)
	m["__widget"] = &widgetPtr{w: tv}
	addBoxMethods(m, tv.Box)

	m["set_root"] = fn("set_root", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		np := toTreeNode(args[0])
		if np == nil {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "tree_node", Found: args[0].TypeName()}
		}
		tv.SetRoot(np)
		return vm.UndefinedValue, nil
	})
	m["set_current_node"] = fn("set_current_node", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		np := toTreeNode(args[0])
		if np == nil {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "tree_node", Found: args[0].TypeName()}
		}
		tv.SetCurrentNode(np)
		return vm.UndefinedValue, nil
	})
	m["set_selected_func"] = fn("set_selected_func", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		if !args[0].CanCall() {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "callable", Found: args[0].TypeName()}
		}
		cb := args[0]
		tv.SetSelectedFunc(func(node *TreeNode) {
			cb.Call(ctx, wrapTreeNode(node))
		})
		return vm.UndefinedValue, nil
	})
	m["set_changed_func"] = fn("set_changed_func", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		if !args[0].CanCall() {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "callable", Found: args[0].TypeName()}
		}
		cb := args[0]
		tv.SetChangedFunc(func(node *TreeNode) {
			cb.Call(ctx, wrapTreeNode(node))
		})
		return vm.UndefinedValue, nil
	})
	return &vm.ImmutableMap{Value: m}
}

// -- TreeNode wrapper -------------------------------------------------

type treeNodePtr struct {
	vm.ObjectImpl
	n *TreeNode
}

func (o *treeNodePtr) TypeName() string { return "cui-tree-node-ptr" }
func (o *treeNodePtr) String() string   { return "<cui-tree-node-ptr>" }
func (o *treeNodePtr) Copy() vm.Object  { return o }

func toTreeNode(obj vm.Object) *TreeNode {
	if m, ok := obj.(*vm.ImmutableMap); ok {
		if w, ok := m.Value["__node"]; ok {
			if np, ok := w.(*treeNodePtr); ok {
				return np.n
			}
		}
	}
	return nil
}

func wrapTreeNode(n *TreeNode) *vm.ImmutableMap {
	m := make(map[string]vm.Object)
	m["__node"] = &treeNodePtr{n: n}

	m["set_text"] = fn("set_text", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		s, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
		}
		n.SetText(s)
		return vm.UndefinedValue, nil
	})
	m["get_text"] = fn("get_text", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		return &vm.String{Value: n.GetText()}, nil
	})
	m["add_child"] = fn("add_child", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		child := toTreeNode(args[0])
		if child == nil {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "tree_node", Found: args[0].TypeName()}
		}
		n.AddChild(child)
		return vm.UndefinedValue, nil
	})
	m["set_expanded"] = fn("set_expanded", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		n.SetExpanded(!args[0].IsFalsy())
		return vm.UndefinedValue, nil
	})
	m["set_selectable"] = fn("set_selectable", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		n.SetSelectable(!args[0].IsFalsy())
		return vm.UndefinedValue, nil
	})
	m["clear_children"] = fn("clear_children", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		n.ClearChildren()
		return vm.UndefinedValue, nil
	})
	m["get_children"] = fn("get_children", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		children := n.GetChildren()
		arr := &vm.Array{}
		for _, child := range children {
			arr.Value = append(arr.Value, wrapTreeNode(child))
		}
		return arr, nil
	})
	return &vm.ImmutableMap{Value: m}
}

// -- ProgressBar wrapper ----------------------------------------------

func wrapProgressBar(pb *ProgressBar) *vm.ImmutableMap {
	m := make(map[string]vm.Object)
	m["__widget"] = &widgetPtr{w: pb}
	addBoxMethods(m, pb.Box)

	m["set_max"] = fn("set_max", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		v, ok := vm.ToInt(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "int", Found: args[0].TypeName()}
		}
		pb.SetMax(v)
		return vm.UndefinedValue, nil
	})
	m["get_max"] = fn("get_max", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		return &vm.Int{Value: int64(pb.GetMax())}, nil
	})
	m["set_progress"] = fn("set_progress", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		v, ok := vm.ToInt(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "int", Found: args[0].TypeName()}
		}
		pb.SetProgress(v)
		return vm.UndefinedValue, nil
	})
	m["get_progress"] = fn("get_progress", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		return &vm.Int{Value: int64(pb.GetProgress())}, nil
	})
	m["add_progress"] = fn("add_progress", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		v, ok := vm.ToInt(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "int", Found: args[0].TypeName()}
		}
		pb.AddProgress(v)
		return vm.UndefinedValue, nil
	})
	m["complete"] = fn("complete", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		return boolVal(pb.Complete()), nil
	})
	m["set_vertical"] = fn("set_vertical", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		pb.SetVertical(!args[0].IsFalsy())
		return vm.UndefinedValue, nil
	})
	return &vm.ImmutableMap{Value: m}
}

// -- DropDown wrapper -------------------------------------------------

func wrapDropDown(dd *DropDown) *vm.ImmutableMap {
	m := make(map[string]vm.Object)
	m["__widget"] = &widgetPtr{w: dd}
	addBoxMethods(m, dd.Box)

	m["set_label"] = fn("set_label", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		s, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
		}
		dd.SetLabel(s)
		return vm.UndefinedValue, nil
	})
	m["get_label"] = fn("get_label", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		return &vm.String{Value: dd.GetLabel()}, nil
	})
	m["set_options"] = fn("set_options", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		var opts []string
		for _, a := range args {
			s, ok := strictStr(a)
			if !ok {
				return nil, vm.ErrInvalidArgumentType{Name: "option", Expected: "string", Found: a.TypeName()}
			}
			opts = append(opts, s)
		}
		dd.SetOptionsSimple(nil, opts...)
		return vm.UndefinedValue, nil
	})
	m["set_current_option"] = fn("set_current_option", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		idx, ok := vm.ToInt(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "int", Found: args[0].TypeName()}
		}
		dd.SetCurrentOption(idx)
		return vm.UndefinedValue, nil
	})
	m["get_current_option"] = fn("get_current_option", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		idx, opt := dd.GetCurrentOption()
		if opt == nil {
			return &vm.ImmutableMap{Value: map[string]vm.Object{
				"index": &vm.Int{Value: int64(idx)},
				"text":  vm.UndefinedValue,
			}}, nil
		}
		return &vm.ImmutableMap{Value: map[string]vm.Object{
			"index": &vm.Int{Value: int64(idx)},
			"text":  &vm.String{Value: opt.GetText()},
		}}, nil
	})
	return &vm.ImmutableMap{Value: m}
}

// -- Frame wrapper ----------------------------------------------------

func wrapFrame(fr *Frame) *vm.ImmutableMap {
	m := make(map[string]vm.Object)
	m["__widget"] = &widgetPtr{w: fr}
	addBoxMethods(m, fr.Box)

	m["add_text"] = fn("add_text", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) < 3 {
			return nil, vm.ErrWrongNumArguments
		}
		text, ok := strictStr(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "text", Expected: "string", Found: args[0].TypeName()}
		}
		header := !args[1].IsFalsy()
		align, ok := vm.ToInt(args[2])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "align", Expected: "int", Found: args[2].TypeName()}
		}
		fr.AddText(text, header, align, Styles.PrimaryTextColor)
		return vm.UndefinedValue, nil
	})
	m["clear"] = fn("clear", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		fr.Clear()
		return vm.UndefinedValue, nil
	})
	m["set_borders"] = fn("set_borders", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 6 {
			return nil, vm.ErrWrongNumArguments
		}
		top, _ := vm.ToInt(args[0])
		bottom, _ := vm.ToInt(args[1])
		header, _ := vm.ToInt(args[2])
		footer, _ := vm.ToInt(args[3])
		left, _ := vm.ToInt(args[4])
		right, _ := vm.ToInt(args[5])
		fr.SetBorders(top, bottom, header, footer, left, right)
		return vm.UndefinedValue, nil
	})
	return &vm.ImmutableMap{Value: m}
}

// strictStr returns the string value only if o is a *vm.String.
func strictStr(o vm.Object) (string, bool) {
	if s, ok := o.(*vm.String); ok {
		return s.Value, true
	}
	return "", false
}

// fn creates a BuiltinFunction.
func fn(name string, f vm.CallableFunc) vm.Object {
	return &vm.BuiltinFunction{Name: name, Value: f}
}

// helper: convert vm.Object to Widget (stored as *vm.ImmutableMap with "__widget" key)
func toWidget(obj vm.Object) Widget {
	if m, ok := obj.(*vm.ImmutableMap); ok {
		if w, ok := m.Value["__widget"]; ok {
			if wp, ok := w.(*widgetPtr); ok {
				return wp.w
			}
		}
	}
	return nil
}

// widgetPtr wraps a Widget so it can be stored in an ImmutableMap.
type widgetPtr struct {
	vm.ObjectImpl
	w Widget
}

func (o *widgetPtr) TypeName() string { return "cui-widget-ptr" }
func (o *widgetPtr) String() string   { return "<cui-widget-ptr>" }
func (o *widgetPtr) Copy() vm.Object  { return o }

// boolVal returns the appropriate vm.Object for a bool.
func boolVal(b bool) vm.Object {
	if b {
		return vm.TrueValue
	}
	return vm.FalseValue
}
