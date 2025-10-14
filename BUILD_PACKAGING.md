# Building vacuum for Package Managers

This document provides guidance for package maintainers who need to build vacuum with custom version information.

## Version Information

vacuum supports two methods for embedding version information:

### 1. Automatic Version Detection (Default)
When built from a git repository or installed via `go install`, vacuum automatically detects version information using Go's `debug.ReadBuildInfo()`.

### 2. Custom Version via ldflags (For Package Maintainers)
Package maintainers can override version information using ldflags during build time. This is useful when building from source tarballs or when you need to specify your own version scheme.

## Building with Custom Version Information

Use the `-ldflags` option with `go build` to set custom version information:

```bash
# From within the vacuum source directory:
go build -ldflags "-X main.version=<version> -X main.commit=<commit> -X 'main.date=<date>'" \
    -o vacuum

# Or specify the current directory explicitly:
go build -ldflags "-X main.version=<version> -X main.commit=<commit> -X 'main.date=<date>'" \
    -o vacuum \
    .
```

### Parameters

- `main.version`: The version string (e.g., `v0.18.6`, `0.18.6-1`, `0.18.6-nixpkg`)
- `main.commit`: Git commit hash (short or full)
- `main.date`: Build date (any format, but RFC3339 or human-readable preferred)

### Examples

#### Arch Linux (AUR) PKGBUILD
```bash
pkgver="0.18.6"
_commit="a535d6c"
_date=$(date '+%a, %d %b %Y %H:%M:%S %Z')

go build -ldflags "-linkmode=external \
    -X main.version=$pkgver \
    -X main.commit=$_commit \
    -X 'main.date=$_date'" \
    -o vacuum
```

#### Nix Package
```nix
buildGoModule {
  pname = "vacuum";
  version = "0.18.6";

  ldflags = [
    "-X main.version=${version}"
    "-X main.commit=${src.rev or "unknown"}"
    "-X main.date=1970-01-01T00:00:00Z"
  ];
}
```

#### Debian Package
```makefile
VERSION := $(shell dpkg-parsechangelog -S Version)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -R)

build:
	go build -ldflags "-X main.version=$(VERSION) \
		-X main.commit=$(COMMIT) \
		-X 'main.date=$(DATE)'" \
		-o vacuum
```

#### Homebrew Formula
```ruby
def install
  ldflags = %W[
    -X main.version=#{version}
    -X main.commit=#{Utils.git_short_head}
    -X main.date=#{time.iso8601}
  ]

  system "go", "build", *std_go_args(ldflags: ldflags)
end
```

## Testing Your Build

After building with custom ldflags, verify the version information:

```bash
# Check version
./vacuum version
# Expected output: your custom version

# Check full version info in banner
./vacuum lint --help
# The banner will show: version: <your-version> | compiled: <your-date>
```

## Partial ldflags

You can specify only some of the ldflags:

```bash
# Only set version (commit will show "unknown", date will default to current time)
go build -ldflags "-X main.version=v0.18.6" -o vacuum

# Only set version and commit (date will default to current time)
go build -ldflags "-X main.version=v0.18.6 -X main.commit=a535d6c" -o vacuum
```

Note: Unspecified `version` and `commit` values will show as "unknown", while an unspecified `date` will default to the current build time.

## Compatibility Notes

- If ldflags are not provided, vacuum falls back to automatic detection via `debug.ReadBuildInfo()`
- This dual approach ensures compatibility with both `go install` users and package managers
- The ldflags approach takes precedence when provided
- Date strings can be in any format, but RFC3339 format will be automatically reformatted for display

## Integration with CI/CD

For automated builds, you can dynamically generate version information:

```bash
VERSION=$(git describe --tags --always)
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

go build -ldflags "-X main.version=${VERSION} \
    -X main.commit=${COMMIT} \
    -X main.date=${DATE}" \
    -o vacuum
```

## Need Help?

If you encounter issues with version information in your package, please open an issue at:
https://github.com/daveshanley/vacuum/issues