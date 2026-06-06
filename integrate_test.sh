#
#  Licensed to the Apache Software Foundation (ASF) under one or more
#  contributor license agreements.  See the NOTICE file distributed with
#  this work for additional information regarding copyright ownership.
#  The ASF licenses this file to You under the Apache License, Version 2.0
#  (the "License"); you may not use this file except in compliance with
#  the License.  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

#!/bin/bash

set -e
set -x

echo 'start integrate-test'

# set root workspace
ROOT_DIR=$(pwd)
echo "integrate-test root work-space -> ${ROOT_DIR}"

# show all github-env
echo "github current commit id  -> $2"
echo "github pull request branch -> ${GITHUB_REF}"
echo "github pull request slug -> ${GITHUB_REPOSITORY}"
echo "github pull request repo slug -> ${GITHUB_REPOSITORY}"
echo "github pull request actor -> ${GITHUB_ACTOR}"
echo "github pull request repo param -> $1"
echo "github pull request base branch -> $3"
echo "github pull request head branch -> ${GITHUB_HEAD_REF}"

echo "use seata-go-samples $3 branch for integration testing"
SAMPLES_DIR=$(mktemp -d "${ROOT_DIR}/.seata-go-samples.XXXXXX")
cleanup_samples() {
    rm -rf "${SAMPLES_DIR}"
}
trap cleanup_samples EXIT

git clone https://github.com/apache/incubator-seata-go-samples "${SAMPLES_DIR}" && cd "${SAMPLES_DIR}"

adapt_samples_to_seata_go_v2() {
    find . -type f -name '*.go' -print0 | while IFS= read -r -d '' file; do
        perl -pi -e 's#seata\.apache\.org/seata-go/pkg#seata.apache.org/seata-go/v2/pkg#g' "$file"
    done

    go mod edit -droprequire=seata.apache.org/seata-go || true
    go mod edit -require=seata.apache.org/seata-go/v2@v2.0.0
}

cleanup_saga_e2e() {
    docker-compose -f saga/e2e/docker-compose.yml down -v --remove-orphans || true
}

run_saga_e2e_test() {
    set +e
    ./saga/e2e/run_all.sh --up --seata saga/e2e/seatago.yaml --engine saga/e2e/config.yaml
    local result=$?
    set -e

    cleanup_saga_e2e
    return $result
}

# update seata-go to current dir
adapt_samples_to_seata_go_v2
go mod edit -replace=seata.apache.org/seata-go/v2="${ROOT_DIR}"

go mod tidy

# ensure docker-compose is available (newer Docker uses 'docker compose')
if ! command -v docker-compose &> /dev/null; then
    mkdir -p /tmp/docker-shims
    printf '#!/bin/sh\nexec docker compose "$@"\n' > /tmp/docker-shims/docker-compose
    chmod +x /tmp/docker-shims/docker-compose
    export PATH="/tmp/docker-shims:$PATH"
fi

# start integrate test
./start_integrate_test.sh

# start saga e2e test
run_saga_e2e_test
