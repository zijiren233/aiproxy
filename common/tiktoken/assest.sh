#!/bin/bash

set -ex

ASSETS=$(
    cat <<EOF
https://openaipublic.blob.core.windows.net/encodings/o200k_base.tiktoken
https://openaipublic.blob.core.windows.net/encodings/cl100k_base.tiktoken
https://openaipublic.blob.core.windows.net/encodings/p50k_base.tiktoken
https://openaipublic.blob.core.windows.net/encodings/r50k_base.tiktoken
EOF
)

mkdir -p "$(dirname $0)/assets"

rm -f "$(dirname $0)/assets/*"

for asset in $ASSETS; do
    curl -L -f -o "$(dirname $0)/assets/$(basename $asset)" "$asset"
done
