<picture>
 <source media="(prefers-color-scheme: light)" srcset="doc/img/rumo_light.png">
  <source media="(prefers-color-scheme: dark)" srcset="doc/img/rumo.png">
 <img alt="rumo logo" src="doc/img/rumo.png" width="500">
</picture>

# ![test](https://github.com/malivvan/rumo/workflows/test/badge.svg) [![Release](https://img.shields.io/github/v/release/malivvan/vv.svg?sort=semver)](https://github.com/malivvan/rumo/releases/latest) [![Go Report Card](https://goreportcard.com/badge/github.com/malivvan/rumo)](https://goreportcard.com/report/github.com/malivvan/rumo) [![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
rumo is a small, fast and secure script language for Go supporting routines and channels

> This is pre release software so expect bugs and breaking changes

## Usage

### package usage
```golang
package main

import (
	"fmt"
	"github.com/malivvan/rumo"
)

func main() {
	// script code
	src := `
each := func(seq, fn) {
    for x in seq { fn(x) }
}

sum := 0
mul := 1
each([a, b, c, d], func(x) {
	sum += x
	mul *= x
})`

	// create a new script instance
	script := rumo.NewScript([]byte(src))

	// add variables with default values
	_ = script.Add("a", 0)
	_ = script.Add("b", 0)
	_ = script.Add("c", 0)
	_ = script.Add("d", 0)

	// compile script to program
	program, err := script.Compile()
	if err != nil {
		panic(err)
	}
	
	// clone a new instance of the program and set values
	instance := program.Clone()
	_ = instance.Set("a", 1)
	_ = instance.Set("b", 9)
	_ = instance.Set("c", 8)
	_ = instance.Set("d", 4)
	
	// run the instance
	err = instance.Run()
	if err != nil {
		panic(err)
	}

	// retrieve variable values
	sum := instance.Get("sum")
	mul := instance.Get("mul")
	fmt.Println(sum, mul) // "22 288"
}
```

### language usage

```golang
fmt := import("fmt")

each := func(seq, fn) {
    for x in seq { fn(x) }
}

sum := func(init, seq) {
each(seq, func(x) { init += x })
    return init
}

fmt.println(sum(0, [1, 2, 3]))   // "6"
fmt.println(sum("", [1, 2, 3]))  // "123"
```

#### Routines
```golang
v := 0

f1 := func(a,b) { v = 10; return a+b }
f2 := func(a,b,c) { v = 11; return a+b+c }

rvm1 := start(f1,1,2)
rvm2 := start(f2,1,2,5)

fmt.println(rvm1.result()) // 3
fmt.println(rvm2.result()) // 8
fmt.println(v) // 10 or 11
```

#### Channels
```golang
unbufferedChan := chan()
bufferedChan := chan(128)

// Send will block if the channel is full.
bufferedChan.send("hello") // send string
bufferedChan.send(55) // send int
bufferedChan.send([66, chan(1)]) // channel in channel

// Receive will block if the channel is empty.
obj := bufferedChan.recv()

// Send to a closed channel causes panic.
// Receive from a closed channel returns undefined value.
unbufferedChan.close()
bufferedChan.close()
```

#### Routines and Channels
```golang
reqChan := chan(8)
repChan := chan(8)

client := func(interval) {
	reqChan.send("hello")
	for i := 0; true; i++ {
		fmt.println(repChan.recv())
		times.sleep(interval*times.second)
		reqChan.send(i)
	}
}

server := func() {
	for {
		req := reqChan.recv()
		if req == "hello" {
			fmt.println(req)
			repChan.send("world")
		} else {
			repChan.send(req+100)
		}
	}
}

rClient := start(client, 2)
rServer := start(server)

if ok := rClient.wait(5); !ok {
	rClient.abort()
}
rServer.abort()

//output:
//hello
//world
//100
//101
```
## Building

```bash
make test       # run tests
make install    # install tools 
make build      # build for current platform 
make release    # build for all platforms
```

## Fork / Credits
This is a continuation of the [github.com/d5/tengo](https://github.com/d5/tengo) project starting of this [pull request](https://github.com/d5/tengo/pull/330) implementing go routines and channels. Special thanks goes to [d5](https://github.com/d5/) for his work on the tengo language and [Bai-Yingjie](https://github.com/Bai-Yingjie) for implementing the foundation of concurrency while retaining the original tests of the project. 