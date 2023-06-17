#!/bin/sh
set -e

# Adapted/Copied from https://raw.githubusercontent.com/railwayapp/cli/master/install.sh
#
# vacuum
# https://quobix.com/vacuum/start
#
# Designed for quick installs over the network and CI/CD
#   sh -c "$(curl -sSL https://github.com/daveshanley/vacuum/blob/main/bin/install.sh)"

INSTALL_DIR=${INSTALL_DIR:-"/usr/local/bin"}
BINARY_NAME=${BINARY_NAME:-"vacuum"}

REPO_NAME="daveshanley/vacuum"
ISSUE_URL="https://github.com/daveshanley/vacuum/issues/new"

# get_latest_release "daveshanley/vacuum"
get_latest_release() {
  curl --retry 5 --silent "https://api.github.com/repos/$1/releases/latest" | # Get latest release from GitHub api
    grep '"tag_name":' |                                            # Get tag line
    sed -E 's/.*"([^"]+)".*/\1/'                                    # Pluck JSON value
}

get_asset_name() {
  echo "vacuum_$1_$2_$3.tar.gz"
}

get_download_url() {
  local asset_name=$(get_asset_name $1 $2 $3)
  echo "https://github.com/daveshanley/vacuum/releases/download/v$1/${asset_name}"
}

get_checksum_url() {
  echo "https://github.com/daveshanley/vacuum/releases/download/v$1/checksums.txt"
}

# https://github.com/daveshanley/vacuum/releases/download/v0.0.15/vacuum_0.0.15_darwin_x86_64.tar.gz

command_exists() {
  command -v "$@" >/dev/null 2>&1
}

fmt_error() {
  echo ${RED}"Error: $@"${RESET} >&2
}

fmt_warning() {
  echo ${YELLOW}"Warning: $@"${RESET} >&2
}

fmt_underline() {
  echo "$(printf '\033[4m')$@$(printf '\033[24m')"
}

fmt_code() {
  echo "\`$(printf '\033[38;5;247m')$@${RESET}\`"
}

setup_color() {
  # Only use colors if connected to a terminal
  if [ -t 1 ]; then
    RED=$(printf '\033[31m')
    GREEN=$(printf '\033[32m')
    YELLOW=$(printf '\033[33m')
    BLUE=$(printf '\033[34m')
    MAGENTA=$(printf '\033[35m')
    BOLD=$(printf '\033[1m')
    RESET=$(printf '\033[m')
  else
    RED=""
    GREEN=""
    YELLOW=""
    BLUE=""
    MAGENTA=""
    BOLD=""
    RESET=""
  fi
}

get_os() {
  case "$(uname -s)" in
    *linux* ) echo "linux" ;;
    *Linux* ) echo "linux" ;;
    *darwin* ) echo "darwin" ;;
    *Darwin* ) echo "darwin" ;;
  esac
}

get_machine() {
  case "$(uname -m)" in
    "x86_64"|"amd64"|"x64")
      echo "x86_64" ;;
    "i386"|"i86pc"|"x86"|"i686")
      echo "i386" ;;
    "arm64"|"armv6l"|"aarch64")
      echo "arm64"
  esac
}

get_tmp_dir() {
  echo $(mktemp -d)
}

do_checksum() {
  checksum_url=$(get_checksum_url $version)
  get_checksum_url $version
  expected_checksum=$(curl -sL $checksum_url | grep $asset_name | awk '{print $1}')



  if command_exists sha256sum; then
    checksum=$(sha256sum $asset_name | awk '{print $1}')
  elif command_exists shasum; then
    checksum=$(shasum -a 256 $asset_name | awk '{print $1}')
  else
    fmt_warning "Could not find a checksum program. Install shasum or sha256sum to validate checksum."
    return 0
  fi

  if [ "$checksum" != "$expected_checksum" ]; then
    fmt_error "Checksums do not match"
    exit 1
  fi
}

do_install_binary() {
  asset_name=$(get_asset_name $version $os $machine)
  download_url=$(get_download_url $version $os $machine)

  command_exists curl || {
    fmt_error "curl is not installed"
    exit 1
  }

  command_exists tar || {
    fmt_error "tar is not installed"
    exit 1
  }

  local tmp_dir=$(get_tmp_dir)

  # Download tar.gz to tmp directory
  echo "Downloading $download_url"
  (cd $tmp_dir && curl -sL -O "$download_url")

  (cd $tmp_dir && do_checksum)

  # Extract download
  (cd $tmp_dir && tar -xzf "$asset_name")

  # Install binary
  mv "$tmp_dir/$BINARY_NAME" $INSTALL_DIR
  echo "Installed vacuum to $INSTALL_DIR"

  # Cleanup
  rm -rf $tmp_dir
}

install_termux() {
  echo "Installing vacuum, this may take a few minutes..."
  pkg upgrade && pkg install golang git -y && git clone https://github.com/daveshanley/vacuum.git && cd cli/ && go build -o $PREFIX/bin/vacuum
}

main() {
  setup_color

  latest_tag=$(get_latest_release $REPO_NAME)
  latest_version=$(echo $latest_tag | sed 's/v//')
  version=${VERSION:-$latest_version}

  os=$(get_os)
  if test -z "$os"; then
    fmt_error "$(uname -s) os type is not supported"
    echo "Please create an issue so we can add support. $ISSUE_URL"
    exit 1
  fi

  machine=$(get_machine)
  if test -z "$machine"; then
    fmt_error "$(uname -m) machine type is not supported"
    echo "Please create an issue so we can add support. $ISSUE_URL"
    exit 1
  fi
  if [ ${TERMUX_VERSION} ] ; then
    install_termux
  else
    do_install_binary
  fi

  printf "$MAGENTA"
  cat <<'EOF'
                   .
         /^\     .
    /\   "V"
   /__\   I      O  o
  //..\\  I     .                            Poof!
  \].`[/  I
  /l\/j\  (]    .  O
 /. ~~ ,\/I          .               vacuum is now installed
 \\L__j^\/I       o              Run `vacuum help` for commands
  \/--v}  I     o   .
  |    |  I   _________
  |    |  I c(`       ')o
  |    l  I   \.     ,/
_/j  L l\_!  _//^---^\\_

EOF
  printf "$RESET"

}

main

