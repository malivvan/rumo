// SharedWorker bootstrap. Loads the Go WASM runtime once and serves every
// page tab that connects through the standard SharedWorker MessagePort
// protocol. The runtime itself (in main_js.go) installs the global `rumo`
// API and registers an `onconnect` handler that the Go side controls.

importScripts("./wasm_exec.js");

const go = new Go();
const wasmReady = WebAssembly.instantiateStreaming(fetch("./rumo.wasm"), go.importObject)
    .then((res) => {
        // go.run never resolves under normal use — main() calls select{}.
        // We deliberately don't await it; the Go side will wire onconnect
        // synchronously during startup.
        go.run(res.instance);
        return true;
    })
    .catch((err) => {
        console.error("rumo: failed to start wasm", err);
        throw err;
    });

// Until the wasm has wired its own onconnect handler, queue connections so
// no port is dropped. The Go side is expected to overwrite self.onconnect
// shortly after instantiation; we forward any pre-startup connections to it
// after the promise resolves.
const pending = [];
self.onconnect = (e) => {
    pending.push(e);
};

wasmReady.then(() => {
    // The Go installer should have replaced self.onconnect by now.
    if (typeof self.onconnect !== "function" || pending.length === 0) {
        return;
    }
    for (const e of pending) {
        try {
            self.onconnect(e);
        } catch (err) {
            console.error("rumo: pending connect dispatch failed", err);
        }
    }
    pending.length = 0;
});
