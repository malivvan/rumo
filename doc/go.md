---
title: go
---
## go

The `go` keyword launches an independent concurrent routine from a function call expression.

If the function is a CompiledFunction, the current running VM will be cloned to create
a new VM in which the CompiledFunction will be running.
The function can also be any object that has Call() method, such as BuiltinFunction,
in which case no cloned VM will be created.
Returns a routine handle map with `wait`, `result`, and `cancel` methods.

The routine will not exit unless:
1. All its descendant routines exit
2. It calls `cancel()`
3. Its handle's `cancel()` method is called on behalf of its parent VM

The latter 2 cases will trigger the cancellation of all descendant routines,
which will further result in #1 above.

```golang
v := 0

f1 := func(a,b) { v = 10; return a+b }
f2 := func(a,b,c) { v = 11; return a+b+c }

rvm1 := go f1(1,2)
rvm2 := go f2(1,2,5)

fmt.println(rvm1.result()) // 3
fmt.println(rvm2.result()) // 8
fmt.println(v) // 10 or 11
```

* `wait()` waits for the routine to complete up to timeout seconds and
  returns true if the routine exited (successfully or not) within the
  timeout. It waits forever if the optional timeout is not specified,
  or timeout < 0.
* `cancel()` triggers the termination process of the routine and all
  its descendant VMs.
* `result()` waits for the routine to complete, returns Error object if
  any runtime error occurred during the execution, otherwise returns the
  result value of the function call.

### 1 client 1 server

Below is a simple client server example:

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

rClient := go client(2)
rServer := go server()

if ok := rClient.wait(5); !ok {
	rClient.cancel()
}
rServer.cancel()

//output:
//hello
//world
//100
//101
```

### n client n server, channel in channel

```golang
sharedReqChan := chan(128)

client = func(name, interval, timeout) {
	print := func(s) {
		fmt.println(name, s)
	}
	print("started")

	repChan := chan(1)
	msg := {chan:repChan}

	msg.data = "hello"
	sharedReqChan.send(msg)
	print(repChan.recv())

	for i := 0; i * interval < timeout; i++ {
		msg.data = i
		sharedReqChan.send(msg)
		print(repChan.recv())
		times.sleep(interval*times.second)
	}
}

server = func(name) {
	print := func(s) {
		fmt.println(name, s)
	}
	print("started")

	for {
		req := sharedReqChan.recv()
		if req.data == "hello" {
			req.chan.send("world")
		} else {
			req.chan.send(req.data+100)
		}
	}
}

clients := func() {
	for i :=0; i < 5; i++ {
		go client(format("client %d: ", i), 1, 4)
	}
}

servers := func() {
	for i :=0; i < 2; i++ {
		go server(format("server %d: ", i))
	}
}

// After 4 seconds, all clients should have exited normally
rclts := go clients()
// If servers exit earlier than clients, then clients may be
// blocked forever waiting for the reply chan, because servers
// were cancelled with the req fetched from sharedReqChan before
// sending back the reply.
// In such case, do below to cancel() the clients manually
//go func(){times.sleep(6*times.second); gclts.cancel()}()

// Servers are infinite loop, cancel() them after 5 seconds
rsrvs := go servers()
if ok := rsrvs.wait(5); !ok {
	rsrvs.cancel()
}

// Main VM waits here until all the child routines finish

// If somehow the main VM is stuck, that is because there is
// at least one child VM that has not exited as expected, we
// can do cancel() to force exit.
cancel()

//output:
//3
//8
//hello
//world
//100
//101

//unordered output:
//client 4: started
//server 0: started
//client 4: world
//client 4: 100
//client 3: started
//client 3: world
//client 3: 100
//client 2: started
//client 2: world
//client 2: 100
//client 0: started
//client 0: world
//client 0: 100
//client 1: started
//client 1: world
//client 1: 100
//server 1: started
//client 1: 101
//client 2: 101
//client 4: 101
//client 0: 101
//client 3: 101
//client 3: 102
//client 0: 102
//client 2: 102
//client 1: 102
//client 4: 102
//client 0: 103
//client 3: 103
//client 2: 103
//client 1: 103
//client 4: 103

```

## cancel
Triggers the termination process of the current VM and all its descendant VMs.
