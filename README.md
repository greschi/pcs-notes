# PCS Notes & Nginx Unix Socket Proxy on Debian Trixie

A secure, sandboxed containerized architecture running **Debian Trixie**, **Nginx**, and a **Go note-taking application** communicating over a **Unix domain socket**.

## Architecture Overview
- **Base Image**: `debian:trixie-slim` with systemd running as PID 1.
- **Nginx**: Acts as the reverse proxy on port `8080` (mapped to port `80` inside the container), proxying all `/notes*` requests to a Unix domain socket (`/run/notes/notes.sock`).
- **Go Note Application**: A lightweight, zero-external-dependency CRUD note-taking app with tag filtering, keyword search, and JSON file persistence (`notes.json`).
- **Security & Sandboxing**:
  - Runs under a dedicated restricted system user (`notes-user`).
  - Systemd sandboxing (`ProtectSystem=strict`, `ProtectHome=true`, `RestrictAddressFamilies=AF_UNIX`, `SystemCallFilter=@system-service`, etc.).
  - Request payload size limiting (`client_max_body_size 1M` in Nginx and `http.MaxBytesReader` in Go).

---

## Prerequisites
- Docker (with systemd and cgroups v2 support)
- Go (version 1.22+)
- Make

---

## Makefile Targets & Usage

1. **Build and Run Container**:
   ```bash
   sudo make docker-run
   ```
   *Compiles the Go app, builds the Docker image, creates the data directory, and runs the container with systemd support.*

2. **Run Trivy Vulnerability Scan**:
   ```bash
   make trivy-scan
   ```

3. **Stop Container**:
   ```bash
   make docker-stop
   ```

4. **Clean Build Artifacts**:
   ```bash
   make clean
   ```

---

## API Endpoints (`http://localhost:8080/notes`)

- **Create Note**:
  ```bash
  curl -X POST http://localhost:8080/notes \
    -H "Content-Type: application/json" \
    -d '{"title": "First Note", "content": "Hello via Unix socket!", "tags": ["golang", "nginx"]}'
  ```

- **List All Notes**:
  ```bash
  curl http://localhost:8080/notes
  ```

- **Search / Filter by Tag or Keyword**:
  ```bash
  curl "http://localhost:8080/notes?q=Hello"
  curl "http://localhost:8080/notes?tag=golang"
  ```

- **Get Note by ID**:
  ```bash
  curl http://localhost:8080/notes/ID
  ```

- **Update Note**:
  ```bash
  curl -X PUT http://localhost:8080/notes/ID \
    -H "Content-Type: application/json" \
    -d '{"title": "Updated Title", "content": "Updated content"}'
  ```

- **Delete Note**:
  ```bash
  curl -X DELETE http://localhost:8080/notes/ID
  ```
