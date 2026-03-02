# z/OS Porting Guide (CLI-First)

This guide describes a practical workflow for porting open-source software to z/OS using local `zopen-*` CLI commands.

## Core Rules

1. Use local CLI commands directly (`zopen-generate`, `zopen-build`, etc.). Do not rely on MCP wrapper command names.
2. Use `--help` as the source of truth for flags/syntax in your installed version.
3. Prefer Homebrew formula metadata and upstream project metadata first. Use web search only as fallback.
4. Do not create patch files in `patches/` until the build succeeds.

## Preflight

```bash
command -v zopen-generate zopen-build zopen-info zopen-query zopen-version jq git bump
zopen-generate --help
zopen-build --help
zopen-info --help
zopen-version || zopen-version --help
```

If a command reports `Source the zopen-config prior to running ...`, source zopen config first, then rerun.

## Quickstart (Happy Path)

1. Collect metadata and valid values:
```bash
zopen-generate --list-licenses
zopen-generate --list-categories
zopen-generate --list-build-systems
curl -s "https://formulae.brew.sh/api/formula/${PROJECT}.json" | jq .
```
2. Map dependencies to exact zopen names:
```bash
curl -s https://raw.githubusercontent.com/zopencommunity/meta/refs/heads/main/docs/api/zopen_releases_latest.json \
  | jq -r '.release_data | keys[]' | sort
```
3. Generate project:
```bash
zopen-generate \
  --name <name> \
  --description "<description>" \
  --categories "<cat1 cat2>" \
  --license <spdx_or_unknown> \
  --type BUILD \
  --build-system "<GNU Make|CMake|Go|Gradle|Maven|Meson|Python>" \
  --stable-url "<url>" \
  --stable-deps "<dep1 dep2>" \
  --build-line stable \
  --dev-deps "<dep1 dep2>" \
  --non-interactive
```
4. Build:
```bash
cd <name>port
zopen-build -v
```
5. Iterate on failures until success.
6. After success, create patch + finalize buildenv/bump/.gitignore.

## Detailed Workflow

### 1. Gather Project Information

Collect:
1. Project name (lowercase, no spaces)
2. One-line description
3. Repository URL
4. SPDX license
5. Categories
6. Build system
7. Stable source URL and dependencies

Use Homebrew formula JSON for source/dependency hints:
- `https://formulae.brew.sh/api/formula/${PROJECT}.json`

Find similar existing ports for reference (especially `buildenv` patterns):
- `https://raw.githubusercontent.com/zopencommunity/meta/refs/heads/main/docs/api/zopen_releases_latest.json`
- `https://github.com/zopencommunity/<similarport>/blob/main/buildenv`

### 2. Map Dependencies (Strict)

You must map dependencies to exact package names from `zopen_releases_latest.json`.

Rules:
1. Start with brew dependencies.
2. Map each dependency to exact zopen package name.
3. Keep required dependencies only.
4. If a required build dependency is missing in zopen package list, fail and explain why.

Special names:
- Python runtime/compiler checks: `check_python` (not `python`)
- Go compiler checks: `check_go` (not `go`)

Additional rule:
- If `flex` is needed, add `m4` before `flex`.
- If `cmake` is needed, add `make` as a dependency as well.

### 3. Generate Port Skeleton

Before generating, confirm `<name>port` does not already exist. If it exists, ask whether to proceed.

Required generation flags:
- `--name`
- `--description`
- `--categories`
- `--license`
- `--type`
- `--build-system`
- `--stable-url`
- `--stable-deps`
- `--build-line`

Optional generation flags:
- `--dev-url`
- `--dev-deps`
- `--runtime-deps`
- `--force` (if regenerating existing directory)

Notes:
- For Go-based projects, use `--build-system Go`.
- Default to `--build-line stable` unless there is a strong reason to start with dev.
- Keep upstream source URLs (`--stable-url` / `--dev-url`) as `https://` URLs.

### 4. Build

Run from project directory:
```bash
cd <name>port
zopen-build -v
```

Useful build flags:
- `--build stable|dev`
- `-f, --force-rebuild` when you need bootstrap/configure rerun
- `-g, --get-source` if you only need source + patch apply

### 5. Failure Loop

If build fails:
1. Read latest logs in `log.STABLE` or `log.DEV`.
2. Identify root cause.
3. Modify source or `buildenv`.
4. Re-run `zopen-build -v`.
5. Repeat until success.

Important:
- Once source is extracted by build, edit source directly first.
- Do not create `patches/PR*.patch` until build succeeds.

### 6. Common Failure Patterns

1. `configure: not found`
- If `configure.ac` exists: set `ZOPEN_BOOTSTRAP` (for example `./autogen.sh`).
- If project has no configure step: set `ZOPEN_CONFIGURE="skip"`.

2. Missing headers/libraries
- Add missing dependency in buildenv using exact zopen package names.
- For C runtime compatibility gaps, evaluate `zoslib` or guarded source changes.

3. Platform-specific APIs/syscalls missing
- Guard with `#ifdef __MVS__` and provide z/OS-safe path.

4. Missing functions/macros
- Implement missing function in source when appropriate.
- For `O_NOFOLLOW`-style macro gaps, add `-D__XPLAT` to `ZOPEN_EXTRA_CPPFLAGS` and rebuild (`-f` when needed).

5. Encoding issues (EBCDIC/ASCII)
- Apply source/runtime changes for tagging/auto-conversion where required.

6. Build-system flag tuning
- Add `ZOPEN_EXTRA_CFLAGS`, `ZOPEN_EXTRA_CXXFLAGS`, `ZOPEN_EXTRA_CPPFLAGS`, `ZOPEN_EXTRA_LDFLAGS`, `ZOPEN_EXTRA_LIBS` as needed.

### 7. Post-Success Finalization

After successful build:

1. Create patch from modified source:
```bash
# from extracted source directory inside <name>port
git diff HEAD > ../patches/PR1.patch
```

2. Verify package info and smoke-test binaries:
```bash
zopen-info <name>
```

3. If port produces libs/headers for downstream ports, update `zopen_append_to_env` in `buildenv`:
```bash
zopen_append_to_env() {
  cat <<END
if [ ! -z "\$ZOPEN_IN_ZOPEN_BUILD" ]; then
  export ZOPEN_EXTRA_CFLAGS="\${ZOPEN_EXTRA_CFLAGS} -I\${PWD}/include"
  export ZOPEN_EXTRA_CXXFLAGS="\${ZOPEN_EXTRA_CXXFLAGS} -I\${PWD}/include"
  export ZOPEN_EXTRA_LDFLAGS="\${ZOPEN_EXTRA_LDFLAGS} -L\${PWD}/lib"
  export ZOPEN_EXTRA_LIBS="\${ZOPEN_EXTRA_LIBS} -l<library_name>"
fi
END
}
```

4. Update versioning/bump configuration in `buildenv`:
- Set version variable (VRM).
- Ensure bump line matches source style (tarball/git).
- Use version variable in `ZOPEN_STABLE_URL` or `ZOPEN_STABLE_TAG`.
- Ensure `zopen_get_version()` returns that variable.
- Validate:
```bash
bump --help
bump current buildenv
bump check buildenv
```

5. Update `.gitignore` to ignore extracted source dirs:
```bash
echo "" >> .gitignore
echo "# Ignore source directories created by zopen-build" >> .gitignore
echo "<package-name>-*/" >> .gitignore
```

6. Document changes in `patches/README.md`:
- source modifications
- buildenv customizations
- dependencies added
- known limitations

## Bump Line Guidance (Condensed)

Tarball style:
```bash
# bump: <package>-version /<VAR_NAME>="(.*)"/ <stable_url_with_version>|semver:*
<VAR_NAME>="V.R.M"
```

Git style:
```bash
# bump: <package>-version /<VAR_NAME>="(.*)"/ git:<repo_url>|semver:*
<VAR_NAME>="V.R.M"
```

Use Homebrew `livecheck` regex as input when available:
1. Read brew JSON `ruby_source_path`.
2. Fetch formula file from homebrew-core.
3. Extract/adapt `livecheck` regex for bump.

## Go Dependency Strategy (When Needed)

For failing Go dependencies:
1. Clone problematic dependency beside main project.
2. Check out fixed tag/version.
3. Apply wharf patch (or local patch).
4. Create `go work` including main project + patched dependency.
5. Run `wharf ./<project>/...`.
6. Return to main project and continue build.

Key defaults for Go ports:
- `export CGO_ENABLED=0` unless required otherwise.
- In `zopen_init()`, unset `CC`/`CXX` when appropriate.
- Clean workspace artifacts in `zopen_clean()`.

References:
- `https://raw.githubusercontent.com/ZOSOpenTools/wharf/main/deps-patches/`
- `https://github.com/zopencommunity/crushport/blob/main/buildenv`
- `https://github.com/zopencommunity/gitlab-runnerport/blob/main/buildenv`
- `https://github.com/zopencommunity/gumport/blob/main/buildenv`

## Optional: Create Repo and CI Job

After successful local porting and verification:

Prerequisites:
- These steps are for core contributors with appropriate permissions.
- `zopen-create-repo` requires GitHub CLI (`gh`) and token auth (`GITHUB_TOKEN` or `--github-token`).

1. Ask the user whether to create a GitHub repository now.
2. Create repository (core contributors/admins):
```bash
zopen-create-repo --help
zopen-create-repo -n <name> -d "zopen port of <name>"
```

3. Push code using SSH remote:
```bash
cd <name>port
git init
git remote add origin git@github.com:zopencommunity/<name>port.git
git add .
git commit -m "Initial port of <name>"
git push origin main
```

4. Ask the user whether to create a Jenkins CI job now.
5. Create Jenkins CI job:
```bash
zopen-create-cicd-job --help
zopen-create-cicd-job -n <name> -b stable -s cicd-stable.groovy -r yes
```

## Build Analysis Shortcut

If user asks only for failure analysis:
1. Read latest files in `log.STABLE`/`log.DEV`.
2. Identify root cause.
3. Propose concrete fixes.

## Debugging on z/OS

Use low-friction techniques:

```c
fprintf(stderr, "Debug: var=%d\n", var);

#ifdef __MVS__
__display_backtrace(2);
#endif
```

## Done Checklist

A port is done when all are true:
1. `zopen-build` succeeds.
2. Required dependencies are correctly mapped to exact zopen names.
3. Source edits are captured in `patches/PR*.patch` after successful build.
4. `buildenv` version + bump line are valid (`bump current/check`).
5. `.gitignore` excludes extracted source directories.
6. `patches/README.md` documents modifications and caveats.
7. (Optional) repo/CI created and validated.

## Reference Commands

```bash
zopen-version
zopen-generate --help
zopen-build --help
zopen-info --help
zopen-query --help
zopen-create-repo --help
zopen-create-cicd-job --help
```

## Reference URLs

- zopen releases index: `https://raw.githubusercontent.com/zopencommunity/meta/refs/heads/main/docs/api/zopen_releases_latest.json`
- Homebrew formula JSON: `https://formulae.brew.sh/api/formula/${PROJECT}.json`
- Homebrew core formula source: `https://raw.githubusercontent.com/Homebrew/homebrew-core/refs/heads/main/{ruby_source_path}`
- bump tool: `https://github.com/wader/bump`
- zoslib: `https://github.com/ibmruntimes/zoslib/tree/zopen2`
