// rumo.js — page-side wrapper around the SharedWorker that hosts the rumo
// runtime. It mirrors the github.com/malivvan/rumo Go package surface as a
// Promise-based JavaScript API. All work is dispatched to the SharedWorker
// over its MessagePort; multiple page contexts (tabs, iframes) connect to the
// same worker and therefore share the in-memory filesystem and runtime
// registries.

(function (root) {
    "use strict";

    function Rumo(workerURL) {
        const worker = new SharedWorker(workerURL || "./worker.js", { name: "rumo" });
        this._worker = worker;
        this._port = worker.port;

        this._nextId = 1;
        this._pending = new Map();   // id -> {resolve, reject}
        this._streams = new Map();   // streamId -> onChunk
        this._routines = new Map();  // routineId -> {onChunk, donePending: Promise}
        this._monitorSubs = [];      // user-provided lifecycle callbacks
        this._monitorActive = false; // remote subscription requested

        this._readyResolve = null;
        this.ready = new Promise((resolve) => { this._readyResolve = resolve; });

        this._port.onmessage = (e) => this._onMessage(e.data);
        this._port.onmessageerror = (e) => console.error("rumo: messageerror", e);
        this._port.start();
    }

    Rumo.prototype._onMessage = function (msg) {
        if (!msg) return;
        if (msg.type === "ready") {
            if (this._readyResolve) {
                this._readyResolve(msg);
                this._readyResolve = null;
            }
            return;
        }
        if (msg.type === "output") {
            if (msg.streamId !== undefined) {
                const cb = this._streams.get(msg.streamId);
                if (cb) cb(msg.chunk);
                return;
            }
            if (msg.routineId !== undefined) {
                const r = this._routines.get(msg.routineId);
                if (r && r.onChunk) r.onChunk(msg.chunk);
                return;
            }
            return;
        }
        if (msg.type === "done") {
            const r = this._routines.get(msg.routineId);
            if (r && r._resolveDone) {
                r._resolveDone({ error: msg.error || null });
            }
            return;
        }
        if (msg.type === "routine:spawned" || msg.type === "routine:done") {
            for (const cb of this._monitorSubs) {
                try { cb(msg); } catch (err) { console.error("rumo: monitor cb threw", err); }
            }
            return;
        }
        if (msg.id !== undefined) {
            const p = this._pending.get(msg.id);
            if (!p) return;
            this._pending.delete(msg.id);
            if (msg.error !== undefined && msg.error !== null) {
                p.reject(new Error(msg.error));
            } else {
                p.resolve(msg.result);
            }
        }
    };

    Rumo.prototype._call = function (op, payload) {
        return new Promise((resolve, reject) => {
            const id = this._nextId++;
            this._pending.set(id, { resolve, reject });
            const msg = Object.assign({ id, op }, payload || {});
            this._port.postMessage(msg);
        });
    };

    Rumo.prototype.version = function () { return this._call("version"); };
    Rumo.prototype.commit = function () { return this._call("commit"); };
    Rumo.prototype.modules = function () { return this._call("modules"); };
    Rumo.prototype.exports = function () { return this._call("exports"); };

    Rumo.prototype.compile = function (source, path) {
        return this._call("compile", { source: encode(source), path: path || "" });
    };

    Rumo.prototype.run = function (source, opts) {
        opts = opts || {};
        const streamId = opts.onOutput ? this._nextId++ : undefined;
        if (streamId !== undefined) this._streams.set(streamId, opts.onOutput);
        const payload = {
            source: encode(source),
            path: opts.path || "",
            args: opts.args || [],
            stdin: opts.stdin || "",
            streamId,
        };
        const promise = this._call("run", payload);
        if (streamId !== undefined) {
            promise.finally(() => this._streams.delete(streamId));
        }
        return promise;
    };

    Rumo.prototype.runCompiled = function (bytes, opts) {
        opts = opts || {};
        const streamId = opts.onOutput ? this._nextId++ : undefined;
        if (streamId !== undefined) this._streams.set(streamId, opts.onOutput);
        const payload = {
            bytecode: bytes,
            args: opts.args || [],
            stdin: opts.stdin || "",
            streamId,
        };
        const promise = this._call("runCompiled", payload);
        if (streamId !== undefined) {
            promise.finally(() => this._streams.delete(streamId));
        }
        return promise;
    };

    Rumo.prototype.spawn = async function (source, opts) {
        opts = opts || {};
        const id = await this._call("spawn", {
            source: encode(source),
            path: opts.path || "",
            args: opts.args || [],
            stdin: opts.stdin || "",
        });
        const self = this;
        const entry = { onChunk: opts.onOutput || null };
        let resolveDone;
        entry.donePromise = new Promise((res) => { resolveDone = res; });
        entry._resolveDone = resolveDone;
        self._routines.set(id, entry);

        return {
            id,
            cancel: () => self._call("routine.cancel", { routineId: id }),
            wait: (seconds) => self._call("routine.wait", {
                routineId: id,
                seconds: typeof seconds === "number" ? seconds : -1,
            }),
            result: () => self._call("routine.result", { routineId: id }),
            write: (data) => self._call("routine.write", { routineId: id, chunk: encode(data) }),
            close: () => {
                self._routines.delete(id);
                return self._call("routine.close", { routineId: id });
            },
            done: entry.donePromise,
            onOutput: (cb) => { entry.onChunk = cb; },
        };
    };

    Rumo.prototype.routines = function () { return this._call("routines.list"); };
    Rumo.prototype.pruneRoutines = function () { return this._call("routines.prune"); };

    /**
     * Subscribe to routine lifecycle events. The callback fires for every
     * spawned and finished routine (run, runCompiled, spawn) hosted by the
     * SharedWorker — across all page tabs that share it.
     * @param {(ev: {type: string, routine: object}) => void} cb
     * @returns {() => void} unsubscribe handle
     */
    Rumo.prototype.subscribe = function (cb) {
        this._monitorSubs.push(cb);
        if (!this._monitorActive) {
            this._monitorActive = true;
            this._call("monitor.subscribe").catch(() => { this._monitorActive = false; });
        }
        return () => {
            const idx = this._monitorSubs.indexOf(cb);
            if (idx !== -1) this._monitorSubs.splice(idx, 1);
        };
    };

    /**
     * Convenience helper: poll rumo.routines() at a fixed interval and invoke
     * `render` with the latest snapshot. Returns a stop function.
     * @param {(list: object[]) => void} render
     * @param {number} [intervalMs=500]
     */
    Rumo.prototype.monitor = function (render, intervalMs) {
        const period = typeof intervalMs === "number" ? intervalMs : 500;
        let stopped = false;
        const tick = async () => {
            if (stopped) return;
            try { render(await this.routines()); } catch (err) { /* swallow */ }
            if (!stopped) setTimeout(tick, period);
        };
        tick();
        return () => { stopped = true; };
    };

    Rumo.prototype.fs = {
        // bound when constructed below
    };

    function bindFS(rumo) {
        return {
            put: (path, content) => rumo._call("fs.put", { path, content: encode(content) }),
            get: (path) => rumo._call("fs.get", { path }),
            list: () => rumo._call("fs.list"),
            delete: (path) => rumo._call("fs.delete", { path }),
            clear: () => rumo._call("fs.clear"),
        };
    }

    function encode(v) {
        if (v === null || v === undefined) return new Uint8Array(0);
        if (v instanceof Uint8Array) return v;
        if (v instanceof ArrayBuffer) return new Uint8Array(v);
        if (typeof v === "string") return new TextEncoder().encode(v);
        throw new TypeError("rumo: expected string, Uint8Array, or ArrayBuffer");
    }

    /**
     * Create a new client connected to the rumo SharedWorker.
     * @param {string} [workerURL="./worker.js"] URL of the SharedWorker bootstrap.
     * @returns {Promise<Rumo>} resolves once the runtime has reported `ready`.
     */
    async function connect(workerURL) {
        const r = new Rumo(workerURL);
        r.fs = bindFS(r);
        await r.ready;
        return r;
    }

    root.Rumo = { connect };
})(typeof window !== "undefined" ? window : self);
