package vm_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/malivvan/vv/vm"
)

func Test_builtinDelete(t *testing.T) {
	var builtinDelete func(ctx context.Context, args ...vm.Object) (vm.Object, error)
	for _, f := range vm.GetAllBuiltinFunctions() {
		if f.Name == "delete" {
			builtinDelete = f.Value
			break
		}
	}
	if builtinDelete == nil {
		t.Fatal("builtin delete not found")
	}
	type args struct {
		args []vm.Object
	}
	tests := []struct {
		name      string
		args      args
		want      vm.Object
		wantErr   bool
		wantedErr error
		target    interface{}
	}{
		{name: "invalid-arg", args: args{[]vm.Object{&vm.String{},
			&vm.String{}}}, wantErr: true,
			wantedErr: vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "map",
				Found:    "string"},
		},
		{name: "no-args",
			wantErr: true, wantedErr: vm.ErrWrongNumArguments},
		{name: "empty-args", args: args{[]vm.Object{}}, wantErr: true,
			wantedErr: vm.ErrWrongNumArguments,
		},
		{name: "3-args", args: args{[]vm.Object{
			(*vm.Map)(nil), (*vm.String)(nil), (*vm.String)(nil)}},
			wantErr: true, wantedErr: vm.ErrWrongNumArguments,
		},
		{name: "nil-map-empty-key",
			args: args{[]vm.Object{&vm.Map{}, &vm.String{}}},
			want: vm.UndefinedValue,
		},
		{name: "nil-map-nonstr-key",
			args: args{[]vm.Object{
				&vm.Map{}, &vm.Int{}}}, wantErr: true,
			wantedErr: vm.ErrInvalidArgumentType{
				Name: "second", Expected: "string", Found: "int"},
		},
		{name: "nil-map-no-key",
			args: args{[]vm.Object{&vm.Map{}}}, wantErr: true,
			wantedErr: vm.ErrWrongNumArguments,
		},
		{name: "map-missing-key",
			args: args{
				[]vm.Object{
					&vm.Map{Value: map[string]vm.Object{
						"key": &vm.String{Value: "value"},
					}},
					&vm.String{Value: "key1"}}},
			want: vm.UndefinedValue,
			target: &vm.Map{
				Value: map[string]vm.Object{
					"key": &vm.String{
						Value: "value"}}},
		},
		{name: "map-emptied",
			args: args{
				[]vm.Object{
					&vm.Map{Value: map[string]vm.Object{
						"key": &vm.String{Value: "value"},
					}},
					&vm.String{Value: "key"}}},
			want:   vm.UndefinedValue,
			target: &vm.Map{Value: map[string]vm.Object{}},
		},
		{name: "map-multi-keys",
			args: args{
				[]vm.Object{
					&vm.Map{Value: map[string]vm.Object{
						"key1": &vm.String{Value: "value1"},
						"key2": &vm.Int{Value: 10},
					}},
					&vm.String{Value: "key1"}}},
			want: vm.UndefinedValue,
			target: &vm.Map{Value: map[string]vm.Object{
				"key2": &vm.Int{Value: 10}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := builtinDelete(context.Background(), tt.args.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("builtinDelete() error = %v, wantErr %v",
					err, tt.wantErr)
				return
			}
			if tt.wantErr && !errors.Is(err, tt.wantedErr) {
				if err.Error() != tt.wantedErr.Error() {
					t.Errorf("builtinDelete() error = %v, wantedErr %v",
						err, tt.wantedErr)
					return
				}
			}
			if got != tt.want {
				t.Errorf("builtinDelete() = %v, want %v", got, tt.want)
				return
			}
			if !tt.wantErr && tt.target != nil {
				switch v := tt.args.args[0].(type) {
				case *vm.Map, *vm.Array:
					if !reflect.DeepEqual(tt.target, tt.args.args[0]) {
						t.Errorf("builtinDelete() objects are not equal "+
							"got: %+v, want: %+v", tt.args.args[0], tt.target)
					}
				default:
					t.Errorf("builtinDelete() unsuporrted arg[0] type %s",
						v.TypeName())
					return
				}
			}
		})
	}
}

func Test_builtinSplice(t *testing.T) {
	var builtinSplice func(ctx context.Context, args ...vm.Object) (vm.Object, error)
	for _, f := range vm.GetAllBuiltinFunctions() {
		if f.Name == "splice" {
			builtinSplice = f.Value
			break
		}
	}
	if builtinSplice == nil {
		t.Fatal("builtin splice not found")
	}
	tests := []struct {
		name      string
		args      []vm.Object
		deleted   vm.Object
		Array     *vm.Array
		wantErr   bool
		wantedErr error
	}{
		{name: "no args", args: []vm.Object{}, wantErr: true,
			wantedErr: vm.ErrWrongNumArguments,
		},
		{name: "invalid args", args: []vm.Object{&vm.Map{}},
			wantErr: true,
			wantedErr: vm.ErrInvalidArgumentType{
				Name: "first", Expected: "array", Found: "map"},
		},
		{name: "invalid args",
			args:    []vm.Object{&vm.Array{}, &vm.String{}},
			wantErr: true,
			wantedErr: vm.ErrInvalidArgumentType{
				Name: "second", Expected: "int", Found: "string"},
		},
		{name: "negative index",
			args:      []vm.Object{&vm.Array{}, &vm.Int{Value: -1}},
			wantErr:   true,
			wantedErr: vm.ErrIndexOutOfBounds},
		{name: "non int count",
			args: []vm.Object{
				&vm.Array{}, &vm.Int{Value: 0},
				&vm.String{Value: ""}},
			wantErr: true,
			wantedErr: vm.ErrInvalidArgumentType{
				Name: "third", Expected: "int", Found: "string"},
		},
		{name: "negative count",
			args: []vm.Object{
				&vm.Array{Value: []vm.Object{
					&vm.Int{Value: 0},
					&vm.Int{Value: 1},
					&vm.Int{Value: 2}}},
				&vm.Int{Value: 0},
				&vm.Int{Value: -1}},
			wantErr:   true,
			wantedErr: vm.ErrIndexOutOfBounds,
		},
		{name: "insert with zero count",
			args: []vm.Object{
				&vm.Array{Value: []vm.Object{
					&vm.Int{Value: 0},
					&vm.Int{Value: 1},
					&vm.Int{Value: 2}}},
				&vm.Int{Value: 0},
				&vm.Int{Value: 0},
				&vm.String{Value: "b"}},
			deleted: &vm.Array{Value: []vm.Object{}},
			Array: &vm.Array{Value: []vm.Object{
				&vm.String{Value: "b"},
				&vm.Int{Value: 0},
				&vm.Int{Value: 1},
				&vm.Int{Value: 2}}},
		},
		{name: "insert",
			args: []vm.Object{
				&vm.Array{Value: []vm.Object{
					&vm.Int{Value: 0},
					&vm.Int{Value: 1},
					&vm.Int{Value: 2}}},
				&vm.Int{Value: 1},
				&vm.Int{Value: 0},
				&vm.String{Value: "c"},
				&vm.String{Value: "d"}},
			deleted: &vm.Array{Value: []vm.Object{}},
			Array: &vm.Array{Value: []vm.Object{
				&vm.Int{Value: 0},
				&vm.String{Value: "c"},
				&vm.String{Value: "d"},
				&vm.Int{Value: 1},
				&vm.Int{Value: 2}}},
		},
		{name: "insert with zero count",
			args: []vm.Object{
				&vm.Array{Value: []vm.Object{
					&vm.Int{Value: 0},
					&vm.Int{Value: 1},
					&vm.Int{Value: 2}}},
				&vm.Int{Value: 1},
				&vm.Int{Value: 0},
				&vm.String{Value: "c"},
				&vm.String{Value: "d"}},
			deleted: &vm.Array{Value: []vm.Object{}},
			Array: &vm.Array{Value: []vm.Object{
				&vm.Int{Value: 0},
				&vm.String{Value: "c"},
				&vm.String{Value: "d"},
				&vm.Int{Value: 1},
				&vm.Int{Value: 2}}},
		},
		{name: "insert with delete",
			args: []vm.Object{
				&vm.Array{Value: []vm.Object{
					&vm.Int{Value: 0},
					&vm.Int{Value: 1},
					&vm.Int{Value: 2}}},
				&vm.Int{Value: 1},
				&vm.Int{Value: 1},
				&vm.String{Value: "c"},
				&vm.String{Value: "d"}},
			deleted: &vm.Array{
				Value: []vm.Object{&vm.Int{Value: 1}}},
			Array: &vm.Array{Value: []vm.Object{
				&vm.Int{Value: 0},
				&vm.String{Value: "c"},
				&vm.String{Value: "d"},
				&vm.Int{Value: 2}}},
		},
		{name: "insert with delete multi",
			args: []vm.Object{
				&vm.Array{Value: []vm.Object{
					&vm.Int{Value: 0},
					&vm.Int{Value: 1},
					&vm.Int{Value: 2}}},
				&vm.Int{Value: 1},
				&vm.Int{Value: 2},
				&vm.String{Value: "c"},
				&vm.String{Value: "d"}},
			deleted: &vm.Array{Value: []vm.Object{
				&vm.Int{Value: 1},
				&vm.Int{Value: 2}}},
			Array: &vm.Array{
				Value: []vm.Object{
					&vm.Int{Value: 0},
					&vm.String{Value: "c"},
					&vm.String{Value: "d"}}},
		},
		{name: "delete all with positive count",
			args: []vm.Object{
				&vm.Array{Value: []vm.Object{
					&vm.Int{Value: 0},
					&vm.Int{Value: 1},
					&vm.Int{Value: 2}}},
				&vm.Int{Value: 0},
				&vm.Int{Value: 3}},
			deleted: &vm.Array{Value: []vm.Object{
				&vm.Int{Value: 0},
				&vm.Int{Value: 1},
				&vm.Int{Value: 2}}},
			Array: &vm.Array{Value: []vm.Object{}},
		},
		{name: "delete all with big count",
			args: []vm.Object{
				&vm.Array{Value: []vm.Object{
					&vm.Int{Value: 0},
					&vm.Int{Value: 1},
					&vm.Int{Value: 2}}},
				&vm.Int{Value: 0},
				&vm.Int{Value: 5}},
			deleted: &vm.Array{Value: []vm.Object{
				&vm.Int{Value: 0},
				&vm.Int{Value: 1},
				&vm.Int{Value: 2}}},
			Array: &vm.Array{Value: []vm.Object{}},
		},
		{name: "nothing2",
			args: []vm.Object{
				&vm.Array{Value: []vm.Object{
					&vm.Int{Value: 0},
					&vm.Int{Value: 1},
					&vm.Int{Value: 2}}}},
			Array: &vm.Array{Value: []vm.Object{}},
			deleted: &vm.Array{Value: []vm.Object{
				&vm.Int{Value: 0},
				&vm.Int{Value: 1},
				&vm.Int{Value: 2}}},
		},
		{name: "pop without count",
			args: []vm.Object{
				&vm.Array{Value: []vm.Object{
					&vm.Int{Value: 0},
					&vm.Int{Value: 1},
					&vm.Int{Value: 2}}},
				&vm.Int{Value: 2}},
			deleted: &vm.Array{Value: []vm.Object{&vm.Int{Value: 2}}},
			Array: &vm.Array{Value: []vm.Object{
				&vm.Int{Value: 0}, &vm.Int{Value: 1}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := builtinSplice(context.Background(), tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("builtinSplice() error = %v, wantErr %v",
					err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.deleted) {
				t.Errorf("builtinSplice() = %v, want %v", got, tt.deleted)
			}
			if tt.wantErr && tt.wantedErr.Error() != err.Error() {
				t.Errorf("builtinSplice() error = %v, wantedErr %v",
					err, tt.wantedErr)
			}
			if tt.Array != nil && !reflect.DeepEqual(tt.Array, tt.args[0]) {
				t.Errorf("builtinSplice() arrays are not equal expected"+
					" %s, got %s", tt.Array, tt.args[0].(*vm.Array))
			}
		})
	}
}

func Test_builtinRange(t *testing.T) {
	var builtinRange func(ctx context.Context, args ...vm.Object) (vm.Object, error)
	for _, f := range vm.GetAllBuiltinFunctions() {
		if f.Name == "range" {
			builtinRange = f.Value
			break
		}
	}
	if builtinRange == nil {
		t.Fatal("builtin range not found")
	}
	tests := []struct {
		name      string
		args      []vm.Object
		result    *vm.Array
		wantErr   bool
		wantedErr error
	}{
		{name: "no args", args: []vm.Object{}, wantErr: true,
			wantedErr: vm.ErrWrongNumArguments,
		},
		{name: "single args", args: []vm.Object{&vm.Map{}},
			wantErr:   true,
			wantedErr: vm.ErrWrongNumArguments,
		},
		{name: "4 args", args: []vm.Object{&vm.Map{}, &vm.String{}, &vm.String{}, &vm.String{}},
			wantErr:   true,
			wantedErr: vm.ErrWrongNumArguments,
		},
		{name: "invalid start",
			args:    []vm.Object{&vm.String{}, &vm.String{}},
			wantErr: true,
			wantedErr: vm.ErrInvalidArgumentType{
				Name: "start", Expected: "int", Found: "string"},
		},
		{name: "invalid stop",
			args:    []vm.Object{&vm.Int{}, &vm.String{}},
			wantErr: true,
			wantedErr: vm.ErrInvalidArgumentType{
				Name: "stop", Expected: "int", Found: "string"},
		},
		{name: "invalid step",
			args:    []vm.Object{&vm.Int{}, &vm.Int{}, &vm.String{}},
			wantErr: true,
			wantedErr: vm.ErrInvalidArgumentType{
				Name: "step", Expected: "int", Found: "string"},
		},
		{name: "zero step",
			args:      []vm.Object{&vm.Int{}, &vm.Int{}, &vm.Int{}}, //must greate than 0
			wantErr:   true,
			wantedErr: vm.ErrInvalidRangeStep,
		},
		{name: "negative step",
			args:      []vm.Object{&vm.Int{}, &vm.Int{}, intObject(-2)}, //must greate than 0
			wantErr:   true,
			wantedErr: vm.ErrInvalidRangeStep,
		},
		{name: "same bound",
			args:    []vm.Object{&vm.Int{}, &vm.Int{}},
			wantErr: false,
			result: &vm.Array{
				Value: nil,
			},
		},
		{name: "positive range",
			args:    []vm.Object{&vm.Int{}, &vm.Int{Value: 5}},
			wantErr: false,
			result: &vm.Array{
				Value: []vm.Object{
					intObject(0),
					intObject(1),
					intObject(2),
					intObject(3),
					intObject(4),
				},
			},
		},
		{name: "negative range",
			args:    []vm.Object{&vm.Int{}, &vm.Int{Value: -5}},
			wantErr: false,
			result: &vm.Array{
				Value: []vm.Object{
					intObject(0),
					intObject(-1),
					intObject(-2),
					intObject(-3),
					intObject(-4),
				},
			},
		},

		{name: "positive with step",
			args:    []vm.Object{&vm.Int{}, &vm.Int{Value: 5}, &vm.Int{Value: 2}},
			wantErr: false,
			result: &vm.Array{
				Value: []vm.Object{
					intObject(0),
					intObject(2),
					intObject(4),
				},
			},
		},

		{name: "negative with step",
			args:    []vm.Object{&vm.Int{}, &vm.Int{Value: -10}, &vm.Int{Value: 2}},
			wantErr: false,
			result: &vm.Array{
				Value: []vm.Object{
					intObject(0),
					intObject(-2),
					intObject(-4),
					intObject(-6),
					intObject(-8),
				},
			},
		},

		{name: "large range",
			args:    []vm.Object{intObject(-10), intObject(10), &vm.Int{Value: 3}},
			wantErr: false,
			result: &vm.Array{
				Value: []vm.Object{
					intObject(-10),
					intObject(-7),
					intObject(-4),
					intObject(-1),
					intObject(2),
					intObject(5),
					intObject(8),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := builtinRange(context.Background(), tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("builtinRange() error = %v, wantErr %v",
					err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.wantedErr.Error() != err.Error() {
				t.Errorf("builtinRange() error = %v, wantedErr %v",
					err, tt.wantedErr)
			}
			if tt.result != nil && !reflect.DeepEqual(tt.result, got) {
				t.Errorf("builtinRange() arrays are not equal expected"+
					" %s, got %s", tt.result, got.(*vm.Array))
			}
		})
	}
}
