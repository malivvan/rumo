// Demo code for the TreeView widget.
package main

import (
	"io/ioutil"
	"path/filepath"

	"github.com/gdamore/tcell/v3/color"
	"github.com/malivvan/rumo/std/cui"
)

// Show a navigable tree view of the current directory.
func main() {
	app := cui.NewApp()
	defer app.HandlePanic()

	app.EnableMouse(true)

	rootDir := "."
	root := cui.NewTreeNode(rootDir)
	root.SetColor(color.Red)
	tree := cui.NewTreeView()
	tree.SetRoot(root)
	tree.SetCurrentNode(root)

	// A helper function which adds the files and directories of the given path
	// to the given target node.
	add := func(target *cui.TreeNode, path string) {
		files, err := ioutil.ReadDir(path)
		if err != nil {
			panic(err)
		}
		for _, file := range files {
			node := cui.NewTreeNode(file.Name())
			node.SetReference(filepath.Join(path, file.Name()))
			node.SetSelectable(file.IsDir())
			if file.IsDir() {
				node.SetColor(color.Green)
			}
			target.AddChild(node)
		}
	}

	// Add the current directory to the root node.
	add(root, rootDir)

	// If a directory was selected, open it.
	tree.SetSelectedFunc(func(node *cui.TreeNode) {
		reference := node.GetReference()
		if reference == nil {
			return // Selecting the root node does nothing.
		}
		children := node.GetChildren()
		if len(children) == 0 {
			// Load and show files in this directory.
			path := reference.(string)
			add(node, path)
		} else {
			// Collapse if visible, expand if collapsed.
			node.SetExpanded(!node.IsExpanded())
		}
	})

	app.SetRoot(tree, true)
	if err := app.Run(); err != nil {
		panic(err)
	}
}
