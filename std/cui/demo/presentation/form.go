package main

import (
	"github.com/malivvan/rumo/std/cui"
)

// Form demonstrates forms.
func Form(nextSlide func()) (title string, info string, content cui.Widget) {
	f := cui.NewForm()
	f.AddInputField("First name:", "", 20, nil, nil)
	f.AddInputField("Last name:", "", 20, nil, nil)
	f.AddDropDownSimple("Role:", 0, nil, "Engineer", "Manager", "Administration")
	f.AddPasswordField("Password:", "", 10, '*', nil)
	f.AddCheckBox("", "On vacation", false, nil)
	f.AddButton("Save", nextSlide)
	f.AddButton("Cancel", nextSlide)
	f.SetBorder(true)
	f.SetTitle("Employee Information")
	return "Form", formInfo, Code(f, 36, 15, "form")
}
