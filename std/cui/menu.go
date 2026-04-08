package cui

import (
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
)

type MenuItem struct {
	*Box
	Title    string
	SubItems []*MenuItem
	onClick  func(*MenuItem)
}

func NewMenuItem(title string) *MenuItem {
	return &MenuItem{
		Box:      NewBox(),
		Title:    title,
		SubItems: make([]*MenuItem, 0),
	}
}

func (menuItem *MenuItem) AddItem(item *MenuItem) *MenuItem {
	menuItem.SubItems = append(menuItem.SubItems, item)
	return menuItem
}

func (menuItem *MenuItem) SetOnClick(fn func(*MenuItem)) *MenuItem {
	menuItem.onClick = fn
	return menuItem
}

func (menuItem *MenuItem) Draw(screen tcell.Screen) {
	menuItem.Box.Draw(screen)
	x, y, _, _ := menuItem.GetInnerRect()
	PrintSimple(screen, []byte(menuItem.Title), x, y)
}

type SubMenu struct {
	*Box
	Items         []*MenuItem
	parent        *MenuBar
	childMenu     *SubMenu
	currentSelect int
}

func NewSubMenu(parent *MenuBar, items []*MenuItem) *SubMenu {
	subMenu := &SubMenu{
		Box:           NewBox(),
		Items:         items,
		parent:        parent,
		currentSelect: -1,
	}
	subMenu.SetBorder(true)
	return subMenu
}

func (subMenu *SubMenu) Draw(screen tcell.Screen) {
	anySubItems := false
	maxWidth := 0
	for _, item := range subMenu.Items {
		if itemTitleLen := len(item.Title); itemTitleLen > maxWidth {
			maxWidth = itemTitleLen
		}
		if len(item.SubItems) > 0 {
			anySubItems = true
		}
	}

	rectX, rectY, _, _ := subMenu.GetRect()
	rectWid := maxWidth
	if anySubItems {
		rectWid += 1
	}
	rectHig := len(subMenu.Items)
	// +2 - add space one space for each side of rect - to fit text inside
	subMenu.SetRect(rectX, rectY, rectWid+2, rectHig+2)
	subMenu.Box.Draw(screen)

	x, y, _, _ := subMenu.GetInnerRect()
	for i, item := range subMenu.Items {
		if i == subMenu.currentSelect {
			Print(screen, []byte(item.Title), x, y+i, 20, 0, color.Blue)
			if len(item.SubItems) > 0 {
				Print(screen, []byte(">"), x+maxWidth, y+i, 20, 0, color.Blue)
			}
			continue
		}
		PrintSimple(screen, []byte(item.Title), x, y+i)
		if len(item.SubItems) > 0 {
			PrintSimple(screen, []byte(">"), x+maxWidth, y+i)
		}
	}
	if subMenu.childMenu != nil {
		subMenu.childMenu.Draw(screen)
	}
}

func (subMenu *SubMenu) MouseHandler() func(action MouseAction, event *tcell.EventMouse, setFocus func(w Widget)) (consumed bool, capture Widget) {
	return subMenu.WrapMouseHandler(func(action MouseAction, event *tcell.EventMouse, setFocus func(w Widget)) (consumed bool, capture Widget) {
		if subMenu.childMenu != nil {
			consumed, capture = subMenu.childMenu.MouseHandler()(action, event, setFocus)

			if consumed {
				return
			}
		}
		rectX, rectY, rectW, _ := subMenu.Box.GetInnerRect()
		if !subMenu.Box.InRect(event.Position()) {
			// Close the menu if the user clicks outside the menu box
			if action == MouseLeftClick {
				subMenu.parent.subMenu = nil
			}
			return false, nil
		}
		_, y := event.Position()
		index := y - rectY

		subMenu.currentSelect = index
		consumed = true

		if action == MouseLeftClick {
			setFocus(subMenu)
			if index >= 0 && index < len(subMenu.Items) {
				handler := subMenu.Items[index].onClick
				if handler != nil {
					handler(subMenu.Items[index])
				}
				if len(subMenu.Items[index].SubItems) > 0 {
					subMenu.childMenu = NewSubMenu(subMenu.parent, subMenu.Items[index].SubItems)
					subMenu.childMenu.SetRect(rectX+rectW, y, 15, 10)
					return
				}
			}
			subMenu.parent.subMenu = nil
		}
		return
	})
}

type MenuBar struct {
	*Box
	MenuItems     []*MenuItem
	subMenu       *SubMenu // sub menu if not nil will be drawn
	currentOption int
}

func NewMenuBar() *MenuBar {
	return &MenuBar{
		Box:       NewBox(),
		MenuItems: make([]*MenuItem, 0),
	}
}

func (menuBar *MenuBar) AfterDraw() func(tcell.Screen) {
	return func(screen tcell.Screen) {
		if menuBar.subMenu != nil {
			menuBar.subMenu.Draw(screen)
		}
	}
}

func (menuBar *MenuBar) AddItem(item *MenuItem) *MenuBar {
	menuBar.MenuItems = append(menuBar.MenuItems, item)
	return menuBar
}

func (menuBar *MenuBar) Draw(screen tcell.Screen) {
	menuBar.Box.Draw(screen)

	x, y, width, _ := menuBar.GetInnerRect()

	for i := 0; i < width; i += 1 {
		screen.SetContent(x+i, y, tcell.RuneBlock, nil, tcell.StyleDefault)
	}

	menuItemOffset := 1
	for _, mi := range menuBar.MenuItems {
		itemLen := len([]rune(mi.Title))
		mi.SetRect(menuItemOffset, y, itemLen, 1)
		mi.Draw(screen)
		menuItemOffset += itemLen + 1
	}
}

func (menuBar *MenuBar) InputHandler() func(event *tcell.EventKey, setFocus func(w Widget)) {
	return menuBar.WrapInputHandler(func(event *tcell.EventKey, setFocus func(w Widget)) {
		switch event.Key() {
		case tcell.KeyLeft:
			menuBar.currentOption--
			if menuBar.currentOption < 0 {
				menuBar.currentOption = -1
			}
		case tcell.KeyRight:
			menuBar.currentOption++
			if menuBar.currentOption >= len(menuBar.MenuItems) {
				menuBar.currentOption = len(menuBar.MenuItems) - 1
			}
		}
	})
}

func (menuBar *MenuBar) MouseHandler() func(action MouseAction, event *tcell.EventMouse, setFocus func(w Widget)) (consumed bool, capture Widget) {
	return menuBar.WrapMouseHandler(func(action MouseAction, event *tcell.EventMouse, setFocus func(w Widget)) (consumed bool, capture Widget) {
		if menuBar.subMenu != nil {
			consumed, capture = menuBar.subMenu.MouseHandler()(action, event, setFocus)
			if consumed {
				//p.subMenu = nil
				return
			}
		}
		if !menuBar.InRect(event.Position()) {
			return false, nil
		}
		// Pass mouse events down.
		for _, item := range menuBar.MenuItems {
			consumed, capture = item.MouseHandler()(action, event, setFocus)
			if consumed {
				menuBar.subMenu = NewSubMenu(menuBar, item.SubItems)
				x, y, _, _ := item.GetRect()
				menuBar.subMenu.Box.SetRect(x+1, y+1, 15, 10)
				return
			}
		}

		// ...handle mouse events not directed to the child widget...
		return true, nil
	})
}

func (menuBar *MenuBar) Focus(delegate func(w Widget)) {
	//if menuBar.subMenu != nil {
	//	delegate(menuBar.subMenu)
	//} else {
	menuBar.Box.Focus(delegate)
	menuBar.subMenu = nil
	//}
}
