package vte

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCUPDefaultsAndOriginMode(t *testing.T) {
	vt := New()
	vt.Resize(10, 5)

	vt.cursor.row = 4
	vt.cursor.col = 9
	vt.cup([]int{0, 0})
	assert.Equal(t, row(0), vt.cursor.row)
	assert.Equal(t, column(0), vt.cursor.col)

	vt.margin.top = 1
	vt.margin.bottom = 3
	vt.mode |= decom
	vt.cup([]int{1, 1})
	assert.Equal(t, row(1), vt.cursor.row)
	assert.Equal(t, column(0), vt.cursor.col)

	vt.cup([]int{999, 999})
	assert.Equal(t, row(3), vt.cursor.row)
	assert.Equal(t, column(9), vt.cursor.col)
}

func TestDECSTBMDefaultsAndHome(t *testing.T) {
	vt := New()
	vt.Resize(10, 4)

	vt.cursor.row = 3
	vt.cursor.col = 7
	vt.decstbm([]int{0, 3})
	assert.Equal(t, row(0), vt.margin.top)
	assert.Equal(t, row(2), vt.margin.bottom)
	assert.Equal(t, row(0), vt.cursor.row)
	assert.Equal(t, column(0), vt.cursor.col)

	vt.mode |= decom
	vt.cursor.row = 0
	vt.cursor.col = 9
	vt.decstbm([]int{2, 4})
	assert.Equal(t, row(1), vt.margin.top)
	assert.Equal(t, row(3), vt.margin.bottom)
	assert.Equal(t, row(1), vt.cursor.row)
	assert.Equal(t, column(0), vt.cursor.col)
}

func TestPrivateDSRResponses(t *testing.T) {
	vt := New()
	vt.Resize(10, 5)
	vt.margin.top = 1
	vt.margin.bottom = 3
	vt.mode |= decom
	vt.cursor.row = 2
	vt.cursor.col = 4

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	defer func() {
		_ = r.Close()
	}()
	vt.pty = w

	vt.csi("?n", []int{5})
	vt.csi("?n", []int{6})
	assert.NoError(t, w.Close())

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	assert.Equal(t, "\x1b[?13n\x1b[?2;5R", string(out))
}

func TestDECOMHomesCursor(t *testing.T) {
	vt := New()
	vt.Resize(10, 5)
	vt.margin.top = 2
	vt.margin.bottom = 4
	vt.cursor.row = 4
	vt.cursor.col = 8

	vt.decset([]int{6})
	assert.Equal(t, row(2), vt.cursor.row)
	assert.Equal(t, column(0), vt.cursor.col)

	vt.cursor.row = 4
	vt.cursor.col = 8
	vt.decrst([]int{6})
	assert.Equal(t, row(0), vt.cursor.row)
	assert.Equal(t, column(0), vt.cursor.col)
}

func TestAlternateScreenModeVariants(t *testing.T) {
	t.Run("47 and 1047 switch screens without clobbering primary", func(t *testing.T) {
		vt := New()
		vt.Resize(2, 1)
		vt.print('p')

		vt.decset([]int{47})
		vt.cursor.row = 0
		vt.cursor.col = 0
		vt.print('a')
		vt.decrst([]int{47})
		assert.Equal(t, "p ", vt.String())

		vt.decset([]int{1047})
		vt.cursor.row = 0
		vt.cursor.col = 0
		vt.print('b')
		vt.decrst([]int{1047})
		assert.Equal(t, "p ", vt.String())
	})

	t.Run("1048 saves and restores cursor", func(t *testing.T) {
		vt := New()
		vt.Resize(2, 2)
		vt.cursor.row = 1
		vt.cursor.col = 1

		vt.decset([]int{1048})
		vt.cursor.row = 0
		vt.cursor.col = 0
		vt.decrst([]int{1048})

		assert.Equal(t, row(1), vt.cursor.row)
		assert.Equal(t, column(1), vt.cursor.col)
	})
}

