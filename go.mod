module github.com/malivvan/rumo

go 1.26.2

replace github.com/malivvan/readline => ../readline

require (
	github.com/ebitengine/purego v0.10.0
	github.com/malivvan/readline v0.0.0-00010101000000-000000000000
	golang.org/x/text v0.36.0
)

require golang.org/x/sys v0.43.0 // indirect
