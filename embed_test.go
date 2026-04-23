package rumo_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/malivvan/rumo"
	"github.com/malivvan/rumo/vm/require"
)

// writeTemp writes content to a file inside dir and returns the file path.
func writeTemp(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEmbed_SingleFileString(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "hello.txt", "hello world")

	src := `
//embed hello.txt
content := ""
`
	s := rumo.NewScript([]byte(src))
	require.NoError(t, s.SetImportDir(dir))

	compiled, err := s.Compile()
	require.NoError(t, err)

	err = compiled.RunContext(context.Background())
	require.NoError(t, err)

	v := compiled.Get("content")
	require.NotNil(t, v)
	require.Equal(t, "hello world", v.String())
}

func TestEmbed_SingleFileBytes(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "data.bin", "binary data")

	src := `
//embed data.bin
content := bytes("")
`
	s := rumo.NewScript([]byte(src))
	require.NoError(t, s.SetImportDir(dir))

	compiled, err := s.Compile()
	require.NoError(t, err)

	err = compiled.RunContext(context.Background())
	require.NoError(t, err)

	v := compiled.Get("content")
	require.NotNil(t, v)
	require.Equal(t, "binary data", v.String())
}

func TestEmbed_MultiFileStringMap(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "a.txt", "file a")
	writeTemp(t, dir, "b.txt", "file b")

	src := `
//embed *.txt
files := {}
`
	s := rumo.NewScript([]byte(src))
	require.NoError(t, s.SetImportDir(dir))

	compiled, err := s.Compile()
	require.NoError(t, err)

	err = compiled.RunContext(context.Background())
	require.NoError(t, err)

	v := compiled.Get("files")
	require.NotNil(t, v)

	m, ok := v.Value().(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, 2, len(m))
	require.Equal(t, "file a", m["a.txt"])
	require.Equal(t, "file b", m["b.txt"])
}

func TestEmbed_MultiFileBytesMap(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "c.txt", "file c")
	writeTemp(t, dir, "d.txt", "file d")

	src := `
//embed *.txt
files := bytes({})
`
	s := rumo.NewScript([]byte(src))
	require.NoError(t, s.SetImportDir(dir))

	compiled, err := s.Compile()
	require.NoError(t, err)

	err = compiled.RunContext(context.Background())
	require.NoError(t, err)

	v := compiled.Get("files")
	require.NotNil(t, v)

	m, ok := v.Value().(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, 2, len(m))
}

func TestEmbed_MultiplePatterns(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "readme.md", "# readme")
	writeTemp(t, dir, "config.json", "{}")

	src := `
//embed readme.md config.json
assets := {}
`
	s := rumo.NewScript([]byte(src))
	require.NoError(t, s.SetImportDir(dir))

	compiled, err := s.Compile()
	require.NoError(t, err)

	err = compiled.RunContext(context.Background())
	require.NoError(t, err)

	v := compiled.Get("assets")
	require.NotNil(t, v)

	m, ok := v.Value().(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, 2, len(m))
}

func TestEmbed_SubdirectoryPattern(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "sub/file1.txt", "sub file 1")
	writeTemp(t, dir, "sub/file2.txt", "sub file 2")

	src := `
//embed sub/*.txt
files := {}
`
	s := rumo.NewScript([]byte(src))
	require.NoError(t, s.SetImportDir(dir))

	compiled, err := s.Compile()
	require.NoError(t, err)

	err = compiled.RunContext(context.Background())
	require.NoError(t, err)

	v := compiled.Get("files")
	require.NotNil(t, v)

	m, ok := v.Value().(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, 2, len(m))
}

func TestEmbed_NoImportDir(t *testing.T) {
	src := `
//embed hello.txt
content := ""
`
	s := rumo.NewScript([]byte(src))
	// No SetImportDir call — importDir is empty.
	_, err := s.Compile()
	require.Error(t, err)
}

func TestEmbed_PatternNoMatch(t *testing.T) {
	dir := t.TempDir()

	src := `
//embed *.nonexistent
files := {}
`
	s := rumo.NewScript([]byte(src))
	require.NoError(t, s.SetImportDir(dir))

	_, err := s.Compile()
	require.Error(t, err)
}

func TestEmbed_SingleFileMultipleMatches(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "a.txt", "a")
	writeTemp(t, dir, "b.txt", "b")

	src := `
//embed *.txt
content := ""
`
	s := rumo.NewScript([]byte(src))
	require.NoError(t, s.SetImportDir(dir))

	_, err := s.Compile()
	require.Error(t, err) // glob matches 2 files but target is a single string
}

func TestEmbed_UsedInExpression(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "greeting.txt", "Hello")

	src := `
//embed greeting.txt
msg := ""
result := msg + ", World!"
`
	s := rumo.NewScript([]byte(src))
	require.NoError(t, s.SetImportDir(dir))

	compiled, err := s.Compile()
	require.NoError(t, err)

	err = compiled.RunContext(context.Background())
	require.NoError(t, err)

	v := compiled.Get("result")
	require.NotNil(t, v)
	require.Equal(t, "Hello, World!", v.String())
}

