#!/usr/bin/env bash
# Pinned independent SRV reader for CI; no runtime application dependency.
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p build/walls-consumer
cd build/walls-consumer
curl --fail --location --silent --show-error https://survex.com/software/1.4.22/survex-1.4.22.tar.gz -o survex.tar.gz
printf '%s\n' 'a26962d888621cfd2043a7cc6a227fc905130dfbd14c450dc1c4c02fbfdd9dbc  survex.tar.gz' | sha256sum -c -
tar -xzf survex.tar.gz
cd survex-1.4.22
./configure
make -C src -j2 cavern dump3d
./src/cavern --version
