# Claude Instructions for z/OS Software Porting

This document provides step-by-step instructions to port open-source software to z/OS.

## Overview

You have access to zopen tools through MCP that allow you to:
- Generate zopen-compatible project structures
- Build projects on z/OS
- Query package information
- Install and manage z/OS packages

## Porting to z/OS Workflow

### Step 1: Gather Project Information

Before starting, collect the following information about the project, here on out referred to as ${PROJECT}.

1. **Project Name** (lowercase, no spaces)
2. **Description** (brief, one-sentence summary)
3. **Repository URL** (GitHub or other git repository)
4. **License** (SPDX identifier). Call zopen_generate_list_licenses to see all valid license identifiers
5. **Categories** Call zopen_generate_list_categories to see all valid categories
6. **Build System** (e.g., "GNU Make", "CMake", "Meson") Call zopen_generate_list_build_systems to see all valid build systems

**Action**: Use `zopen_generate_list_licenses`, `zopen_generate_list_categories`, and `zopen_generate_list_build_systems` to get valid options. Use the brew json information to get the additional data such as source location.

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
   - **Solution**: Update the buildenv file to set ZOPEN_CONFIGURE or it's possible that a configure.ac exists and you need to set ZOPEN_BOOTSTRAP="./autogen.sh", which when run will create the configure script.
   - **Example**: `ZOPEN_CONFIGURE="./Configure"`

2. **Missing Dependencies**
   - **Symptom**: "library not found" or "header not found"
   - **Solution**: Add dependencies to stable_deps 
   - **Action**: Use `zopen_query` to find available packages

3. **EBCDIC/ASCII Issues**
   - **Symptom**: Character encoding errors
   - **Solution**: May require source code changes. 

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

### Step 5: Verify Installation

After a successful build, verify the port:

1. Use `zopen_info` to check package details
2. Test the built binaries
3. Verify dependencies are correctly listed

### Step 6: Document Changes

Keep track of:
- Any source code modifications
- buildenv customizations
- Dependencies added
- Known issues or limitations
Create a README.md file in the patches directory

## Helper Tools

### Query Package Information

- `zopen_list`: List all available packages
- `zopen_query`: Get details about specific packages
- `zopen_info`: Detailed information about a package

### Environment

- `zopen_version`: Check zopen version

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
