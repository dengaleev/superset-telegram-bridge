# Releasing

Releases are cut by pushing a semver tag. The
[`release` workflow](.github/workflows/release.yml) runs
[GoReleaser](https://goreleaser.com/), which builds the binaries, generates the
changelog and checksums, and publishes the GitHub release and the multi-arch
container image to GHCR.

## Versioning

This project follows [Semantic Versioning](https://semver.org/). Tags are
prefixed with `v` (e.g. `v1.2.3`).

- **patch** (`v1.2.3` → `v1.2.4`) — bug fixes, no behaviour change.
- **minor** (`v1.2.0` → `v1.3.0`) — new, backward-compatible features.
- **major** (`v1.0.0` → `v2.0.0`) — breaking changes (config, endpoints, …).

## Cutting a release

1. Make sure `main` is green (CI passes) and you have everything you want in
   the release.
2. Pick the next version per the rules above.
3. Tag and push:

   ```bash
   git checkout main && git pull
   git tag -a v1.2.3 -m "v1.2.3"
   git push origin v1.2.3
   ```

4. Watch the **release** workflow in the Actions tab. On success it produces:
   - a GitHub release with auto-generated notes and `checksums.txt`,
   - `tar.gz` archives for `linux/amd64` and `linux/arm64`,
   - `ghcr.io/dengaleev/superset-telegram-bridge:1.2.3` and `:latest`.

## Verifying

```bash
docker pull ghcr.io/dengaleev/superset-telegram-bridge:1.2.3
# The bridge logs its embedded version on startup:
#   {"level":"INFO","msg":"starting","version":"1.2.3",...}
docker run --rm -e TELEGRAM_TOKEN=x -e TELEGRAM_CHAT_ID=x \
  ghcr.io/dengaleev/superset-telegram-bridge:1.2.3
```

## Notes

- The changelog groups commits by Conventional Commit prefix (`feat:`, `fix:`,
  …). Using those prefixes keeps the release notes tidy, but they are not
  required.
- To preview what a release would contain without publishing, run
  `goreleaser release --snapshot --clean` locally (needs GoReleaser and Docker).
- There is no rollback. To supersede a bad release, fix forward and tag a new
  patch version.
