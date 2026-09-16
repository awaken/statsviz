Contributing
============

First of all, thank you for considering to contribute to Statsviz!

Pull-requests are welcome!

## Go library

Statsviz Go public API surface is relatively limited, by design, and it's highly
unlikely that that will change. However new options can be added to
`statsviz.Register` and `statsviz.NewServer` without breaking compatibility.

Big changes should be discussed on the issue tracker prior to start working on
the code.

If you've decided to contribute, thank you so much, please comment on the
existing issue or create one stating what you want to tackle.


## User interface (html/css/javascript)

The user interface aims to be simple, light and minimal.

To bootstrap the UI for development:
 - cd to `internal/static`
 - run `npm install`
 - run `npm run dev` and leave it running
 - in another terminal, cd to an example, for example `_example/default`
 - run `go mod edit -replace=github.com/arl/statsviz=../../` to build the
   example with your local version of the Go code. If you haven't touched to the
   Go code you can skip this step.

To build the production UI:
 - cd to `internal/static`
 - prepare the Docker toolchain and dependency cache below
 - run `make release RELEASE_IMAGE="$release_image"`
 - only commit `dist.zip`. `dist` directory is ignored.

### Building the UI without installing Node.js (Docker)

If you don't want to install Node.js/npm on your machine, you can drive the
whole frontend workflow through Docker using the `Makefile` in
`internal/static/`. It builds a tiny image on top of `node:24-alpine` (see
`internal/static/Dockerfile`) and runs everything as your host UID/GID, so
files created in `node_modules/`, `dist/`, `dist.zip` and `package-lock.json`
are owned by you and not root.

Common targets (run from `internal/static/`):

 - `make dev` — start the Vite dev server on http://localhost:5173. Override
   the port with `PORT=3000 make dev`.
 - `make build` — `npm ci` + `npm run build`.
 - `make zip` — build and regenerate `dist.zip`.
 - `make release RELEASE_IMAGE="$release_image"` — offline clean build from
   the lockfile in an immutable toolchain image, then deterministic packaging.
 - `make install` — `npm install` only.
 - `make shell` — open an interactive shell in the container.
 - `make clean` / `make distclean` — remove build artifacts / also drop the
   Docker image and npm-cache volume.
 - `make help` — list all targets.

To run **any other npm/npx/shell command** inside the container, use the
generic passthrough target:

```sh
make run CMD="npm outdated"
make run CMD="npm ls plotly.js-cartesian-dist"
make run CMD="npx --yes npm-check-updates"
```

Every `image` prerequisite runs `docker build`; Docker reuses unchanged layers.
Changes to `IMAGE_TAG` or `NODE_IMAGE` and deleted images cannot be hidden by a
local stamp. The npm cache remains in the named `statsviz-npm-cache` volume.

Prepare a release toolchain and its lockfile cache on the build machine:

```sh
make image
release_image=$(docker image inspect --format '{{.Id}}' statsviz-ui)
make run CMD="npm ci --no-audit --no-fund"
make release RELEASE_IMAGE="$release_image"
```

Record the complete image digest with the release and retain that image. Release
runs use `--pull=never --network=none`; a missing image or cache fails instead of
fetching new inputs. Digest-pinned repository references also work. Reuse the
same image, architecture, source tree, lockfile and dependency cache to reproduce
an archive. The development `NODE_IMAGE` tag does not select the release image.

`npm ci` preserves the lockfile. ZIP entries have sorted paths, UTC timestamps
fixed at 1980-01-01, mode 0644 and no host extra fields. Source files are not
modified. Packaging atomically replaces `dist.zip` only after success. Run
`npm test` and `sh scripts/make_test.sh` for build-tool regressions.


### Bumping npm dependencies

To bump all dependencies to the latest **minor/patch** versions allowed by the
`^` ranges in `package.json`, then rebuild:

```sh
cd internal/static
make run CMD="npm update"
make run CMD="npm outdated"   # sanity check what's still behind (major bumps)
make release RELEASE_IMAGE="$release_image"  # rebuild dist/ and dist.zip
./scripts/checkzip.sh         # verify dist.zip is up to date
```

To also bump **major** versions, use `npm-check-updates`:

```sh
make run CMD="npx --yes npm-check-updates -u"
make run CMD="npm install"
make release RELEASE_IMAGE="$release_image"
```

Then commit `package.json`, `package-lock.json` and the regenerated `dist.zip`.


Assets are located in the `internal/static` directory and are embedded with
[`go:embed`](https://pkg.go.dev/embed). To reduce the space taken by the assets
in the final binary, the `dist` directory is zipped into `dist.zip`. Use
`scripts/zip.sh` to do it. At runtime, when Statsviz serves the UI, the
`dist.zip` is then decompressed into a `fs.FS`, served via
`http.FileServerFS()`.


## `STATSVIZ_DEBUG`

Declare `STATSVIZ_DEBUG=1` environment variable when you develop in order to:
 - print WebSocket errors on standard error.
 - bypass WebSocket origin checks

Obviously, this is not recommended for production use!

## Documentation

No contribution is too small. Improvements to code, comments or README
are welcome!


## Examples

There are many Go libraries to handle HTTP requests, routing, etc..

Feel free to add an example to show how to register Statsviz with your favourite
library.

To do so, please add a directory under `./_example`. For instance, if you want to add an
example showing how to register Statsviz within library `foobar`:

 - create a directory `./_example/foobar/`
 - create a file `./_example/foobar/main.go`
 - call `go example.Work()` as the first line of your example (see other
   examples). This forces the garbage collector to _do something_ so that
   Statsviz interface won't remain static when an user runs your example.
 - the code should be `gofmt`ed
 - the example should compile and run
 - when ran, Statsviz interface should be accessible at http://localhost:8080/debug/statsviz


Thank you!
