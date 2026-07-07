# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

mini-GED — a tiny demo document-management app backed by [Garage](https://garagehq.deuxfleurs.fr/) (an S3-compatible object store). It exists to demonstrate, live, that Garage lacks native S3 object versioning: `handleOverwriteDemo` writes two versions of the same key back-to-back and shows only the latter survives.

The whole app is two files:
- `main.go` — HTTP handlers (`/`, `/upload`, `/download`, `/delete`, `/overwrite-demo`) and S3 client setup via aws-sdk-go-v2.
- `template.go` — a single inline `html/template` (`pageTmpl`) rendering the whole UI. No separate template files, no CSS/JS assets.

UI copy is in French; keep new user-facing strings consistent with that.

## Running

Requires a reachable Garage (or any S3-compatible) endpoint. Config is env-vars only, no flags, no config file:

```bash
S3_ENDPOINT=http://<lxc-ip>:3900 \
S3_REGION=garage-demo \
S3_BUCKET=demo-documents \
S3_ACCESS_KEY=xxxx \
S3_SECRET_KEY=xxxx \
go run main.go
```

`LISTEN_ADDR` optionally overrides the default `:8080`. Then browse to `http://localhost:8080`.

`S3_ENDPOINT`, `S3_BUCKET`, `S3_ACCESS_KEY`, `S3_SECRET_KEY` are required (the process calls `log.Fatalf` if missing); `S3_REGION` defaults to `garage-demo`.

## Build/lint

Standard Go toolchain, no Makefile or build scripts:

```bash
go build ./...
go vet ./...
gofmt -l .
```

There are no tests in this repo.

## Architecture notes

- Global `s3Client *s3.Client` and `bucket string` are set once in `main()` and used directly by handlers — no dependency injection, no interfaces (single implementation, kept simple on purpose).
- `s3Client` is configured with `UsePathStyle = true`, required because Garage in this demo setup has no wildcard DNS for virtual-hosted-style bucket URLs.
- Client-provided filenames (`header.Filename`) are used directly as S3 keys with no sanitization — this is demo code, not hardened for untrusted input.
- `stringReadSeeker` in `main.go` is a minimal hand-rolled `io.ReadSeeker` over a string, used only by the overwrite demo, to give the S3 SDK a seekable body for content-length calculation.
- Flash messages are passed via a `flash` query param on redirect (`redirectFlash`), read back in `handleIndex`, and rendered into the template — there's no session/cookie state anywhere in the app.
