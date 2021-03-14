#!/bin/sh

PROJECT_ROOT=$(cd $(dirname ${BASH_SOURCE})/..; pwd)
MOD_ROOT=${GOPATH}/pkg/mod

echo "buf version: $(buf --version 2>&1)"
echo "swagger version: $(swagger version)"

buf beta mod update
buf generate

# collect_swagger gathers swagger files into a subdirectory
collect_swagger() {
    SWAGGER_ROOT="$1"
    EXPECTED_COLLISIONS="$2"
    SWAGGER_OUT="${PROJECT_ROOT}/apis/swagger.json"
    SWAGGER_OUT_ASSETS="${PROJECT_ROOT}/server/assets/swagger.json"
    PRIMARY_SWAGGER=`mktemp`
    COMBINED_SWAGGER=`mktemp`

    cat <<EOF > "${PRIMARY_SWAGGER}"
{
  "swagger": "2.0",
  "info": {
    "title": "${BIN_NAME}",
    "description": "Description of all APIs",
    "version": "${VERSION}"
  },
  "paths": {}
}
EOF

    rm -f "${SWAGGER_OUT}"

    find "${SWAGGER_ROOT}" -name '*.swagger.json' -exec swagger mixin -c "${EXPECTED_COLLISIONS}" "${PRIMARY_SWAGGER}" '{}' \+ > "${COMBINED_SWAGGER}"
    jq -r 'del(.definitions[].properties[]? | select(."$ref"!=null and .description!=null).description) | del(.definitions[].properties[]? | select(."$ref"!=null and .title!=null).title)' "${COMBINED_SWAGGER}" > "${SWAGGER_OUT}"

    cp "${SWAGGER_OUT}" "${SWAGGER_OUT_ASSETS}"
    /bin/rm "${PRIMARY_SWAGGER}" "${COMBINED_SWAGGER}"
}

# clean up generated swagger files (should come after collect_swagger)
clean_swagger() {
    SWAGGER_ROOT="$1"
    find "${SWAGGER_ROOT}" -name '*.swagger.json' -delete
}

# If additional types are added, the number of expected collisions may need to be increased
EXPECTED_COLLISION_COUNT=55
collect_swagger pkg/apis ${EXPECTED_COLLISION_COUNT}
clean_swagger pkg/apis
