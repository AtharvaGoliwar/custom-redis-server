# 🔐 RedVault — Custom Redis-Inspired Server with Vault-Grade Features

RedVault is a fully custom-built Redis clone written in **Go** — designed from scratch with extended features such as data persistence (AOF & RDB), TTL support, transactions, access control, and a security-first foundation for building a digital vault system.

---

## 🚀 Features

-  RESP Protocol (Redis Serialization Protocol) compliant parser
-  Supports core Redis commands: `SET`, `GET`, `DEL`, `EXISTS`, `EXPIRE`, `TTL`, `PERSIST`
-  Transactions support via `MULTI` / `EXEC`
-  TTL-based key expiration (both passive & active eviction)
-  Append Only File (AOF) logging for write durability
-  RDB Snapshots for efficient state recovery
-  AOF Rewrite to optimize log size after snapshot
-  Authentication & Access Control:
    - Add Users
    - Delete Users
    - List Users
    - Group-based data isolation (multi-tenant design)
-  Pub/Sub system for real-time messaging
-  Clean connection handling with graceful shutdown
-  Redis-Benchmark compatible and stress-tested

---

## 🏗 Architecture

- **Language:** Go
- **Persistence:** 
  - AOF (Append Only File)
  - RDB Snapshotting
- **Concurrency:** Goroutines & Mutex Locks for safe concurrent access
- **Protocol:** RESP (Custom Parser & Encoder)
- **Security:** 
  - User & Group Access Control
  - Encryption-ready (planned VaultGuard encryption layer)

---

## 📦 Tech Stack

- `Go` (Golang)
- `RESP` protocol (custom implementation)
- `File I/O` for AOF & RDB persistence
- `Sync primitives` (RWMutex)
- `Net.TCP` server for client connections
- `redis-benchmark` compatible

---

## 💻 Usage

### 1️⃣ Build the server:

```bash
go build -o redvault main.go
````

### 2️⃣ Start the server:

```bash
./redvault
```

Server will run on `localhost:6380` by default.

### 3️⃣ Use redis-cli to test:

```bash
redis-cli -p 6380
```

Sample commands:

```bash
SET key1 value1
GET key1
EXPIRE key1 10
TTL key1
MULTI
SET key2 value2
DEL key1
EXEC
LISTUSERS
```

---

## 🧪 Benchmarking

Using redis-benchmark:

```bash
redis-benchmark -p 6380 -t set,get -n 100000 -q
```

Achieved:

* SET: \~31k - 33k ops/sec
* GET: \~31k - 38k ops/sec


## 🚀 Future Roadmap

*  Encrypted file storage & transfer layer
*  Real-time peer-to-peer messaging
*  Multi-client sync
*  Encryption Key Management
*  Distributed cluster mode
*  Cloud Deployment

---

## ⚠ Disclaimer

RedVault is a learning & research project to deeply understand how Redis works internally along with designing production-grade secure storage systems. Not intended for production use (yet).

---

## ✨ Author

Built by **Atharva Goliwar**


---
