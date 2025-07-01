````markdown
# 🧠 Fredis - A Minimal Redis-Inspired In-Memory Database

**Fredis** is a lightweight, Redis-like in-memory key-value store built from scratch in **Go**. It speaks a simplified version of the RESP protocol and supports basic Redis commands along with TTL-based expiry.

---

## 🔧 Features

- ⚡ Fast, single-node in-memory key-value store
- ✅ Supports basic Redis commands over TCP
- 🔁 RESP (REdis Serialization Protocol) compatible
- ⏱️ TTL and expiry support
- 🔐 Thread-safe with fine-grained locking
- 🖥️ Python CLI for command-line interaction
- 🧠 Configurable eviction policies (LRU,RandomEviction, etc.) Implemented
- 💾 Append-only file (AOF) persistence Implemented

---

## ✅ Supported Commands

| Command                               | Description                                       |
|---------------------------------------|---------------------------------------------------|
| `PING`                                | Returns `PONG` or a custom message                |
| `SET key value`                       | Sets the value for a key                          |
| `GET key`                             | Retrieves the value of a key                      |
| `DEL key`                             | Deletes the specified key                         |
| `EXPIRE key sec`                      | Sets an expiry time in seconds for a given key    |
| `TTL key`                             | Returns the remaining time-to-live in seconds     |
| `CONFIG SET eviction-policy` <policy> | Returns the remaining time-to-live in seconds     |

---

## 🧠 Concurrency & Memory Safety

Fredis uses a two-level locking system:

- A **global `RWMutex`** for the main store (`map[string]*Value`)
- A **per-key `RWMutex`** inside each value entry for fine-grained access

This ensures:

- Safe concurrent reads/writes
- No deadlocks (with a consistent lock acquisition order)
- Expiry checks handled within the value lock
- Deleted keys handled safely without panics or race conditions

---

## 🧪 Python CLI: `fredis-cli`

A minimal CLI client written in Python is available.It lets you send RESP-encoded commands to your Fredis server:

## 📦 Prebuilt CLI Executable

https://github.com/Ghlakshman/fredis-cli

the repository implements a CLI using python with minimal implementations to interact with the server there is packaged executable in the releases section which lets u run the cli without installing python in you local machine
### 🔹 Example

```bash
$ python main.py
Connected to Fredis server at 127.0.0.1 6379
fredis> SET foo bar
OK
fredis> GET foo
bar
fredis> EXPIRE foo 10
1
fredis> TTL foo
9
fredis> PING
PONG
````

---

## 🚀 Running the Server

Make sure you have Go installed. Then:

```bash
go run main.go
```

This starts the Fredis server on `localhost:6379`.

---

## 📌 Future Enhancements

The following features are on the roadmap:

* ✅ Support for `SET` options (`NX`, `XX`, `EX`, `PX`)
* 🧹 Key reaper to proactively delete expired keys
* 💾 Snapshotting (RDB-style) for full-dump backups

---




