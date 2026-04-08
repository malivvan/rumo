package cui_test

import (
	"testing"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/require"
)

// -- Constructor tests ----------------------------------------------------

func TestNewBox(t *testing.T) {
	box := require.Module(t, "cui").Call("new_box")
	box.Call("get_title").Expect("")
	box.Call("get_visible").Expect(true)
}

func TestNewTextView(t *testing.T) {
	tv := require.Module(t, "cui").Call("new_text_view")
	tv.Call("get_text", false).Expect("")
	tv.Call("get_title").Expect("")
}

func TestNewButton(t *testing.T) {
	btn := require.Module(t, "cui").Call("new_button", "Click Me")
	btn.Call("get_label").Expect("Click Me")
}

func TestNewCheckBox(t *testing.T) {
	cb := require.Module(t, "cui").Call("new_check_box")
	cb.Call("is_checked").Expect(false)
	cb.Call("get_label").Expect("")
}

func TestNewInputField(t *testing.T) {
	inp := require.Module(t, "cui").Call("new_input_field")
	inp.Call("get_label").Expect("")
	inp.Call("get_text").Expect("")
}

func TestNewList(t *testing.T) {
	l := require.Module(t, "cui").Call("new_list")
	l.Call("get_item_count").Expect(0)
	l.Call("get_current_item_index").Expect(0)
}

func TestNewTable(t *testing.T) {
	tbl := require.Module(t, "cui").Call("new_table")
	tbl.Call("get_row_count").Expect(0)
	tbl.Call("get_column_count").Expect(0)
}

func TestNewProgressBar(t *testing.T) {
	pb := require.Module(t, "cui").Call("new_progress_bar")
	pb.Call("get_progress").Expect(0)
	pb.Call("get_max").Expect(100)
	pb.Call("complete").Expect(false)
}

func TestNewDropDown(t *testing.T) {
	dd := require.Module(t, "cui").Call("new_drop_down")
	dd.Call("get_label").Expect("")
}

func TestNewFlex(t *testing.T) {
	require.Module(t, "cui").Call("new_flex").Call("get_title").Expect("")
}

func TestNewGrid(t *testing.T) {
	require.Module(t, "cui").Call("new_grid").Call("get_title").Expect("")
}

func TestNewForm(t *testing.T) {
	f := require.Module(t, "cui").Call("new_form")
	f.Call("get_form_item_count").Expect(0)
	f.Call("get_button_count").Expect(0)
}

func TestNewModal(t *testing.T) {
	require.Module(t, "cui").Call("new_modal").Call("get_title").Expect("")
}

func TestNewPanels(t *testing.T) {
	require.Module(t, "cui").Call("new_panels").Call("get_panel_count").Expect(0)
}

func TestNewTabbedPanels(t *testing.T) {
	require.Module(t, "cui").Call("new_tabbed_panels").Call("get_title").Expect("")
}

func TestNewTreeView(t *testing.T) {
	require.Module(t, "cui").Call("new_tree_view").Call("get_title").Expect("")
}

func TestNewTreeNode(t *testing.T) {
	n := require.Module(t, "cui").Call("new_tree_node", "Root")
	n.Call("get_text").Expect("Root")
}

// -- Constructor argument validation tests --------------------------------

func TestConstructorWrongArgs(t *testing.T) {
	require.Module(t, "cui").Call("new_box", "extra").ExpectError()
	require.Module(t, "cui").Call("new_text_view", "extra").ExpectError()
	require.Module(t, "cui").Call("new_button").ExpectError()
	require.Module(t, "cui").Call("new_check_box", "extra").ExpectError()
	require.Module(t, "cui").Call("new_input_field", "extra").ExpectError()
	require.Module(t, "cui").Call("new_list", "extra").ExpectError()
	require.Module(t, "cui").Call("new_table", "extra").ExpectError()
	require.Module(t, "cui").Call("new_flex", "extra").ExpectError()
	require.Module(t, "cui").Call("new_grid", "extra").ExpectError()
	require.Module(t, "cui").Call("new_form", "extra").ExpectError()
	require.Module(t, "cui").Call("new_modal", "extra").ExpectError()
	require.Module(t, "cui").Call("new_panels", "extra").ExpectError()
	require.Module(t, "cui").Call("new_tabbed_panels", "extra").ExpectError()
	require.Module(t, "cui").Call("new_tree_view", "extra").ExpectError()
	require.Module(t, "cui").Call("new_tree_node").ExpectError()
	require.Module(t, "cui").Call("new_progress_bar", "extra").ExpectError()
	require.Module(t, "cui").Call("new_drop_down", "extra").ExpectError()
	require.Module(t, "cui").Call("new_frame").ExpectError()
	require.Module(t, "cui").Call("new_frame", 123).ExpectError()
}

// -- Box methods tests (shared base) ------------------------------------

func TestBoxSetTitle(t *testing.T) {
	box := require.Module(t, "cui").Call("new_box")
	box.Call("get_title").Expect("")
	box.Call("set_title", "Hello").Expect(vm.UndefinedValue)
	box.Call("get_title").Expect("Hello")
	box.Call("set_title", "World").Expect(vm.UndefinedValue)
	box.Call("get_title").Expect("World")
}

func TestBoxSetBorder(t *testing.T) {
	box := require.Module(t, "cui").Call("new_box")
	box.Call("set_border", true).Expect(vm.UndefinedValue)
	box.Call("set_border", false).Expect(vm.UndefinedValue)
}

func TestBoxSetVisible(t *testing.T) {
	box := require.Module(t, "cui").Call("new_box")
	box.Call("get_visible").Expect(true)
	box.Call("set_visible", false).Expect(vm.UndefinedValue)
	box.Call("get_visible").Expect(false)
	box.Call("set_visible", true).Expect(vm.UndefinedValue)
	box.Call("get_visible").Expect(true)
}

func TestBoxSetRect(t *testing.T) {
	box := require.Module(t, "cui").Call("new_box")
	box.Call("set_rect", 1, 2, 30, 40).Expect(vm.UndefinedValue)
}

func TestBoxSetPadding(t *testing.T) {
	box := require.Module(t, "cui").Call("new_box")
	box.Call("set_padding", 1, 2, 3, 4).Expect(vm.UndefinedValue)
}

func TestBoxWrongArgs(t *testing.T) {
	box := require.Module(t, "cui").Call("new_box")
	box.Call("set_title").ExpectError()
	box.Call("set_border").ExpectError()
	box.Call("set_visible").ExpectError()
	box.Call("set_rect", 1).ExpectError()
	box.Call("set_rect", 1, 2).ExpectError()
	box.Call("set_rect", 1, 2, 3).ExpectError()
	box.Call("set_rect", "bad", 0, 0, 0).ExpectError()
	box.Call("set_padding", 1).ExpectError()
}

// -- TextView tests -------------------------------------------------------

func TestTextViewSetText(t *testing.T) {
	tv := require.Module(t, "cui").Call("new_text_view")
	tv.Call("get_text", false).Expect("")
	tv.Call("set_text", "Hello World").Expect(vm.UndefinedValue)
	tv.Call("get_text", false).Expect("Hello World")
	tv.Call("clear").Expect(vm.UndefinedValue)
	tv.Call("get_text", false).Expect("")
}

func TestTextViewOptions(t *testing.T) {
	tv := require.Module(t, "cui").Call("new_text_view")
	tv.Call("set_text_align", 0).Expect(vm.UndefinedValue)
	tv.Call("set_dynamic_colors", true).Expect(vm.UndefinedValue)
	tv.Call("set_regions", true).Expect(vm.UndefinedValue)
	tv.Call("set_scrollable", true).Expect(vm.UndefinedValue)
	tv.Call("set_wrap", true).Expect(vm.UndefinedValue)
	tv.Call("set_word_wrap", true).Expect(vm.UndefinedValue)
	tv.Call("set_max_lines", 100).Expect(vm.UndefinedValue)
	tv.Call("scroll_to_end").Expect(vm.UndefinedValue)
	tv.Call("scroll_to_beginning").Expect(vm.UndefinedValue)
}

func TestTextViewWrongArgs(t *testing.T) {
	tv := require.Module(t, "cui").Call("new_text_view")
	tv.Call("set_text").ExpectError()
	tv.Call("set_text_align").ExpectError()
	tv.Call("set_text_align", "bad").ExpectError()
	tv.Call("set_dynamic_colors").ExpectError()
	tv.Call("set_regions").ExpectError()
	tv.Call("set_scrollable").ExpectError()
	tv.Call("set_wrap").ExpectError()
	tv.Call("set_word_wrap").ExpectError()
	tv.Call("set_max_lines").ExpectError()
	tv.Call("set_max_lines", "bad").ExpectError()
}

// -- Button tests ---------------------------------------------------------

func TestButtonSetLabel(t *testing.T) {
	btn := require.Module(t, "cui").Call("new_button", "OK")
	btn.Call("get_label").Expect("OK")
	btn.Call("set_label", "Cancel").Expect(vm.UndefinedValue)
	btn.Call("get_label").Expect("Cancel")
}

func TestButtonWrongArgs(t *testing.T) {
	btn := require.Module(t, "cui").Call("new_button", "OK")
	btn.Call("set_label").ExpectError()
	btn.Call("set_selected_func").ExpectError()
	btn.Call("set_selected_func", "not_callable").ExpectError()
}

// -- CheckBox tests -------------------------------------------------------

func TestCheckBoxSetChecked(t *testing.T) {
	cb := require.Module(t, "cui").Call("new_check_box")
	cb.Call("is_checked").Expect(false)
	cb.Call("set_checked", true).Expect(vm.UndefinedValue)
	cb.Call("is_checked").Expect(true)
	cb.Call("set_checked", false).Expect(vm.UndefinedValue)
	cb.Call("is_checked").Expect(false)
}

func TestCheckBoxSetLabel(t *testing.T) {
	cb := require.Module(t, "cui").Call("new_check_box")
	cb.Call("get_label").Expect("")
	cb.Call("set_label", "Accept terms").Expect(vm.UndefinedValue)
	cb.Call("get_label").Expect("Accept terms")
}

func TestCheckBoxSetMessage(t *testing.T) {
	cb := require.Module(t, "cui").Call("new_check_box")
	cb.Call("set_message", "I agree").Expect(vm.UndefinedValue)
}

func TestCheckBoxWrongArgs(t *testing.T) {
	cb := require.Module(t, "cui").Call("new_check_box")
	cb.Call("set_label").ExpectError()
	cb.Call("set_checked").ExpectError()
	cb.Call("set_message").ExpectError()
	cb.Call("set_changed_func").ExpectError()
	cb.Call("set_changed_func", "not_callable").ExpectError()
}

// -- InputField tests -----------------------------------------------------

func TestInputFieldSetText(t *testing.T) {
	inp := require.Module(t, "cui").Call("new_input_field")
	inp.Call("get_text").Expect("")
	inp.Call("set_text", "hello").Expect(vm.UndefinedValue)
	inp.Call("get_text").Expect("hello")
}

func TestInputFieldSetLabel(t *testing.T) {
	inp := require.Module(t, "cui").Call("new_input_field")
	inp.Call("get_label").Expect("")
	inp.Call("set_label", "Name:").Expect(vm.UndefinedValue)
	inp.Call("get_label").Expect("Name:")
}

func TestInputFieldOptions(t *testing.T) {
	inp := require.Module(t, "cui").Call("new_input_field")
	inp.Call("set_placeholder", "Enter text...").Expect(vm.UndefinedValue)
	inp.Call("set_field_width", 20).Expect(vm.UndefinedValue)
}

func TestInputFieldWrongArgs(t *testing.T) {
	inp := require.Module(t, "cui").Call("new_input_field")
	inp.Call("set_label").ExpectError()
	inp.Call("set_text").ExpectError()
	inp.Call("set_placeholder").ExpectError()
	inp.Call("set_field_width").ExpectError()
	inp.Call("set_field_width", "bad").ExpectError()
	inp.Call("set_changed_func").ExpectError()
	inp.Call("set_changed_func", "not_callable").ExpectError()
}

// -- List tests -----------------------------------------------------------

func TestListAddItem(t *testing.T) {
	l := require.Module(t, "cui").Call("new_list")
	l.Call("get_item_count").Expect(0)
	l.Call("add_item", "Item A").Expect(vm.UndefinedValue)
	l.Call("get_item_count").Expect(1)
	l.Call("add_item", "Item B", "secondary").Expect(vm.UndefinedValue)
	l.Call("get_item_count").Expect(2)
	l.Call("get_current_item_index").Expect(0)
}

func TestListSetCurrentItem(t *testing.T) {
	l := require.Module(t, "cui").Call("new_list")
	l.Call("add_item", "A").Expect(vm.UndefinedValue)
	l.Call("add_item", "B").Expect(vm.UndefinedValue)
	l.Call("set_current_item", 1).Expect(vm.UndefinedValue)
	l.Call("get_current_item_index").Expect(1)
}

func TestListClear(t *testing.T) {
	l := require.Module(t, "cui").Call("new_list")
	l.Call("add_item", "A").Expect(vm.UndefinedValue)
	l.Call("add_item", "B").Expect(vm.UndefinedValue)
	l.Call("get_item_count").Expect(2)
	l.Call("clear").Expect(vm.UndefinedValue)
	l.Call("get_item_count").Expect(0)
}

func TestListRemoveItem(t *testing.T) {
	l := require.Module(t, "cui").Call("new_list")
	l.Call("add_item", "A").Expect(vm.UndefinedValue)
	l.Call("add_item", "B").Expect(vm.UndefinedValue)
	l.Call("remove_item", 0).Expect(vm.UndefinedValue)
	l.Call("get_item_count").Expect(1)
}

func TestListOptions(t *testing.T) {
	l := require.Module(t, "cui").Call("new_list")
	l.Call("show_secondary_text", true).Expect(vm.UndefinedValue)
	l.Call("set_highlight_full_line", true).Expect(vm.UndefinedValue)
}

func TestListWrongArgs(t *testing.T) {
	l := require.Module(t, "cui").Call("new_list")
	l.Call("add_item").ExpectError()
	l.Call("set_current_item").ExpectError()
	l.Call("set_current_item", "bad").ExpectError()
	l.Call("remove_item").ExpectError()
	l.Call("remove_item", "bad").ExpectError()
	l.Call("show_secondary_text").ExpectError()
	l.Call("set_highlight_full_line").ExpectError()
	l.Call("set_selected_func").ExpectError()
	l.Call("set_selected_func", "not_callable").ExpectError()
	l.Call("set_changed_func").ExpectError()
	l.Call("set_changed_func", "not_callable").ExpectError()
}

// -- Table tests ----------------------------------------------------------

func TestTableSetCell(t *testing.T) {
	tbl := require.Module(t, "cui").Call("new_table")
	tbl.Call("get_row_count").Expect(0)
	tbl.Call("set_cell", 0, 0, "A1").Expect(vm.UndefinedValue)
	tbl.Call("set_cell", 0, 1, "B1").Expect(vm.UndefinedValue)
	tbl.Call("set_cell", 1, 0, "A2").Expect(vm.UndefinedValue)
	tbl.Call("get_row_count").Expect(2)
	tbl.Call("get_column_count").Expect(2)
	tbl.Call("get_cell_text", 0, 0).Expect("A1")
	tbl.Call("get_cell_text", 0, 1).Expect("B1")
	tbl.Call("get_cell_text", 1, 0).Expect("A2")
}

func TestTableClear(t *testing.T) {
	tbl := require.Module(t, "cui").Call("new_table")
	tbl.Call("set_cell", 0, 0, "A1").Expect(vm.UndefinedValue)
	tbl.Call("clear").Expect(vm.UndefinedValue)
	tbl.Call("get_row_count").Expect(0)
}

func TestTableOptions(t *testing.T) {
	tbl := require.Module(t, "cui").Call("new_table")
	tbl.Call("set_borders", true).Expect(vm.UndefinedValue)
	tbl.Call("set_fixed", 1, 1).Expect(vm.UndefinedValue)
	tbl.Call("set_selectable", true, true).Expect(vm.UndefinedValue)
	tbl.Call("scroll_to_beginning").Expect(vm.UndefinedValue)
	tbl.Call("scroll_to_end").Expect(vm.UndefinedValue)
}

func TestTableWrongArgs(t *testing.T) {
	tbl := require.Module(t, "cui").Call("new_table")
	tbl.Call("set_cell", 0, 0).ExpectError()
	tbl.Call("set_cell", 0).ExpectError()
	tbl.Call("set_cell", "bad", 0, "text").ExpectError()
	tbl.Call("set_cell", 0, "bad", "text").ExpectError()
	tbl.Call("set_cell", 0, 0, 123).ExpectError()
	tbl.Call("get_cell_text").ExpectError()
	tbl.Call("get_cell_text", 0).ExpectError()
	tbl.Call("get_cell_text", "bad", 0).ExpectError()
	tbl.Call("get_cell_text", 0, "bad").ExpectError()
	tbl.Call("set_borders").ExpectError()
	tbl.Call("set_fixed").ExpectError()
	tbl.Call("set_fixed", 1).ExpectError()
	tbl.Call("set_selectable").ExpectError()
	tbl.Call("set_selectable", true).ExpectError()
	tbl.Call("select_row").ExpectError()
	tbl.Call("select_row", "bad").ExpectError()
	tbl.Call("set_selected_func").ExpectError()
	tbl.Call("set_selected_func", "not_callable").ExpectError()
	tbl.Call("set_selection_changed_func").ExpectError()
	tbl.Call("set_selection_changed_func", "not_callable").ExpectError()
}

// -- ProgressBar tests ----------------------------------------------------

func TestProgressBarSetProgress(t *testing.T) {
	pb := require.Module(t, "cui").Call("new_progress_bar")
	pb.Call("get_progress").Expect(0)
	pb.Call("set_progress", 50).Expect(vm.UndefinedValue)
	pb.Call("get_progress").Expect(50)
	pb.Call("complete").Expect(false)
	pb.Call("set_progress", 100).Expect(vm.UndefinedValue)
	pb.Call("get_progress").Expect(100)
	pb.Call("complete").Expect(true)
}

func TestProgressBarSetMax(t *testing.T) {
	pb := require.Module(t, "cui").Call("new_progress_bar")
	pb.Call("get_max").Expect(100)
	pb.Call("set_max", 200).Expect(vm.UndefinedValue)
	pb.Call("get_max").Expect(200)
}

func TestProgressBarAddProgress(t *testing.T) {
	pb := require.Module(t, "cui").Call("new_progress_bar")
	pb.Call("add_progress", 25).Expect(vm.UndefinedValue)
	pb.Call("get_progress").Expect(25)
	pb.Call("add_progress", 25).Expect(vm.UndefinedValue)
	pb.Call("get_progress").Expect(50)
	pb.Call("add_progress", 50).Expect(vm.UndefinedValue)
	pb.Call("get_progress").Expect(100)
	pb.Call("complete").Expect(true)
}

func TestProgressBarOptions(t *testing.T) {
	pb := require.Module(t, "cui").Call("new_progress_bar")
	pb.Call("set_vertical", true).Expect(vm.UndefinedValue)
}

func TestProgressBarWrongArgs(t *testing.T) {
	pb := require.Module(t, "cui").Call("new_progress_bar")
	pb.Call("set_max").ExpectError()
	pb.Call("set_max", "bad").ExpectError()
	pb.Call("set_progress").ExpectError()
	pb.Call("set_progress", "bad").ExpectError()
	pb.Call("add_progress").ExpectError()
	pb.Call("add_progress", "bad").ExpectError()
	pb.Call("set_vertical").ExpectError()
}

// -- DropDown tests -------------------------------------------------------

func TestDropDownSetLabel(t *testing.T) {
	dd := require.Module(t, "cui").Call("new_drop_down")
	dd.Call("get_label").Expect("")
	dd.Call("set_label", "Choose:").Expect(vm.UndefinedValue)
	dd.Call("get_label").Expect("Choose:")
}

func TestDropDownSetOptions(t *testing.T) {
	dd := require.Module(t, "cui").Call("new_drop_down")
	dd.Call("set_options", "A", "B", "C").Expect(vm.UndefinedValue)
	dd.Call("set_current_option", 1).Expect(vm.UndefinedValue)
}

func TestDropDownWrongArgs(t *testing.T) {
	dd := require.Module(t, "cui").Call("new_drop_down")
	dd.Call("set_label").ExpectError()
	dd.Call("set_label", 123).ExpectError()
	dd.Call("set_options", 123).ExpectError()
	dd.Call("set_current_option").ExpectError()
	dd.Call("set_current_option", "bad").ExpectError()
}

// -- Flex tests -----------------------------------------------------------

func TestFlexSetDirection(t *testing.T) {
	f := require.Module(t, "cui").Call("new_flex")
	f.Call("set_direction", 0).Expect(vm.UndefinedValue)
	f.Call("set_full_screen", true).Expect(vm.UndefinedValue)
}

func TestFlexAddItem(t *testing.T) {
	f := require.Module(t, "cui").Call("new_flex")
	box := require.Module(t, "cui").Call("new_box")
	f.Call("add_item", box.O, 0, 1, false).Expect(vm.UndefinedValue)
}

func TestFlexWrongArgs(t *testing.T) {
	f := require.Module(t, "cui").Call("new_flex")
	f.Call("add_item").ExpectError()
	f.Call("add_item", "not_widget").ExpectError()
	f.Call("set_direction").ExpectError()
	f.Call("set_direction", "bad").ExpectError()
	f.Call("set_full_screen").ExpectError()
	f.Call("remove_item").ExpectError()
	f.Call("remove_item", "not_widget").ExpectError()
}

// -- Grid tests -----------------------------------------------------------

func TestGridSetRowsCols(t *testing.T) {
	g := require.Module(t, "cui").Call("new_grid")
	g.Call("set_rows", 10, 20, 30).Expect(vm.UndefinedValue)
	g.Call("set_columns", 10, 20).Expect(vm.UndefinedValue)
	g.Call("set_borders", true).Expect(vm.UndefinedValue)
	g.Call("set_gap", 1, 1).Expect(vm.UndefinedValue)
}

func TestGridAddItem(t *testing.T) {
	g := require.Module(t, "cui").Call("new_grid")
	box := require.Module(t, "cui").Call("new_box")
	g.Call("add_item", box.O, 0, 0, 1, 1, 0, 0, false).Expect(vm.UndefinedValue)
}

func TestGridWrongArgs(t *testing.T) {
	g := require.Module(t, "cui").Call("new_grid")
	g.Call("set_rows", "bad").ExpectError()
	g.Call("set_columns", "bad").ExpectError()
	g.Call("set_borders").ExpectError()
	g.Call("set_gap").ExpectError()
	g.Call("set_gap", 1).ExpectError()
	g.Call("set_gap", "bad", 1).ExpectError()
	g.Call("set_gap", 1, "bad").ExpectError()
	g.Call("add_item", 1, 0, 0, 1, 1, 0, 0, false).ExpectError()
}

// -- Form tests -----------------------------------------------------------

func TestFormAddItems(t *testing.T) {
	f := require.Module(t, "cui").Call("new_form")
	f.Call("get_form_item_count").Expect(0)
	f.Call("add_input_field", "Name:", "").Expect(vm.UndefinedValue)
	f.Call("get_form_item_count").Expect(1)
	f.Call("add_password_field", "Password:", "").Expect(vm.UndefinedValue)
	f.Call("get_form_item_count").Expect(2)
	f.Call("add_check_box", "Accept", "", false).Expect(vm.UndefinedValue)
	f.Call("get_form_item_count").Expect(3)
	f.Call("add_button", "Submit").Expect(vm.UndefinedValue)
	f.Call("get_button_count").Expect(1)
}

func TestFormClear(t *testing.T) {
	f := require.Module(t, "cui").Call("new_form")
	f.Call("add_input_field", "Name:", "").Expect(vm.UndefinedValue)
	f.Call("add_button", "OK").Expect(vm.UndefinedValue)
	f.Call("get_form_item_count").Expect(1)
	f.Call("get_button_count").Expect(1)
	f.Call("clear", true).Expect(vm.UndefinedValue)
	f.Call("get_form_item_count").Expect(0)
	f.Call("get_button_count").Expect(0)
}

func TestFormOptions(t *testing.T) {
	f := require.Module(t, "cui").Call("new_form")
	f.Call("set_horizontal", true).Expect(vm.UndefinedValue)
}

func TestFormWrongArgs(t *testing.T) {
	f := require.Module(t, "cui").Call("new_form")
	f.Call("add_input_field").ExpectError()
	f.Call("add_input_field", "label").ExpectError()
	f.Call("add_input_field", 123, "").ExpectError()
	f.Call("add_input_field", "label", 123).ExpectError()
	f.Call("add_password_field").ExpectError()
	f.Call("add_password_field", "label").ExpectError()
	f.Call("add_password_field", 123, "").ExpectError()
	f.Call("add_password_field", "label", 123).ExpectError()
	f.Call("add_check_box").ExpectError()
	f.Call("add_check_box", "label").ExpectError()
	f.Call("add_check_box", "label", "msg").ExpectError()
	f.Call("add_check_box", 123, "msg", false).ExpectError()
	f.Call("add_check_box", "label", 123, false).ExpectError()
	f.Call("add_button").ExpectError()
	f.Call("add_button", 123).ExpectError()
	f.Call("set_horizontal").ExpectError()
	f.Call("set_cancel_func").ExpectError()
	f.Call("set_cancel_func", "not_callable").ExpectError()
}

// -- Modal tests ----------------------------------------------------------

func TestModalSetText(t *testing.T) {
	m := require.Module(t, "cui").Call("new_modal")
	m.Call("set_text", "Are you sure?").Expect(vm.UndefinedValue)
}

func TestModalAddButtons(t *testing.T) {
	m := require.Module(t, "cui").Call("new_modal")
	m.Call("add_buttons", "Yes", "No").Expect(vm.UndefinedValue)
}

func TestModalWrongArgs(t *testing.T) {
	m := require.Module(t, "cui").Call("new_modal")
	m.Call("set_text").ExpectError()
	m.Call("set_text", 123).ExpectError()
	m.Call("add_buttons", 123).ExpectError()
	m.Call("set_done_func").ExpectError()
	m.Call("set_done_func", "not_callable").ExpectError()
}

// -- Panels tests ---------------------------------------------------------

func TestPanelsAddPanel(t *testing.T) {
	p := require.Module(t, "cui").Call("new_panels")
	box := require.Module(t, "cui").Call("new_box")
	p.Call("get_panel_count").Expect(0)
	p.Call("add_panel", "p1", box.O, true, true).Expect(vm.UndefinedValue)
	p.Call("get_panel_count").Expect(1)
}

func TestPanelsShowHide(t *testing.T) {
	p := require.Module(t, "cui").Call("new_panels")
	box := require.Module(t, "cui").Call("new_box")
	p.Call("add_panel", "p1", box.O, true, false).Expect(vm.UndefinedValue)
	p.Call("show_panel", "p1").Expect(vm.UndefinedValue)
	p.Call("hide_panel", "p1").Expect(vm.UndefinedValue)
}

func TestPanelsRemovePanel(t *testing.T) {
	p := require.Module(t, "cui").Call("new_panels")
	box := require.Module(t, "cui").Call("new_box")
	p.Call("add_panel", "p1", box.O, true, true).Expect(vm.UndefinedValue)
	p.Call("get_panel_count").Expect(1)
	p.Call("remove_panel", "p1").Expect(vm.UndefinedValue)
	p.Call("get_panel_count").Expect(0)
}

func TestPanelsSetCurrentPanel(t *testing.T) {
	p := require.Module(t, "cui").Call("new_panels")
	box1 := require.Module(t, "cui").Call("new_box")
	box2 := require.Module(t, "cui").Call("new_box")
	p.Call("add_panel", "p1", box1.O, true, true).Expect(vm.UndefinedValue)
	p.Call("add_panel", "p2", box2.O, true, false).Expect(vm.UndefinedValue)
	p.Call("set_current_panel", "p2").Expect(vm.UndefinedValue)
}

func TestPanelsWrongArgs(t *testing.T) {
	p := require.Module(t, "cui").Call("new_panels")
	p.Call("add_panel").ExpectError()
	p.Call("add_panel", "name").ExpectError()
	p.Call("add_panel", "name", "not_widget").ExpectError()
	p.Call("add_panel", 123, "w", true).ExpectError()
	p.Call("remove_panel").ExpectError()
	p.Call("remove_panel", 123).ExpectError()
	p.Call("show_panel").ExpectError()
	p.Call("show_panel", 123).ExpectError()
	p.Call("hide_panel").ExpectError()
	p.Call("hide_panel", 123).ExpectError()
	p.Call("set_current_panel").ExpectError()
	p.Call("set_current_panel", 123).ExpectError()
}

// -- TabbedPanels tests ---------------------------------------------------

func TestTabbedPanelsAddTab(t *testing.T) {
	tp := require.Module(t, "cui").Call("new_tabbed_panels")
	box := require.Module(t, "cui").Call("new_box")
	tp.Call("add_tab", "t1", "Tab 1", box.O).Expect(vm.UndefinedValue)
}

func TestTabbedPanelsSetCurrentTab(t *testing.T) {
	tp := require.Module(t, "cui").Call("new_tabbed_panels")
	box1 := require.Module(t, "cui").Call("new_box")
	box2 := require.Module(t, "cui").Call("new_box")
	tp.Call("add_tab", "t1", "Tab 1", box1.O).Expect(vm.UndefinedValue)
	tp.Call("add_tab", "t2", "Tab 2", box2.O).Expect(vm.UndefinedValue)
	tp.Call("set_current_tab", "t2").Expect(vm.UndefinedValue)
}

func TestTabbedPanelsWrongArgs(t *testing.T) {
	tp := require.Module(t, "cui").Call("new_tabbed_panels")
	tp.Call("add_tab").ExpectError()
	tp.Call("add_tab", "name").ExpectError()
	tp.Call("add_tab", "name", "label").ExpectError()
	tp.Call("add_tab", 123, "label", "widget").ExpectError()
	tp.Call("add_tab", "name", 123, "widget").ExpectError()
	tp.Call("add_tab", "name", "label", "not_widget").ExpectError()
	tp.Call("set_current_tab").ExpectError()
	tp.Call("set_current_tab", 123).ExpectError()
}

// -- TreeView / TreeNode tests --------------------------------------------

func TestTreeNodeSetText(t *testing.T) {
	n := require.Module(t, "cui").Call("new_tree_node", "Root")
	n.Call("get_text").Expect("Root")
	n.Call("set_text", "Updated").Expect(vm.UndefinedValue)
	n.Call("get_text").Expect("Updated")
}

func TestTreeNodeExpandSelectable(t *testing.T) {
	n := require.Module(t, "cui").Call("new_tree_node", "Root")
	n.Call("set_expanded", true).Expect(vm.UndefinedValue)
	n.Call("set_selectable", true).Expect(vm.UndefinedValue)
}

func TestTreeNodeAddChild(t *testing.T) {
	root := require.Module(t, "cui").Call("new_tree_node", "Root")
	child := require.Module(t, "cui").Call("new_tree_node", "Child")
	root.Call("add_child", child.O).Expect(vm.UndefinedValue)
}

func TestTreeNodeClearChildren(t *testing.T) {
	root := require.Module(t, "cui").Call("new_tree_node", "Root")
	child := require.Module(t, "cui").Call("new_tree_node", "Child")
	root.Call("add_child", child.O).Expect(vm.UndefinedValue)
	root.Call("clear_children").Expect(vm.UndefinedValue)
}

func TestTreeNodeWrongArgs(t *testing.T) {
	n := require.Module(t, "cui").Call("new_tree_node", "Root")
	n.Call("set_text").ExpectError()
	n.Call("set_text", 123).ExpectError()
	n.Call("set_expanded").ExpectError()
	n.Call("set_selectable").ExpectError()
	n.Call("add_child").ExpectError()
	n.Call("add_child", "not_node").ExpectError()
}

func TestTreeViewSetRoot(t *testing.T) {
	tv := require.Module(t, "cui").Call("new_tree_view")
	root := require.Module(t, "cui").Call("new_tree_node", "Root")
	tv.Call("set_root", root.O).Expect(vm.UndefinedValue)
}

func TestTreeViewSetCurrentNode(t *testing.T) {
	tv := require.Module(t, "cui").Call("new_tree_view")
	root := require.Module(t, "cui").Call("new_tree_node", "Root")
	tv.Call("set_root", root.O).Expect(vm.UndefinedValue)
	tv.Call("set_current_node", root.O).Expect(vm.UndefinedValue)
}

func TestTreeViewWrongArgs(t *testing.T) {
	tv := require.Module(t, "cui").Call("new_tree_view")
	tv.Call("set_root").ExpectError()
	tv.Call("set_root", "not_node").ExpectError()
	tv.Call("set_current_node").ExpectError()
	tv.Call("set_current_node", "not_node").ExpectError()
	tv.Call("set_selected_func").ExpectError()
	tv.Call("set_selected_func", "not_callable").ExpectError()
	tv.Call("set_changed_func").ExpectError()
	tv.Call("set_changed_func", "not_callable").ExpectError()
}

// -- Frame tests ----------------------------------------------------------

func TestFrameAddText(t *testing.T) {
	box := require.Module(t, "cui").Call("new_box")
	fr := require.Module(t, "cui").Call("new_frame", box.O)
	fr.Call("add_text", "Header", true, 0).Expect(vm.UndefinedValue)
	fr.Call("add_text", "Footer", false, 1).Expect(vm.UndefinedValue)
	fr.Call("clear").Expect(vm.UndefinedValue)
}

func TestFrameSetBorders(t *testing.T) {
	box := require.Module(t, "cui").Call("new_box")
	fr := require.Module(t, "cui").Call("new_frame", box.O)
	fr.Call("set_borders", 1, 1, 0, 0, 1, 1).Expect(vm.UndefinedValue)
}

func TestFrameWrongArgs(t *testing.T) {
	box := require.Module(t, "cui").Call("new_box")
	fr := require.Module(t, "cui").Call("new_frame", box.O)
	fr.Call("add_text").ExpectError()
	fr.Call("add_text", "text").ExpectError()
	fr.Call("add_text", "text", true).ExpectError()
	fr.Call("add_text", 123, true, 0).ExpectError()
	fr.Call("add_text", "text", true, "bad").ExpectError()
	fr.Call("set_borders").ExpectError()
	fr.Call("set_borders", 1, 1, 1, 1, 1).ExpectError()
}

// -- Constants tests ------------------------------------------------------

func TestConstants(t *testing.T) {
	require.Module(t, "cui").Call("new_flex").Call("set_direction", 0).Expect(vm.UndefinedValue) // FlexRow = 0
	require.Module(t, "cui").Call("new_flex").Call("set_direction", 1).Expect(vm.UndefinedValue) // FlexColumn = 1
}

