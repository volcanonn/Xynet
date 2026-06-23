# Maintainer: volcanonn <50715952+volcanonn@users.noreply.github.com>
pkgname=xynet
pkgver=0.1.0
pkgrel=1
pkgdesc="Visual node-based VPN and proxy manager with split-tunneling"
arch=('x86_64')
url="https://github.com/volcanonn/Xynet"
license=('GPL-3.0-or-later')
depends=('sing-box' 'webkit2gtk-4.1' 'gtk3')
optdepends=(
    'dae: eBPF kernel-level routing backend (Linux only)'
    'vopono: Strict mode network namespace isolation'
)
makedepends=('go' 'wails' 'deno')
source=("$pkgname-$pkgver.tar.gz::$url/archive/v$pkgver.tar.gz")
sha256sums=('SKIP')

prepare() {
    cd "Xynet-$pkgver"
    export GOPATH="$srcdir/gopath"
    go mod download
}

build() {
    cd "Xynet-$pkgver"
    export GOPATH="$srcdir/gopath"
    export CGO_CPPFLAGS="${CPPFLAGS}"
    export CGO_CFLAGS="${CFLAGS}"
    export CGO_CXXFLAGS="${CXXFLAGS}"
    export CGO_LDFLAGS="${LDFLAGS}"
    export GOFLAGS="-buildmode=pie -trimpath -ldflags=-linkmode=external -mod=readonly -modcacherw"

    wails build -tags webkit2_41

    go clean -modcache
}

package() {
    cd "Xynet-$pkgver"

    install -Dm755 "build/bin/Xynet" "$pkgdir/usr/bin/xynet"
    install -Dm644 "xynet.desktop" "$pkgdir/usr/share/applications/xynet.desktop"
    install -Dm644 "build/appicon.png" "$pkgdir/usr/share/icons/hicolor/256x256/apps/xynet.png"
    install -Dm644 "org.xynet.pkexec.sing-box.policy" "$pkgdir/usr/share/polkit-1/actions/org.xynet.pkexec.sing-box.policy"
}
