package main

import (
	"log"

	"github.com/malivvan/rumo/std/cui"
)

func clickedMessageFn(msg string) func(*cui.MenuItem) {
	return func(*cui.MenuItem) { log.Printf("%v clicked\n", msg) }
}

func main() {
	app := cui.NewApp()

	fileMenu := cui.NewMenuItem("File")
	fileMenu.AddItem(cui.NewMenuItem("New File").SetOnClick(clickedMessageFn("New File")))
	fileMenu.AddItem(cui.NewMenuItem("Open File").SetOnClick(clickedMessageFn("Open File")))

	saveSubForReal := cui.NewMenuItem("Save For Real").
		AddItem(cui.NewMenuItem("For really real").SetOnClick(clickedMessageFn("For really real"))).
		AddItem(cui.NewMenuItem("For really fake").SetOnClick(clickedMessageFn("For really fake")))
	saveSubForFake := cui.NewMenuItem("Save For Fake").SetOnClick(clickedMessageFn("Safe for fake"))

	fileMenu.AddItem(cui.NewMenuItem("Save File").
		// Add submenu items to save
		AddItem(saveSubForReal).
		AddItem(saveSubForFake).SetOnClick(clickedMessageFn("Save File")))

	fileMenu.AddItem(cui.NewMenuItem("Close File").SetOnClick(clickedMessageFn("Close File")))
	fileMenu.AddItem(cui.NewMenuItem("Exit").SetOnClick(func(*cui.MenuItem) { app.Stop() }))
	editMenu := cui.NewMenuItem("Edit")
	editMenu.AddItem(cui.NewMenuItem("Copy").SetOnClick(clickedMessageFn("Copy")))
	editMenu.AddItem(cui.NewMenuItem("Cut").SetOnClick(clickedMessageFn("Cut")))
	editMenu.AddItem(cui.NewMenuItem("Paste").SetOnClick(clickedMessageFn("Paste")))

	menuBar := cui.NewMenuBar().
		AddItem(fileMenu).
		AddItem(editMenu)

	menuBar.SetRect(0, 0, 100, 15)

	box := cui.NewBox()
	box.SetBorder(true)
	box.SetTitle("Hello, world!")

	flex := cui.NewFlex()
	flex.SetDirection(cui.FlexRow)
	flex.AddItem(menuBar, 1, 1, false)
	flex.AddItem(box, 0, 1, true)

	app.EnableMouse(true)
	app.SetRoot(flex, true)
	app.SetFocus(flex)
	app.SetAfterDrawFunc(menuBar.AfterDraw())

	if err := app.Run(); err != nil {
		panic(err)
	}
}
