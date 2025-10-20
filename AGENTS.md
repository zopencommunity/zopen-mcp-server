# Claude Instructions for z/OS Software Porting

This document provides step-by-step instructions to port open-source software to z/OS.

## Overview

You have access to zopen tools through MCP that allow you to:
- Generate zopen-compatible project structures
- Build projects on z/OS
- Query package information
- Install and manage z/OS packages
- All zopen tools are compiled with UTF8/ASCII mode turned on. Enhanced Auto-Conversion is enabled meaning files are auto-converted based on the file tag (ccsid) to ASCII / UTF8.
- The library zoslib is used in all C and C++ projects to bridge the gap between z/OS LE runtime environment and the Linux C runtime. Source is here: https://github.com/ibmruntimes/zoslib/tree/zopen2

## Porting to z/OS Workflow

### Step 1: Gather Project Information

Before starting, collect the following information about the project, here on out referred to as ${PROJECT}.

1. **Project Name** (lowercase, no spaces)
2. **Description** (brief, one-sentence summary)
3. **Repository URL** (GitHub or other git repository)
4. **License** (SPDX identifier). Call zopen_generate_list_licenses to see all valid license identifiers
5. **Categories** Call zopen_generate_list_categories to see all valid categories
6. **Build System** Call zopen_generate_list_build_systems to see all valid build systems
7. **Version Check Regex** (for bump configuration - from Homebrew livecheck)

**Action**: Use `zopen_generate_list_licenses`, `zopen_generate_list_categories`, and `zopen_generate_list_build_systems` to get valid options. Use the brew json information to get the additional data such as source location.

**Getting Version Check Regex from Homebrew:**
- From the brew JSON (e.g., `https://formulae.brew.sh/api/formula/${PROJECT}.json`), get the `ruby_source_path` field
- Construct the Homebrew formula URL: `https://raw.githubusercontent.com/Homebrew/homebrew-core/refs/heads/main/{ruby_source_path}`
- Example: `ruby_source_path: "Formula/m/midnight-commander.rb"` → `https://raw.githubusercontent.com/Homebrew/homebrew-core/refs/heads/main/Formula/m/midnight-commander.rb`
- Fetch the formula file and look for the `livecheck` block which contains the version detection regex
- Save this regex for later use in configuring the bump line (Step 6.2)

**Find Similar Projects**: After detecting the tool, find a similar tool from `https://raw.githubusercontent.com/zopencommunity/meta/refs/heads/main/docs/api/zopen_releases_latest.json` for reference. Use the buildenv from that similar project as context, e.g., `https://github.com/zopencommunity/curlport/blob/main/buildenv`.

### Step 2: Generate the zopen Project

Use the `zopen_generate` tool to create the project structure.

Before proceeding to the `zopen_generate` step, ensure that ${PROJECT}port directory does not exist. If it does exist, notify the user and ask the user if they want to proceed.

Follow the advice provided. Do not use web search.

**Required Parameters:**
- `name`: Package name (lowercase)
- `description`: Brief description. It can be a tarball or a git repo. You can find this information from brew. For example for curl, the build deps are in curl https://formulae.brew.sh/api/formula/${PROJECT}.json, where PROJECT is the PROJECT name
- `categories`: Space-delimited categories.  This is from the information collected from Step 1.
- `license`: SPDX license identifier (or "unknown"). This is from the information collected from Step 1.
- `type`: "BUILD" (build from source) or "BARE" (binary download). If there is a build system used, go with BUILD
- `build_system`: The build system used (e.g., "GNU Make"). This is from the information collected from Step 1. For example for curl, the build deps are in curl https://formulae.brew.sh/api/formula/${PROJECT}.json, where PROJECT is the PROJECT name.
- `stable_url`: This is the download url. It can be a tarball or a git repo. You can find this information from brew. For example for curl, the build deps are in curl https://formulae.brew.sh/api/formula/${PROJECT}.json, where PROJECT is the PROJECT name. 
- `build_line`: "stable" or "dev". If unknown, start with "stable".
- `stable_deps`: Space-delimited list of dependencies. You can find this information from brew. For example for curl, the build deps are in curl https://formulae.brew.sh/api/formula/${PROJECT}.json, where PROJECT is the PROJECT name. Use https://raw.githubusercontent.com/zopencommunity/meta/refs/heads/main/docs/api/zopen_releases_latest.json | jq -r '.release_data | keys[]' to get the names of all existing packages. If you think the stable dependency detected from brew is considered optional, don't include it. Find the equivalent name as some names don't match up correctly. If a build dependency doesn't exist and is required, FAIL and tell the user why.
- `dev_deps`: Development dependencies. Typically this is the same as the stable_deps.
- `force`: true to overwrite existing project. 

**Optional Parameters:**
- `runtime_deps`: Runtime dependencies

```json
Example call:
{
  "name": "openssl",
  "description": "OpenSSL is a robust, commercial-grade toolkit for TLS/SSL protocols",
  "categories": "security networking",
  "license": "Apache-2.0",
  "type": "BUILD",
  "build_system": "GNU Make",
  "stable_url": "https://github.com/openssl/openssl.git",
  "build_line": "stable"
  "stable_deps": "make autoconf"
}
```

### Step 3: Build the Project

Use the `zopen_build` tool to compile the project.

**Parameters:**
- `directory`: The path to the generated project directory (typically `<name>port`)
- `verbose`: true for detailed build output

```json
Example call:
{
  "directory": "./opensslport",
  "verbose": true
}
```

**Expected Outcomes:**
- ✅ **Success**: Build completes without errors
- ❌ **Failure**: Build fails with error messages

### Step 4: Handle Build Failures

If the build fails, analyze the error messages.

If zopen build gets past the initial, it will create a directory containing the project source code.

Apply changes to this source code directly. Do not create patches in the patches/ directory until after a successful build.

#### Common z/OS Build Issues:

1. **Missing Configure Script**
   - **Symptom**: "configure: not found" or similar
   - **Solution**: Check the project source for a configure.ac script. If configure.ac exists, you need to set ZOPEN_BOOTSTRAP="./autogen.sh", which when run will create the configure script. 
    If none exists, then set ZOPEN_CONFIGURE="skip" to skip this phase.

2. **Missing Dependencies**
   - **Symptom**: "library not found" or "header not found" or "fatal error: file not found"
   - **Solution**: Determine the underlying library that provides the dependencies and check if it is a dependency in buildenv file. 
If it's a c runtime library missing header or function, add it to the zoslib library. 

3. **EBCDIC/ASCII Issues**
   - **Symptom**: Character encoding errors
   - **Solution**: May require source code changes such as tagging of file descriptors to enable auto-conversion.

4. **Platform-Specific Code**
   - **Symptom**: References to unsupported syscalls or APIs
   - **Solution**: Modify source. Apply workaround solution.

5. **Build System Issues**
   - **Symptom**: Make/CMake errors
   - **Solution**: Customize build flags in buildenv. Use the `zopen_build_help` tool to find out the flags available.

The platform macro for z/OS is __MVS__. You can use this to guard new changes or guard out code to work around issues.

#### Iteration Process:

1. Read the build error log carefully
2. Identify the root cause
3. Apply fixes (update buildenv, modify source, add dependencies)
4. Re-run `zopen_build`
5. Repeat until successful
6. After a successful build, create the patch in the patches dir from the modified source using git diff HEAD > ../patches/PR1.patch

### Step 5: Post-Build Configuration

After a successful build, perform these configuration tasks:

#### 5.1 Verify Installation

1. Use `zopen_info` to check package details
2. Test the built binaries
3. Verify dependencies are correctly listed

#### 5.2 Verify Bump Line Configuration

Check the bump line at the top of the buildenv file to ensure version tracking is properly configured. This enables automatic version checking.

**Action:** Verify the bump line matches your project's version source (tarball URL or git repository). See **Step 6.2** for detailed bump line configuration.

#### 5.3 Update zopen_append_to_env for Libraries/Headers

If the project provides libraries or header files that other ports may depend on, you **must** uncomment and update the `zopen_append_to_env` function in the buildenv file.

**Check if your project provides:**
- Libraries (`.so`, `.a` files) in a `lib/` directory
- Header files (`.h`, `.hpp` files) in an `include/` directory

**If yes, update the buildenv file:**

1. Locate the commented `zopen_append_to_env` function in buildenv
2. Uncomment it and update the library name:

```bash
# For libraries, this hook exports variables for other ports to use.
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

**Replace `<library_name>`** with the actual library name (without `lib` prefix or file extension).

**Example:** If the library is `libcurl.so`, use `-lcurl`:
```bash
export ZOPEN_EXTRA_LIBS="\${ZOPEN_EXTRA_LIBS} -lcurl"
```

**When to skip:** If the project only provides executables/binaries (no libraries), leave this function commented out.

### Step 6: Update buildenv with Version Information

After a successful build:

#### 6.1 Update VRM (Version-Release-Modification)

Update the buildenv file with the correct version information.

#### 6.2 Verify and Update Bump Line

The bump line at the top of the buildenv file enables automatic version checking using the [bump tool](https://github.com/wader/bump). Verify and update it to match your project's version tracking.

**Finding Version Check Regex from Homebrew:**

You can leverage the livecheck regex from Homebrew formulas:

1. Get the `ruby_source_path` from the brew JSON (e.g., `https://formulae.brew.sh/api/formula/${PROJECT}.json`)
2. Construct the formula URL: `https://raw.githubusercontent.com/Homebrew/homebrew-core/refs/heads/main/{ruby_source_path}`
3. Fetch the formula file and look for the `livecheck` block, which contains the version regex
4. Adapt the regex for use with bump

**Example:**
- Brew JSON: `https://formulae.brew.sh/api/formula/midnight-commander.json`
- Field: `"ruby_source_path": "Formula/m/midnight-commander.rb"`
- Formula URL: `https://raw.githubusercontent.com/Homebrew/homebrew-core/refs/heads/main/Formula/m/midnight-commander.rb`
- Fetch this file and look for the livecheck block, which contains the version detection regex that can be adapted for bump

**Bump Line Format:**

For **tarball URLs**:
```bash
# bump: <package>-version /<VAR_NAME>="(.*)"/ <stable_url_with_version>|semver:*
# <VAR_NAME>="V.R.M"
```

For **git repositories**:
```bash
# bump: <package>-version /<VAR_NAME>="(.*)"/ git:<repo_url>|semver:*
# <VAR_NAME>="V.R.M"
```

**Example for HTML directory listing with capture group:**
```bash
# bump: bash-version /BASH_VERSION="(.*)"/ https://ftp.gnu.org/gnu/bash/|re:/href="bash-([\d.]+).tar.gz"/$1/|semver:*
BASH_VERSION="5.3"
```

**Example for HTML directory listing without capture group:**
```bash
# bump: c3270-version /C3270_VERSION="(.*)"/ https://sourceforge.net/projects/x3270/files/x3270/|re:/Click.to.enter.([\d.]+g?a?\d+)"/
C3270_VERSION="4.4ga6"
```

**Example for git repository:**
```bash
# bump: git-version /GIT_VERSION="(.*)"/ https://github.com/git/git.git|*
GIT_VERSION="2.51.0"
```

**Important Steps After Updating Bump Line:**

1. **Update ZOPEN_STABLE_URL or ZOPEN_STABLE_TAG to use the version variable:**
   ```bash
   # For tarball URLs:
   export ZOPEN_STABLE_URL="https://example.com/package-${PACKAGE_VERSION}.tar.gz"

   # For git repositories:
   export ZOPEN_STABLE_TAG="v${PACKAGE_VERSION}"
   ```

2. **Update zopen_get_version() function:**
   ```bash
   zopen_get_version() {
     echo "$PACKAGE_VERSION"  # Use your version variable name
   }
   ```

3. **Verify bump works:**
   ```bash
   bump current buildenv  # Show current version (should match your PACKAGE_VERSION)
   bump check buildenv    # Check for newer versions
   ```

**Bump Line Syntax Notes:**
- The version variable name must match the regex pattern (e.g., `BASH_VERSION` matches `/BASH_VERSION="(.*)"/`)
- For HTML directory listings: Use `|re:/regex/` to extract version from HTML
  - The regex should match the content in the HTML
  - Use `[\d.]+` to match standard versions, add custom patterns like `g?a?\d+` for suffixes
  - Can use `/$1/` capture group syntax (e.g., bash example) or extract match automatically (e.g., c3270 example)
  - Pattern: `|re:/href="package-([\d.]+).tar.gz"/$1/|semver:*` or `|re:/Click.to.enter.([\d.]+)"/`
- For git repositories: Use `git.git|*` format (the `|*` matches all tags)
- Always verify with `bump current buildenv` then `bump check buildenv` after updating

#### 6.3 Update .gitignore to exclude source directories:

   After running `zopen build`, source directories are created that should not be committed to the repository. Update the `.gitignore` file in the port directory to exclude them.

   **Check what directory was created:**
   - Run `ls` or `git status` to see the actual directory name
   - It will be in the format `<package-name>-<version>/` (e.g., `xorgproto-2024.1/`)

   **Add the pattern to .gitignore:**
   ```
   # Ignore source directories created by zopen build
   <package-name>-*/
   ```

   **Example for xorgproto:**
   ```
   # Ignore source directories created by zopen build
   xorgproto-*/
   ```

   This wildcard pattern will match the current version and any future versions when you update the port.

### Step 7: Create Repository (Optional)

Upon successful build and verification, ask the user if they want to create a GitHub repository for the port.

If yes, use the `zopen_create_repo` tool:

```json
{
  "name": "curl",
  "description": "Command line tool for transferring data with URLs",
  "user": "username"
}
```

**Parameters:**
- `name` (required): Port name without 'port' suffix (e.g., curl, openssl)
- `description` (optional): Repository description (default: 'zopen port of <name>')
- `user` (optional): GitHub username to assign as admin

**Note**: This tool is only for core contributors with admin permissions in the zopencommunity organization. GitHub token must be set via GITHUB_TOKEN environment variable.

#### Setting Up Git Remote for the Port Repository

After creating the repository, when you're ready to commit and push the port project to the zopencommunity organization:

**IMPORTANT: Use SSH protocol (git@) instead of https:// for the zopen community port repository.**

```bash
# Initialize git in the port directory if not already done
cd <name>port
git init

# Add the remote using SSH protocol (NOT https://)
git remote add origin git@github.com:zopencommunity/<name>port.git

# Commit and push changes
git add .
git commit -m "Initial port of <name>"
git push origin main
```

**Why SSH protocol?** The SSH protocol (git@github.com:) is required for pushing to zopencommunity repositories from the z/OS environment.

**Note:** Upstream source repositories (stable_url parameter) should continue to use https:// protocol.

### Step 8: Create CI/CD Job (Optional)

After creating the repository, ask the user if they want to create a Jenkins CI/CD job.

If yes, use the `zopen_create_cicd_job` tool:

```json
{
  "name": "curl",
  "build_type": "stable",
  "script_name": "cicd-stable.groovy",
  "run_after": "yes"
}
```

**Parameters:**
- `name` (required): Port name without 'port' suffix (e.g., curl, openssl)
- `build_type` (optional): "stable" or "dev" (default: stable)
- `script_name` (optional): Groovy script path in repo (default: cicd-stable.groovy)
- `run_after` (optional): "yes" or "no" to trigger job after creation (default: yes)

### Step 9: Document Changes

Keep track of:
- Any source code modifications
- buildenv customizations
- Dependencies added
- Known issues or limitations
Create a README.md file in the patches directory

## Helper Tools

### Query Package Information

- Download and inspect https://raw.githubusercontent.com/zopencommunity/meta/refs/heads/main/docs/api/zopen_releases_latest.json | jq -r '.release_data | keys[]' for all zopen available packages
- `zopen_info`: Detailed information about a package

### Environment

- `zopen_version`: Check zopen version

## Build Analysis

If the user asks to analyze a problem for a project, skip directly to build analysis:

1. Look in the `log.STABLE` or `log.DEV` directories for the latest log files
2. Analyze the error messages in the logs
3. Identify the root cause of the failure
4. Suggest fixes based on the analysis

## Debugging on z/OS

z/OS does not have a comprehensive debugger like GDB. Use these techniques:

1. **fprintf statements**: Inject `fprintf(stderr, "Debug: variable=%d\n", var);` statements to trace execution
2. **__display_backtrace**: Use z/OS-specific backtrace function to emit stack traces:
   ```c
   #ifdef __MVS__
   __display_backtrace(2);  // Emits stacktrace to screen
   #endif
   ```

## Best Practices

1. **Always query valid options first**: Use the list tools to ensure you're using valid licenses, categories, and build systems

2. **Start with minimal configuration**: Begin with just the required parameters, add complexity as needed

3. **Use verbose mode**: When debugging build issues, always use `verbose: true`

4. **Check existing ports**: Use `zopen_query` to see if similar packages exist for reference

5. **Document dependencies**: Clearly list all runtime and build dependencies

6. **Test incrementally**: After each fix, rebuild to verify the change works

7. **Follow z/OS conventions**: Use lowercase package names, follow existing patterns

## Example: Complete Porting Session

```
1. Query valid options:
   - zopen_generate_list_licenses
   - zopen_generate_list_categories
   - zopen_generate_list_build_systems

2. Generate project:
   zopen_generate({
     "name": "curl",
     "description": "Command line tool for transferring data with URLs",
     "categories": "networking development",
     "license": "MIT",
     "type": "BUILD",
     "build_system": "GNU Make",
     "stable_url": "https://github.com/curl/curl.git",
     "build_line": "stable"
   })

3. Build project:
   zopen_build({
     "directory": "./curlport",
     "verbose": true
   })

4. If build fails, analyze and fix:
   - Read error messages
   - Update buildenv or source code
   - Rebuild

5. Verify:
   zopen_info({"package": "curl"})
```

## Troubleshooting

**Q: "zopen-generate not found in PATH"**
- Ensure zopen is properly installed and in PATH
- Check zopen installation: `zopen_version`

**Q: Build hangs or takes too long**
- Check build logs for infinite loops
- Verify configure script is correct

**Q: "Directory does not exist" error**
- Verify the project was generated successfully
- Use correct path to the port directory (usually `<name>port`)

**Q: Dependencies not found**
- Download and inspect https://raw.githubusercontent.com/zopencommunity/meta/refs/heads/main/docs/api/zopen_releases_latest.json | jq -r '.release_data | keys[]' for all zopen available packages
- Add to stable_deps 

**Q: Project information is not found through brew. What should I do**
- Do a web search for the project. But always check brew first.

**Q. The project requires additional compiler options or macros to pass**
- Macros can be passed with the addition of ZOPEN_EXTRA_CPPFLAGS="-DLOCALEDIR=NULL" in the buildenv, where in this case LOCALEDIR macro is set to NULL
- Additional compiler options can be passed via the addition of ZOPEN_EXTRA_CFLAGS or ZOPEN_EXTRA_CXXFLAGS or both


## Additional Resources

- Use `zopen_generate_help` for detailed parameter information

## Summary

The key to successful porting:
1. ✅ Gather accurate project information. Use the tools provided. Do not search the web.
2. ✅ Use valid metadata (query the list tools)
3. ✅ Generate the project structure
4. ✅ Build and iterate on failures
5. ✅ Test thoroughly

Follow this workflow systematically, and you'll be able to port most open-source software to z/OS efficiently.
