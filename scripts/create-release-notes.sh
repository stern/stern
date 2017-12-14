#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

cat << EOF
## Changelog

TODO

## Checksums

- stern_linux_amd64:
  - sha256: \`$(awk '{print $1}' "$DIR/../artifacts/latest/linux_amd64/SHA256SUMS")\`
- stern_darwin_amd64:
  - sha256: \`$(awk '{print $1}' "$DIR/../artifacts/latest/darwin_amd64/SHA256SUMS")\`
- stern_windows_amd64.exe:
  - sha256: \`$(awk '{print $1}' "$DIR/../artifacts/latest/windows_amd64/SHA256SUMS")\`

EOF

