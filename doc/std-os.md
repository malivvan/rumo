---
title: Standard Library - os
---

## Import

```golang
os := import("os")
```

## Constants

- `o_rdonly`: open the file read-only
- `o_wronly`: open the file write-only
- `o_rdwr`: open the file read-write
- `o_append`: append data to the file when writing
- `o_create`: create a new file if none exists
- `o_excl`: fail if the file already exists
- `o_sync`: open for synchronous I/O
- `o_trunc`: truncate regular writable file when opened
- `mode_dir`
- `mode_append`
- `mode_exclusive`
- `mode_temporary`
- `mode_symlink`
- `mode_device`
- `mode_named_pipe`
- `mode_socket`
- `mode_setuid`
- `mode_setgui`
- `mode_char_device`
- `mode_sticky`
- `mode_type`
- `mode_perm`
- `path_separator`
- `path_list_separator`
- `dev_null`
- `seek_set`
- `seek_cur`
- `seek_end`

## Functions

- `args() => []string`
- `chdir(dir string)`: error
- `chmod(name string, mode int)`: error
- `chown(name string, uid int, gid int)`: error
- `clearenv()`
- `environ() => []string`
- `exit(code int)`
- `expand_env(s string) => string`
- `getegid() => int`
- `getenv(s string) => string`
- `geteuid() => int`
- `getgid() => int`
- `getgroups() => []int`
- `getpagesize() => int`
- `getpid() => int`
- `getppid() => int`
- `getuid() => int`
- `getwd() => string`
- `hostname() => string`
- `lchown(name string, uid int, gid int)`: error
- `link(oldname string, newname string)`: error
- `lookup_env(key string) => bool`
- `mkdir(name string, perm int)`: error
- `mkdir_all(name string, perm int)`: error
- `readlink(name string) => string`
- `remove(name string)`: error
- `remove_all(name string)`: error
- `rename(oldpath string, newpath string)`: error
- `setenv(key string, value string)`: error
- `symlink(oldname string, newname string)`: error
- `temp_dir() => string`
- `truncate(name string, size int)`: error
- `unsetenv(key string)`: error
- `create(name string) => imap(file`: )
- `open(name string) => imap(file`: )
- `open_file(name string, flag int, perm int) => imap(file`: )
- `find_process(pid int) => imap(process`: )
- `start_process(name string, argv array(string)`: , dir string, env array(string)) (process imap(process))
- `exec_look_path(file string) => string`
- `exec(name string, args ...string) => imap(command`: )
- `stat(name string) => imap(fileinfo`: )
- `read_file(name string) => bytes`
