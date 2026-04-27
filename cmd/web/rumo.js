// rumo.js — page-side wrapper. Each top-level VM runs in its own dedicated
// Worker (a DedicatedWorker named "rumo-vm-<id>") so that every active
// rumo VM appears as a separate worker in DevTools and as a distinct row
// in the live monitor. A "rumo-coordinator" SharedWorker holds the
// in-memory filesystem, the routine registry, and the lifecycle bus —
// making them shared across every page tab on the same origin.
//
// Why DedicatedWorker for the vm-host (and not SharedWorker)?
//
//   The vm-host needs to talk to the coordinator (FS, routine registry,
//   chan queues, monitor). To do so it constructs the
//   `new SharedWorker(workerURL, "rumo-coordinator")`. Per the HTML/Workers
//   spec — and as enforced by Chromium and Firefox — a SharedWorker scope
//   does NOT expose the SharedWorker constructor (https://crbug.com/1102827).
//   Running the vm-host as a DedicatedWorker side-steps this restriction:
//   DedicatedWorker scopes CAN construct SharedWorkers and DedicatedWorkers.
//   This is also why each `go fn()` child uses a DedicatedWorker.
//
// Architecture
//
//   page.js
//     │
//     ├──► rumo-coordinator     (1 SharedWorker, owns FS + monitor registry)
//     │
//     ├──► rumo-vm-A            (1 DedicatedWorker per VM run; streams output)
//     │     └──► rumo-vm-A-go-1 (child DedicatedWorker per `go fn()`)
//     ├──► rumo-vm-B
//     └──► rumo-vm-C
//
// Lifecycle for every VM created via rumo.run / runCompiled / spawn:
//   1. ask coordinator for a fresh routine id (`routine.register`)
//   2. fetch the FS snapshot (`fs.snapshot`)
//   3. create `new Worker(workerURL, { name: "rumo-vm-<id>" })`
//   4. forward output chunks to onOutput AND to the coordinator
//      (`routine.update`) so monitor byte counts stay live
//   5. on done → `routine.done` → coordinator promotes the row to the
//      "done"/"error" state so it's visible to every other tab too

(function (root) {
    "use strict";

    const COORD_NAME = "rumo-coordinator";

    // -----------------------------------------------------------------------
    // PortClient — request/response over a MessagePort, SharedWorker.port,
    // or DedicatedWorker. Worker objects don't have .start()/.close(); we
    // detect that and adapt accordingly so the same client works for every
    // role (coordinator SharedWorker, vm-host DedicatedWorker, ...).
    // -----------------------------------------------------------------------
    class PortClient {
        constructor(port) {
            this.port = port;
            this._next = 1;
            this._pending = new Map();
            this._listeners = [];
            this._readyResolve = null;
            this._readyReject = null;
            this.ready = new Promise((res, rej) => {
                this._readyResolve = res;
                this._readyReject = rej;
            });
            port.onmessage = (e) => this._dispatch(e.data);
            port.onmessageerror = (e) => console.error("rumo: messageerror", e);
            // SharedWorker.port / MessagePort require .start(); plain Worker
            // objects auto-start and don't expose it.
            if (typeof port.start === "function") port.start();
        }
        _dispatch(msg) {
            if (!msg) return;
            if (msg.type === "ready") {
                if (this._readyResolve) { this._readyResolve(msg); this._readyResolve = null; this._readyReject = null; }
                return;
            }
            if (msg.id !== undefined && this._pending.has(msg.id)) {
                const p = this._pending.get(msg.id);
                this._pending.delete(msg.id);
                if (msg.error !== undefined && msg.error !== null) {
                    p.reject(new Error(msg.error));
                } else {
                    p.resolve(msg.result);
                }
                return;
            }
            for (const l of this._listeners) l(msg);
        }
        listen(fn) { this._listeners.push(fn); return () => {
            const i = this._listeners.indexOf(fn);
            if (i !== -1) this._listeners.splice(i, 1);
        }; }
        call(op, payload) {
            return new Promise((resolve, reject) => {
                const id = this._next++;
                this._pending.set(id, { resolve, reject });
                this.port.postMessage(Object.assign({ id, op }, payload || {}));
            });
        }
        // callTransfer is identical to call() but additionally hands ownership
        // of the listed transferables (typically MessagePort instances) to the
        // peer. The transferables, if also referenced inside `payload`, will
        // appear at the same position on the receiving side.
        callTransfer(op, payload, transfer) {
            return new Promise((resolve, reject) => {
                const id = this._next++;
                this._pending.set(id, { resolve, reject });
                this.port.postMessage(Object.assign({ id, op }, payload || {}), transfer || []);
            });
        }
        post(msg) { this.port.postMessage(msg); }
        postTransfer(msg, transfer) { this.port.postMessage(msg, transfer || []); }
        close() {
            // SharedWorker.port / MessagePort have .close(); a plain Worker
            // is terminated via .terminate(). We don't own SharedWorker
            // ports beyond the page, so close() is a no-op there.
            if (typeof this.port.close === "function") {
                try { this.port.close(); } catch (_) {}
            }
        }
    }

    function encode(v) {
        if (v === null || v === undefined) return new Uint8Array(0);
        if (v instanceof Uint8Array) return v;
        if (v instanceof ArrayBuffer) return new Uint8Array(v);
        if (typeof v === "string") return new TextEncoder().encode(v);
        throw new TypeError("rumo: expected string, Uint8Array, or ArrayBuffer");
    }

    // -----------------------------------------------------------------------
    // Rumo client
    // -----------------------------------------------------------------------
    class Rumo {
        constructor(workerURL) {
            this._workerURL = workerURL || "./worker.js";
            const coord = new SharedWorker(this._workerURL, COORD_NAME);
            this._coordSW = coord;
            this._coord = new PortClient(coord.port);
            this.ready = this._coord.ready;

            this._monitorSubs = [];
            this._coord.listen((msg) => {
                if (msg.type === "routine:spawned" || msg.type === "routine:done") {
                    for (const cb of this._monitorSubs) {
                        try { cb(msg); } catch (err) { console.error(err); }
                    }
                }
            });
        }

        // ── meta ────────────────────────────────────────────────────────────
        version() { return this._coord.call("version"); }
        commit()  { return this._coord.call("commit"); }
        modules() { return this._coord.call("modules"); }
        exports() { return this._coord.call("exports"); }

        compile(source, path) {
            return this._coord.call("compile", { source: encode(source), path: path || "" });
        }

        // ── per-VM execution (each in its own DedicatedWorker) ─────────────
        async _runInWorker(kind, body, opts) {
            opts = opts || {};
            // 1) register routine in coordinator (assigns id)
            const workerName = "rumo-vm-" + Math.random().toString(36).slice(2, 10);
            const routineId = await this._coord.call("routine.register", {
                kind,
                name: opts.path || opts.name || (kind + ".rumo"),
                workerName,
            });
            // 2) snapshot shared FS so the per-VM worker can resolve imports
            const fsSnap = await this._coord.call("fs.snapshot");
            // 3) Establish a private MessagePort between the new vm-host
            //    and the coordinator. Workers cannot reliably construct
            //    SharedWorkers from inside their own scope (Chromium
            //    restricts SharedWorker creation to Window in many
            //    versions, https://crbug.com/1102827), so we use a
            //    transferable MessageChannel instead — fully portable.
            const coordCh = new MessageChannel();
            this._coord.postTransfer(
                { op: "coord.attach", port: coordCh.port1 },
                [coordCh.port1],
            );
            // 4) spin up the vm-host as a DedicatedWorker. Plain Workers
            //    are creatable from any context (Window or Worker), can
            //    construct further DedicatedWorkers for `go fn()` children,
            //    and accept transferred MessagePorts the same as any other.
            const sw = new Worker(this._workerURL, { name: workerName });
            const vm = new PortClient(sw);
            try {
                await vm.ready;
            } catch (err) {
                await this._coord.call("routine.done", {
                    routineId, error: "vm-host failed to start: " + err.message,
                });
                try { sw.terminate(); } catch (_) {}
                throw err;
            }

            // forward output chunks to onOutput and to coordinator (byte counter)
            let bytes = 0;
            const removeListener = vm.listen((msg) => {
                if (msg.type === "output") {
                    bytes += (msg.bytes !== undefined ? msg.bytes : 0);
                    if (opts.onOutput) {
                        try { opts.onOutput(msg.chunk); } catch (e) { console.error(e); }
                    }
                    // best-effort byte-count update; intentionally not awaited
                    this._coord.call("routine.update", { routineId, bytes }).catch(() => {});
                }
            });

            const op = body.bytecode ? "runVMCompiled" : "runVM";
            const payload = Object.assign({}, body, {
                args: opts.args || [],
                stdin: opts.stdin || "",
                fs: fsSnap,
                path: opts.path || "",
                workerURL: this._workerURL,
                routineId,
                // coordPort is the partner of the port we just handed to
                // the coordinator. The vm-host uses it for routine.allocate
                // / chan.* / etc. without needing to construct a
                // SharedWorker itself.
                coordPort: coordCh.port2,
            });

            const handle = {
                id: routineId,
                workerName,
                _vm: vm,
                _sw: sw,
                _bytes: () => bytes,
            };

            const cleanup = () => {
                removeListener();
                // DedicatedWorker is terminated; the coordinator
                // SharedWorker stays alive across runs.
                try { sw.terminate(); } catch (_) {}
            };

            const donePromise = vm.callTransfer(op, payload, [coordCh.port2]).then((result) => {
                this._coord.call("routine.done", {
                    routineId,
                    error: result.error,
                }).catch(() => {});
                cleanup();
                return result;
            }).catch(async (err) => {
                await this._coord.call("routine.done", { routineId, error: err.message });
                cleanup();
                throw err;
            });

            handle.done = donePromise;
            handle.cancel = () => { try { vm.post({ op: "cancel" }); } catch (_) {} };
            return handle;
        }

        async run(source, opts) {
            const handle = await this._runInWorker("run", { source: encode(source) }, opts);
            return handle.done;
        }

        async runCompiled(bytes, opts) {
            const handle = await this._runInWorker("runCompiled", { bytecode: bytes }, opts);
            return handle.done;
        }

        async spawn(source, opts) {
            const h = await this._runInWorker("spawn", { source: encode(source) }, opts);
            return {
                id: h.id,
                workerName: h.workerName,
                cancel: () => h.cancel(),
                done: h.done,
                result: () => h.done,
                wait: (seconds) => Promise.race([
                    h.done.then(() => true),
                    new Promise((r) => setTimeout(() => r(false),
                        (seconds == null || seconds < 0) ? 365*24*3600*1000 : seconds*1000)),
                ]),
                bytes: () => h._bytes(),
                onOutput: (cb) => { /* callback set via spawn opts.onOutput */
                    console.warn("rumo: pass onOutput in spawn(opts), not after");
                    if (typeof cb === "function") opts.onOutput = cb;
                },
            };
        }

        // ── monitor / registry (coordinator-backed, shared across tabs) ────
        routines()      { return this._coord.call("routines.list"); }
        pruneRoutines() { return this._coord.call("routines.prune"); }
        subscribe(cb) {
            this._monitorSubs.push(cb);
            // ask coordinator to push lifecycle messages on this port
            if (!this._monitorActive) {
                this._monitorActive = true;
                this._coord.call("monitor.subscribe").catch(() => { this._monitorActive = false; });
            }
            return () => {
                const i = this._monitorSubs.indexOf(cb);
                if (i !== -1) this._monitorSubs.splice(i, 1);
            };
        }
        monitor(render, intervalMs) {
            const period = typeof intervalMs === "number" ? intervalMs : 500;
            let stopped = false;
            const tick = async () => {
                if (stopped) return;
                try { render(await this.routines()); } catch (_) {}
                if (!stopped) setTimeout(tick, period);
            };
            tick();
            return () => { stopped = true; };
        }

        // ── shared filesystem ──────────────────────────────────────────────
        get fs() {
            const c = this._coord;
            return {
                put:    (path, content) => c.call("fs.put", { path, content: encode(content) }),
                get:    (path)          => c.call("fs.get", { path }),
                list:   ()              => c.call("fs.list"),
                delete: (path)          => c.call("fs.delete", { path }),
                clear:  ()              => c.call("fs.clear"),
            };
        }
    }

    async function connect(workerURL) {
        const r = new Rumo(workerURL);
        await r.ready;
        return r;
    }

    root.Rumo = { connect };
})(typeof window !== "undefined" ? window : self);
