package module

import (
	"testing"
)

func TestParseExports(t *testing.T) {
	infos, err := ParseExports(`
export {
  /* get(key string) (ret string) get comment
     get newline comment */
  get: func(key) {},

  // set(key string, val string) set comment
  // set newline comment
  set: func(key, val) {},
}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) != 2 {
		t.Fatal("len(infos) != 2")
	}
	if infos[0].Name != "get" {
		t.Fatal("infos[0].Name != \"get\"")
	}
	if len(infos[0].Inputs) != 1 {
		t.Fatal("len(infos[0].Inputs) != 1")
	}
	if infos[0].Inputs[0][0] != "key" {
		t.Fatal("infos[0].Inputs[0][0] != \"key\"")
	}
	if infos[0].Inputs[0][1] != "string" {
		t.Fatal("infos[0].Inputs[0][1] != \"string\"")
	}
	if len(infos[0].Output) != 2 {
		t.Fatal("len(infos[0].Output) != 2")
	}
	if infos[0].Output[0] != "ret" {
		t.Fatal("infos[0].Output[0] != \"ret\"")
	}
	if infos[0].Output[1] != "string" {
		t.Fatal("infos[0].Output[1] != \"string\"")
	}
	if infos[0].Comment != "get comment\nget newline comment" {
		t.Fatal("infos[0].Comment != \"get comment\\nget newline comment\"")
	}
	if infos[1].Name != "set" {
		t.Fatal("infos[1].Name != \"set\"")
	}
	if len(infos[1].Inputs) != 2 {
		t.Fatal("len(infos[1].Inputs) != 2")
	}
	if infos[1].Inputs[0][0] != "key" {
		t.Fatal("infos[1].Inputs[0][0] != \"key\"")
	}
	if infos[1].Inputs[0][1] != "string" {
		t.Fatal("infos[1].Inputs[0][1] != \"string\"")
	}
	if infos[1].Inputs[1][0] != "val" {
		t.Fatal("infos[1].Inputs[1][0] != \"val\"")
	}
	if infos[1].Inputs[1][1] != "string" {
		t.Fatal("infos[1].Inputs[1][1] != \"string\"")
	}
	if len(infos[1].Output) != 0 {
		t.Fatal("len(infos[1].Output) != 0")
	}
	if infos[1].Comment != "set comment\nset newline comment" {
		t.Fatal("infos[1].Comment != \"set comment\\nset newline comment\"")
	}
}

func TestParseExport(t *testing.T) {
	var cases = []struct {
		s    string
		info *Export
	}{
		{s: "someInt int", info: &Export{Name: "someInt"}},
		{s: "someString string", info: &Export{Name: "someString"}},
		{s: "someBool bool", info: &Export{Name: "someBool"}},
		{s: "someFloat float", info: &Export{Name: "someFloat"}},
		{s: "someBytes bytes", info: &Export{Name: "someBytes"}},
		{s: "someChar char", info: &Export{Name: "someChar"}},
		{s: "someTime time", info: &Export{Name: "someTime"}},
		{s: "run()", info: &Export{Name: "run"}},
		{s: "run() (Ret String)", info: &Export{Name: "run", Output: []string{"Ret", "string"}}},
		{s: "run(arg1 int, arg2 string) (ret string)", info: &Export{Name: "run", Inputs: [][]string{{"arg1", "int"}, {"arg2", "string"}}, Output: []string{"ret", "string"}}},
	}
	for _, c := range cases {
		t.Run(c.s, func(t *testing.T) {
			info := ParseExport(c.s)
			if c.info == nil && info != nil {
				t.Fatalf("expected nil, got %v", info)
			}
			if c.info != nil && info == nil {
				t.Fatalf("expected info, got %v", info)
			}
			if c.info.Name != info.Name {
				t.Fatalf("expected name '%s', got '%s'", c.info.Name, info.Name)
			}
			if len(c.info.Inputs) != len(info.Inputs) {
				t.Fatalf("expected %d inputs, got %d", len(c.info.Inputs), len(info.Inputs))
			}
			for i := range c.info.Inputs {
				if len(c.info.Inputs[i]) != len(info.Inputs[i]) {
					t.Fatalf("expected %d inputs for parameter %d, got %d", len(c.info.Inputs[i]), i, len(info.Inputs[i]))
				}
				for ii, expected := range c.info.Inputs[i] {
					if expected != info.Inputs[i][ii] {
						t.Fatalf("expected input %d to be '%s', got '%s'", i, expected, info.Inputs[i])
					}
				}
			}
			if len(c.info.Output) != len(info.Output) {
				t.Fatalf("expected %d outputs, got %d", len(c.info.Output), len(info.Output))
			}
			for i, expected := range c.info.Output {
				if expected != info.Output[i] {
					t.Fatalf("expected output %d to be '%s', got '%s'", i, expected, info.Output[i])
				}
			}
			if c.info.Comment != info.Comment {
				t.Fatalf("expected comment '%s', got '%s'", c.info.Comment, info.Comment)
			}
		})
	}
}
