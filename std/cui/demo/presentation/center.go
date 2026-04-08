package main

import "github.com/malivvan/rumo/std/cui"

// Center returns a new widget which shows the provided widget in its
// center, given the provided widget's size.
func Center(width, height int, p cui.Widget) cui.Widget {
	subFlex := cui.NewFlex()
	subFlex.SetDirection(cui.FlexRow)
	subFlex.AddItem(cui.NewBox(), 0, 1, false)
	subFlex.AddItem(p, height, 1, true)
	subFlex.AddItem(cui.NewBox(), 0, 1, false)

	flex := cui.NewFlex()
	flex.AddItem(cui.NewBox(), 0, 1, false)
	flex.AddItem(subFlex, width, 1, true)
	flex.AddItem(cui.NewBox(), 0, 1, false)

	return flex
}
