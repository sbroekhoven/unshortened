# unshortened

A small web app that shows where a short link really goes. Paste a URL and
unshortened follows every redirect step by step, showing the URL, HTTP status
code and protocol of each hop, without opening the final page in your browser.

Built with Go and [Echo](https://echo.labstack.com/). The redirect lookups use
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

## Running from source

Requires Go 1.26 or newer. Run it from the repository root, because the
templates are loaded from `html/`:

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
| `SITE_URL`   | (empty)       | Public URL of the site. Not used yet.        |

## Releases

Every push to `main` publishes the `latest` image. Pushing a version tag also
publishes versioned images:

```bash
git tag v1.0.0
git push origin v1.0.0
```

This produces the image tags `1.0.0` and `1.0`.

Dependabot keeps Go modules, the Docker base image and GitHub Actions up to
date. Patch and minor updates are merged automatically once the build passes.

## License

[MIT](LICENSE)
