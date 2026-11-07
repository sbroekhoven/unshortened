# unshortened

A small web app that shows where a short link really goes. Paste a URL and
unshortened follows every redirect step by step, showing the URL, HTTP status
code and protocol of each hop, without opening the final page in your browser.

Built with Go's standard library and [Pico CSS](https://picocss.com/). The
redirect lookups use
[sbroekhoven/redirects](https://github.com/sbroekhoven/redirects).

## Running with Docker

A container image is published to the GitHub Container Registry:

```bash
docker run -d -p 8080:8080 ghcr.io/sbroekhoven/unshortened:latest
```

Then open <http://localhost:8080>.

Or use Docker Compose:

```bash
cp .env.example .env   # adjust as needed
docker compose up -d
```

## Running a release binary

Download the archive for your platform from the
[releases page](https://github.com/sbroekhoven/unshortened/releases), unpack it
and run `./unshortened`. Then open <http://localhost:8432>.

## Running from source

Requires Go 1.26 or newer. The templates and stylesheets are built into the
binary, so it runs from any directory:

```bash
go run .
```

Then open <http://localhost:8432>.

## Configuration

Settings are read from environment variables, or from a `.env` file in the
working directory if one exists.

| Variable     | Default       | Description                                  |
|--------------|---------------|----------------------------------------------|
| `SITE_TITLE` | `Unshortened` | Title shown in the page header and tab.      |
| `PORT`       | `8432`        | Port the server listens on. The Docker image and Compose file use `8080`. |

## Running behind a reverse proxy

The app sends its own content security headers, such as the
Content-Security-Policy, so don't add a second CSP in the proxy: the browser
would apply both and the stricter one wins.

The proxy should:

- handle TLS and redirect HTTP to HTTPS
- set HSTS (the app also sends it when the proxy reports HTTPS)
- pass `X-Forwarded-Proto` and `X-Forwarded-For`

## Releases

Every push to `main` publishes the `latest` image. Pushing a version tag
creates a release:

```bash
git tag v1.2.0
git push origin v1.2.0
```

This publishes the image tags `1.2.0` and `1.2`, and
[GoReleaser](https://goreleaser.com/) creates a
[GitHub release](https://github.com/sbroekhoven/unshortened/releases) with
binaries for Linux, macOS and Windows (amd64 and arm64). The binaries contain
the templates and stylesheets, so they run on their own.

Dependabot keeps Go modules, the Docker base image and GitHub Actions up to
date. Patch and minor updates are merged automatically once the build passes.

## License

[MIT](LICENSE)
