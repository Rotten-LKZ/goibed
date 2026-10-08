# goibed
A simple image hosting service written in Go.

It uses only Go's standard `net/http` package for HTTP handling and a single token for admin authentication. Image conversion is
delegated to ImageMagick and libheif rather than implemented in Go.

This project is intentionally small: I built it to host images for my blog and to practice writing Go. Most of the code is handwritten, with AI used to help look up documentation.

A few things to know:

- Uploaded images are immediately converted to AVIF to save storage space. The original files are deleted, so don't use this
service as the only place you keep valuable photos.
- Deletion is permanent. There is no soft-delete or recovery feature.
- The service does not encode or decode images itself; it runs `magick`. Make sure your ImageMagick installation supports AVIF
encoding.

## Usage

Download the binary for your platform from [GitHub Releases](https://github.com/Rotten-LKZ/goibed/releases) and `config.example.json`, create your own configuration file, then run:

```bash
./goibed --config /path/to/your/config.json
```

`./goibed --version` prints the release tag and exits without loading configuration or starting the service. Locally built binaries print `dev` unless built with `-ldflags "-X main.version=<version>"`.

## Run with source code

Clone the repository, create a configuration file, and run the service:
```bash
git clone --depth=1 https://github.com/Rotten-LKZ/goibed.git
cd goibed
go mod tidy
cp config.example.json config.json
vim config.json # whatever editor you use
go run .
```

## Debug logging

Set `LOG_LEVEL=debug` to emit JSON debug logs, or `ENV=development` for human-readable debug logs. By default only info-level and higher logs are emitted. Debug logs cover incoming and completed HTTP requests, authentication outcomes, uploads and deduplication, reconversion batches, queue activity, ImageMagick execution, metadata extraction, database updates, and original-file cleanup. Conversion is asynchronous: the upload response can precede the `conversion completed` log. Logs include image IDs, filenames, hashes, filesystem paths, and client addresses; do not expose debug logs publicly. Tokens, JWTs, and request bodies are not logged.

## API Reference

The service listens on `:8080`. Except for the home page and image files, endpoints return JSON: `{"code": HTTP status code, "msg": "message", "data": ...}`. The `data` field is omitted when absent. Errors use the same format, typically with HTTP status 400, 401, 404, or 500.

`/api/upload` and `/api/manage/*` require the `JWT_TOKEN` cookie obtained after login; a missing or invalid cookie returns 401. A successful login sets a 30-day `HttpOnly`, `Secure` cookie (use HTTPS).

| Method | Path | Request | Success response (HTTP 200) |
| --- | --- | --- | --- |
| GET | `/` | None | Plain text `Hello` |
| GET | `/i/{filename}` | `{filename}` is a UUID followed by `.avif` | Image file; 404 if not found |
| GET | `/info/{filename}` | Same as above | `data` contains an image information object (fields below) |
| POST | `/api/login` | JSON `{"token":"admin token"}` | `msg: "Successful"` and a `JWT_TOKEN` cookie. An incorrect token returns HTTP 200 with `msg: "Wrong token"` and no cookie |
| POST | `/api/upload` | `multipart/form-data` with a `file` field | `msg: "Successful"`; `data` is an image information object (fields below), not a hash/URL pair. An existing file with the same SHA-256 hash returns its existing image object; otherwise AVIF conversion is queued in the background, so the image may not be available immediately | 
| POST | `/api/manage/list` | JSON `{"page":1,"step":20,"keyword":"cat","start_time":0,"end_time":1790944133293}`; page/step zero or omitted default to 1/20; optional keyword matches part of `file_name`; optional time bounds filter `created_at` inclusively | `data` is an array of image information objects, newest first; returns 400 if both bounds are supplied and `end_time < start_time` |
| POST | `/api/manage/update` | JSON `{"ID":"image UUID","file_name":"new filename"}` | `msg: "Successful"`, no `data`; invalid or unknown IDs return 400 |
| POST | `/api/manage/delete` | JSON `{"ID":"image UUID"}` | `msg: "Successful"`, no `data`; invalid or unknown IDs return 400. Deletion is permanent |
| POST | `/api/manage/reconvert` | JSON `{"ID":"image UUID"}` to reconvert one image, or `{"all":false}` to reconvert only images whose stored path does not end in `.avif`, or `{"all":true}` to reconvert every image; `all` is ignored when `ID` is provided | `msg: "Successful"`, no `data`; a request with `ID` queues that image immediately, while batch reconversion runs in the background. Invalid UUIDs, unknown IDs, or a full conversion queue return 400 or 500 respectively |

The `file` field is streamed to disk and limited to 32 MiB (33,554,432 bytes). Larger files return HTTP 413; this limit applies to the file, not the multipart envelope.

`start_time` and `end_time` are signed Unix milliseconds (instants relative to 1970-01-01 00:00:00 UTC), like the response timestamps. Omit a bound or set it to `null` to leave that side unbounded; `0` is a valid bound, and negative values represent times before the Unix epoch. Convert local calendar dates to the intended timezone's instants before sending milliseconds.

SQLite stores `created_at` and `updated_at` as UTC Unix milliseconds (INTEGER), so numeric comparisons and sorting follow chronological order.

The stored image path ending in `.avif` is the service's main signal that conversion succeeded. The reconvert API is primarily intended to retry images whose conversion failed; `{"all":false}` selects images without an `.avif` path. After successful conversion, image responses use `Cache-Control: public, max-age=31536000` (one year); images not yet converted use `Cache-Control: no-cache`. Responses also include a quoted `ETag` derived from the image UUID and database `updated_at`, and `Last-Modified` from that timestamp. Clients may revalidate with `If-None-Match` or `If-Modified-Since`; unchanged images return 304. Because `{"all":true}` and reconversion by `ID` can also reconvert successful images at the same URL, clients with a fresh one-year cached response may continue seeing the old image until that cache expires.

Image information objects (from `/info`, `/api/upload`, and `/api/manage/list`) include `ID`, `type`, `file_hash`, `file_name`, `width`, `height`, `size` (bytes), `created_at`, and `updated_at` (Unix milliseconds). File paths are not returned. For a new upload, conversion runs asynchronously, so dimensions and size may initially be zero. Example:

```json
{
  "code": 200,
  "msg": "Successful",
  "data": {
    "ID": "f86fd195-572e-404b-8972-3aac538271c4",
    "type": "image",
    "file_hash": "d2b38ef638293bfea60cc8910e3c3fd8d9806f960aedadb5b08da8c13e8d4339",
    "file_name": "wallhaven-yq8w67.jpg",
    "width": 7000,
    "height": 3267,
    "size": 480913,
    "created_at": 1790944133293,
    "updated_at": 1790944139184
  }
}
```
