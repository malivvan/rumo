package vm

import "fmt"

// RangeObject is a lazy, iterable representation of an integer range produced
// by the built-in range(start, stop[, step]) function.  It does not allocate
// the individual element values until they are actually consumed by a for-loop
// or explicit iteration, avoiding O(N) memory usage for large ranges.
type RangeObject struct {
	ObjectImpl
	Start int64
	Stop  int64
	Step  int64 // always > 0
}

// rangeLen returns the number of elements the range produces.
func rangeLen(start, stop, step int64) int64 {
	if start == stop {
		return 0
	}
	if start < stop {
		// e.g. range(0, 5, 2) → 3 elements  ⌈(5-0)/2⌉
		return (stop - start + step - 1) / step
	}
	// e.g. range(0, -5, 2) → 3 elements  ⌈(0-(-5))/2⌉
	return (start - stop + step - 1) / step
}

// TypeName implements Object.
func (r *RangeObject) TypeName() string { return "range" }

// String implements Object.
func (r *RangeObject) String() string {
	return fmt.Sprintf("range(%d, %d, %d)", r.Start, r.Stop, r.Step)
}

// IsFalsy returns true when the range contains no elements.
func (r *RangeObject) IsFalsy() bool { return rangeLen(r.Start, r.Stop, r.Step) == 0 }

// Equals reports whether x is a RangeObject with identical parameters.
func (r *RangeObject) Equals(x Object) bool {
	o, ok := x.(*RangeObject)
	if !ok {
		return false
	}
	return r.Start == o.Start && r.Stop == o.Stop && r.Step == o.Step
}

// Copy returns a shallow copy of the RangeObject (all fields are value types).
func (r *RangeObject) Copy() Object {
	return &RangeObject{Start: r.Start, Stop: r.Stop, Step: r.Step}
}

// CanIterate always returns true.
func (r *RangeObject) CanIterate() bool { return true }

// Iterate returns a RangeIterator positioned before the first element.
func (r *RangeObject) Iterate() Iterator {
	return &RangeIterator{
		start: r.Start,
		stop:  r.Stop,
		step:  r.Step,
		len:   rangeLen(r.Start, r.Stop, r.Step),
		i:     0,
	}
}

// IndexGet returns the element at position index (0-based).
// Out-of-bounds access returns UndefinedValue (consistent with Array).
func (r *RangeObject) IndexGet(index Object) (Object, error) {
	idx, ok := index.(*Int)
	if !ok {
		return nil, ErrInvalidIndexType
	}
	i := idx.Value
	length := rangeLen(r.Start, r.Stop, r.Step)
	if i < 0 || i >= length {
		return UndefinedValue, nil
	}
	return NewInt(rangeValueAt(r.Start, r.Stop, r.Step, i)), nil
}

// rangeValueAt computes the i-th value of range(start, stop, step).
func rangeValueAt(start, stop, step, i int64) int64 {
	if start <= stop {
		return start + i*step
	}
	return start - i*step
}

// ---------------------------------------------------------------------------
// RangeIterator
// ---------------------------------------------------------------------------

// RangeIterator is a lazy iterator over a RangeObject.  It computes each
// element on demand, using O(1) memory regardless of range size.
type RangeIterator struct {
	ObjectImpl
	start int64
	stop  int64
	step  int64
	len   int64 // total number of elements
	i     int64 // 1-based current position (0 = before first Next())
}

// TypeName implements Object.
func (it *RangeIterator) TypeName() string { return "range-iterator" }

// String implements Object.
func (it *RangeIterator) String() string { return "<range-iterator>" }

// IsFalsy implements Object.
func (it *RangeIterator) IsFalsy() bool { return true }

// Equals implements Object.
func (it *RangeIterator) Equals(Object) bool { return false }

// Copy returns an independent copy of the iterator at the same position.
func (it *RangeIterator) Copy() Object {
	c := *it
	return &c
}

// Next advances the iterator and returns true if a value is available.
func (it *RangeIterator) Next() bool {
	it.i++
	return it.i <= it.len
}

// Key returns the 0-based index of the current element.
func (it *RangeIterator) Key() Object {
	return NewInt(it.i - 1)
}

// Value returns the current element without allocating the full range.
func (it *RangeIterator) Value() Object {
	return NewInt(rangeValueAt(it.start, it.stop, it.step, it.i-1))
}

