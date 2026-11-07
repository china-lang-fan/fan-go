#!/bin/sh
set -e

OWNER="china-lang-fan"
REPO="fan-go"
BINARY="fan"

usage() {
  cat <<EOF
用法: install.sh [-v 版本] [-d 安装目录]
  -v  指定版本（默认最新正式版，例如 0.1.0）
  -d  安装目录（默认 /usr/local/bin）
EOF
}

VERSION=""
BINDIR="/usr/local/bin"

while getopts "v:d:h" opt; do
  case "$opt" in
    v) VERSION="$OPTARG" ;;
    d) BINDIR="$OPTARG" ;;
    h) usage; exit 0 ;;
    *) usage; exit 1 ;;
  esac
done

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "错误：未找到命令 $1" >&2
    exit 1
  fi
}

need_cmd uname
need_cmd curl
need_cmd tar
need_cmd mktemp

OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
  Linux) OS="linux" ;;
  Darwin) OS="darwin" ;;
  *) echo "错误：不支持的系统 $OS" >&2; exit 1 ;;
esac

case "$ARCH" in
  x86_64 | amd64) ARCH="amd64" ;;
  arm64 | aarch64) ARCH="arm64" ;;
  *) echo "错误：不支持的架构 $ARCH" >&2; exit 1 ;;
esac

if [ -z "$VERSION" ]; then
  if command -v curl >/dev/null 2>&1; then
    TAG="$(curl -fsSL "https://api.github.com/repos/$OWNER/$REPO/releases/latest" | sed -n 's/.*"tag_name": *"\(v\{0,1\}[^"]*\)".*/\1/p' | head -n1)"
    if [ -z "$TAG" ]; then
      echo "错误：无法获取最新版本，请用 -v 手动指定" >&2
      exit 1
    fi
    VERSION="${TAG#v}"
  else
    echo "错误：无法获取最新版本，请用 -v 手动指定" >&2
    exit 1
  fi
fi

NAME="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$OWNER/$REPO/releases/download/v${VERSION}/${NAME}"

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

echo "下载 $URL"
curl -fL "$URL" -o "$TMPDIR/$NAME"
tar -xzf "$TMPDIR/$NAME" -C "$TMPDIR"

if [ ! -f "$TMPDIR/$BINARY" ]; then
  echo "错误：压缩包中未找到 $BINARY" >&2
  exit 1
fi

mkdir -p "$BINDIR"
install -m 0755 "$TMPDIR/$BINARY" "$BINDIR/$BINARY"

echo "安装完成：$BINDIR/$BINARY"
"$BINDIR/$BINARY" version
