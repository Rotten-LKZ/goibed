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

Download the binary and `config.example.json`, create your own configuration file, then run:

```bash
./goibed --config /path/to/your/config.json
```

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
