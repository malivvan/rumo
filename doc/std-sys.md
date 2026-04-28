---
title: Standard Library - sys
---

## Import

```golang
sys := import("sys")
```

## Constants

- `EPERM`
- `ENOENT`
- `ESRCH`
- `EINTR`
- `EIO`
- `ENXIO`
- `E2BIG`
- `EBADF`
- `EAGAIN`
- `ENOMEM`
- `EACCES`
- `EFAULT`
- `EBUSY`
- `EEXIST`
- `EXDEV`
- `ENODEV`
- `ENOTDIR`
- `EISDIR`
- `EINVAL`
- `ENFILE`
- `EMFILE`
- `EFBIG`
- `ENOSPC`
- `ESPIPE`
- `EROFS`
- `EMLINK`
- `EPIPE`
- `ENAMETOOLONG`
- `ENOSYS`
- `ENOTEMPTY`
- `ELOOP`
- `ETIMEDOUT`
- `ECONNREFUSED`
- `ECONNRESET`
- `EHOSTUNREACH`
- `ENETUNREACH`
- `EPROTONOSUPPORT`
- `EAFNOSUPPORT`
- `EADDRINUSE`
- `EADDRNOTAVAIL`
- `ENOTSOCK`
- `EALREADY`
- `EINPROGRESS`
- `EISCONN`
- `ENOTCONN`
- `EMSGSIZE`
- `SIGHUP`
- `SIGINT`
- `SIGQUIT`
- `SIGILL`
- `SIGTRAP`
- `SIGABRT`
- `SIGFPE`
- `SIGKILL`
- `SIGSEGV`
- `SIGPIPE`
- `SIGALRM`
- `SIGTERM`

## Functions

- `goos() => string`: target operating system (runtime.GOOS)
- `goarch() => string`: target architecture (runtime.GOARCH)
- `compiler() => string`: Go compiler used to build the host (gc/gccgo)
- `go_version() => string`: Go runtime version
- `num_cpu() => int`: number of logical CPUs available
- `num_goroutine() => int`: number of currently running goroutines
- `page_size() => int`: memory page size in bytes
- `getpid() => int`: calling process id
- `getppid() => int`: parent process id
- `getuid() => int`: real user id (-1 if unsupported)
- `geteuid() => int`: effective user id (-1 if unsupported)
- `getgid() => int`: real group id (-1 if unsupported)
- `getegid() => int`: effective group id (-1 if unsupported)
- `getgroups() => error`: supplementary group ids
- `hostname() => error`: host name reported by the kernel
- `getwd() => error`: current working directory
- `getenv(key string) => string`: value of an env var ('' if unset)
- `setenv(key string, value string)`: error                set an env var
- `unsetenv(key string)`: error                            remove an env var
- `clearenv()`: remove every env var
- `environ() => []string`: all 'KEY=VAL' env entries
- `lookup_env(key string) => bool`: env var with explicit presence flag
- `expand_env(s string) => string`: ${var}/$var substitution against the current env
- `exit(code int)`: terminate the host process immediately
- `errno_str(errno int) => string`: human-readable error string for the given errno
- `kill(pid int, sig int)`: error  send a signal to a process
