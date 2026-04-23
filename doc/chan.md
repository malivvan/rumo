---
title: chan
---

## chan

Makes a channel to send/receive object and returns a chan object that has
send, recv, close methods.

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

On the time the VM that the chan is running in is cancelled, the sending
or receiving call returns immediately.
