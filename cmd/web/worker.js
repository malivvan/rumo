// Worker bootstrap shared by TWO roles, both of which load the same
// rumo.wasm:
//
//   1. SharedWorker  "rumo-coordinator" — the singleton coordinator
//                                          (shared FS, monitor, routine
//                                          + chan registry across tabs).
//   2. DedicatedWorker (any name)       — a per-VM vm-host. Used both for
//                                          top-level rumo.run/runCompiled/
//                                          spawn calls AND for per-`go fn()`
//                                          children. DedicatedWorker is
//                                          required because SharedWorker
//                                          contexts cannot construct
//                                          further SharedWorkers
//                                          (https://crbug.com/1102827) and
//                                          we need recursive `go fn()`
//                                          fan-out.
//
// The Go side (cmd/main_js.go) detects the scope at startup
// (SharedWorkerGlobalScope vs DedicatedWorkerGlobalScope) and installs
// either an `onconnect` handler (SharedWorker) or an `onmessage` handler
// (DedicatedWorker). The bootstrap below queues whichever events arrive
// before wasm is ready and replays them once the Go-side handler is
// installed.

importScripts("./wasm_exec.js");

const isShared =
    typeof SharedWorkerGlobalScope !== "undefined" &&
    self instanceof SharedWorkerGlobalScope;

const go = new Go();
const wasmReady = WebAssembly.instantiateStreaming(fetch("./rumo.wasm"), go.importObject)
    .then((res) => {
        // go.run never resolves under normal use — main() calls select{}.
        // We deliberately don't await it; the Go side wires its handler
        // synchronously during startup before yielding to the JS event loop.
        go.run(res.instance);
        return true;
    })
    .catch((err) => {
        console.error("rumo: failed to start wasm", err);
        throw err;
    });

// Queue events that arrive before the Go-side handler is installed so we
// don't drop them. We snapshot the bootstrap handler reference so we can
// tell after wasmReady resolves whether the Go side has replaced it.
const pending = [];
let bootstrapHandler;
if (isShared) {
    bootstrapHandler = (e) => { pending.push(e); };
    self.onconnect = bootstrapHandler;
} else {
    bootstrapHandler = (e) => { pending.push(e); };
    self.onmessage = bootstrapHandler;
}

wasmReady.then(() => {
    if (isShared) {
        if (typeof self.onconnect !== "function" || self.onconnect === bootstrapHandler) {
            return;
        }
        for (const e of pending) {
            try { self.onconnect(e); }
            catch (err) { console.error("rumo: pending connect dispatch failed", err); }
        }
    } else {
        if (typeof self.onmessage !== "function" || self.onmessage === bootstrapHandler) {
            return;
        }
        for (const e of pending) {
            try { self.onmessage(e); }
            catch (err) { console.error("rumo: pending message dispatch failed", err); }
        }
    }
    pending.length = 0;
});


