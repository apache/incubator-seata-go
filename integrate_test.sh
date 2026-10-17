#!/usr/bin/env bash

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

set -euo pipefail

ROOT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
CI_LOG_DIR="${CI_LOG_DIR:-${ROOT_DIR}/ci-logs}"
mkdir -p "${CI_LOG_DIR}"
CI_LOG_DIR=$(cd "${CI_LOG_DIR}" && pwd)
export CI_LOG_DIR
exec > >(tee "${CI_LOG_DIR}/integrate.log") 2>&1

echo "integrate-test source -> ${1:-local} ${2:-local} ${3:-local}"
SAMPLES_REF="${SAMPLES_REF:-8c7214f476f83a16622fd2c5911aa8d85bdc6ee0}"
echo "seata-go-samples requested revision -> ${SAMPLES_REF}"

SEATA_COMPOSE_BIN=$(command -v docker-compose || true)
export SEATA_COMPOSE_BIN
if [[ -z "${SEATA_COMPOSE_BIN}" ]] && ! docker compose version; then
    echo "Docker Compose is required for integration tests" >&2
    exit 1
fi

SAMPLES_DIR=$(mktemp -d "${ROOT_DIR}/.seata-go-samples.XXXXXX")
cleanup_samples() {
    local result=$?
    trap - EXIT
    set +e
    printf '%s\n' "${result}" > "${CI_LOG_DIR}/exit-code.txt"
    docker ps -a > "${CI_LOG_DIR}/docker-ps.txt" 2>&1
    mkdir -p "${CI_LOG_DIR}/samples"
    (cd "${SAMPLES_DIR}" && find . -type f -name '*.log' -exec cp --parents '{}' "${CI_LOG_DIR}/samples/" \;)
    for compose_file in dockercompose/docker-compose.yml saga/e2e/docker-compose.yml; do
        if [[ -f "${SAMPLES_DIR}/${compose_file}" ]]; then
            docker-compose -f "${SAMPLES_DIR}/${compose_file}" down -v --remove-orphans
        fi
    done
    rm -rf "${SAMPLES_DIR}"
    exit "${result}"
}
trap cleanup_samples EXIT

git clone https://github.com/apache/incubator-seata-go-samples "${SAMPLES_DIR}"
cd "${SAMPLES_DIR}"
git fetch --depth 1 origin "${SAMPLES_REF}"
git checkout --detach FETCH_HEAD
git rev-parse HEAD | tee "${CI_LOG_DIR}/samples-revision.txt"

# Samples tear down their own services; capture logs before every Compose down.
mkdir -p .ci-bin
cat > .ci-bin/docker-compose <<'COMPOSE'
#!/usr/bin/env bash
compose=(docker compose)
if [[ -n "${SEATA_COMPOSE_BIN:-}" ]]; then
    compose=("${SEATA_COMPOSE_BIN}")
fi
prefix=()
for arg in "$@"; do
    if [[ "$arg" == down ]]; then
        {
            printf '\n=== Before Compose down: %s ===\n' "${prefix[*]}"
            "${compose[@]}" "${prefix[@]}" ps -a
            "${compose[@]}" "${prefix[@]}" logs --no-color
        } >> "${CI_LOG_DIR}/compose.log" 2>&1 || true
        break
    fi
    prefix+=("$arg")
done
exec "${compose[@]}" "$@"
COMPOSE
chmod +x .ci-bin/docker-compose
export PATH="${SAMPLES_DIR}/.ci-bin:$PATH"

adapt_samples_to_seata_go_v2() {
    find . -type f -name '*.go' -print0 | while IFS= read -r -d '' file; do
        perl -pi -e 's#seata\.apache\.org/seata-go/pkg#seata.apache.org/seata-go/v2/pkg#g' "$file"
    done

    go mod edit -droprequire=seata.apache.org/seata-go || true
    go mod edit -require=seata.apache.org/seata-go/v2@v2.0.0

    if grep -R -n --exclude-dir=.git --include='*.go' 'seata\.apache\.org/seata-go/pkg' .; then
        echo "old seata-go v1 imports remain after v2 migration"
        exit 1
    fi
}

# Compile and run the pinned samples against this checkout.
adapt_samples_to_seata_go_v2
go mod edit -replace=seata.apache.org/seata-go/v2="${ROOT_DIR}"
go mod tidy

./start_integrate_test.sh
WAIT_READY_MARGIN="${WAIT_READY_MARGIN:-30}" ./saga/e2e/run_all.sh --up --seata saga/e2e/seatago.yaml --engine saga/e2e/config.yaml
