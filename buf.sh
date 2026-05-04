#!/usr/bin/env bash

##############################################################
# Safeguards
##############################################################

set -eo pipefail

##############################################################
# Functions declarations
##############################################################

function Require {
    command -v $1 > /dev/null 2>&1 || {
        echo "Some of the required software is not installed: $1"
        exit 1;
    }
}

function Buf {
    local PLATFORMS="linux/amd64"
    local IMAGE_TAG="bufbuild:local"
    local OUT_DIR="lib/schema/gen"
    local BUILD_CTX
    local MOUNTS

    BUILD_CTX="$(pwd)"/tools/buf/

    # make some preparation steps based on command

    case "${1}" in
      "generate")
          # Parse --template flag to mount only necessary *.buf.gen.yaml file to the container
          # instead of hardcoding possible variants.
          #
          # see: https://buf.build/docs/configuration/v2/buf-gen-yaml/

          for arg in "$@"; do
            local KEY="${arg%%=*}"
            local VALUE="${arg#*=}"
            KEY="${KEY#--}"

            if [[ "${KEY}" == "template" ]]; then
              MOUNTS+="--mount type=bind,source=$(pwd)/${VALUE},target=/workspace/${VALUE},readonly "
            fi
          done
        ;;
      "format")
          # Mount folder with service's protobuf files to format 'em
          # see: https://buf.build/docs/reference/cli/buf/format/

          MOUNTS+="--mount type=bind,source=$(pwd)/${2},target=/workspace/${2} "
        ;;
      "lint")
        ;;
      *)
        echo "Unknown argument: ${1}."
        exit 3
        ;;
    esac

    echo "Building buf.build docker image ..."
    docker buildx build --quiet --load --platform ${PLATFORMS} -t ${IMAGE_TAG} ${BUILD_CTX}

    echo "Running command in buf.build container ..."

    # To avoid issues with permissions for generated files,
    # we don't mount the output folder to container, but instead
    # copy the generated protos from the container to output directory.

    CONTAINER_NAME="bufbuild-$(date +%s)"

    # We're mounting "buf.yaml" file that located in the root of the repo,
    # because of v2 configs usage, if you're still using v1 - it's not needed
    #
    # see: https://buf.build/docs/migration-guides/migrate-v2-config-files/

    docker run \
      --name "${CONTAINER_NAME}" \
      -v "$(pwd)/proto/":/workspace/proto/:ro \
      --mount type=bind,source="$(pwd)"/buf.yaml,target=/workspace/buf.yaml,readonly \
      ${MOUNTS} \
      ${IMAGE_TAG} "$@"

    CONTAINER_ID=$(docker inspect -f '{{.Id}}' ${CONTAINER_NAME})

    # run additional steps based on command

    case "${1}" in
      "generate")
        mkdir -p ${OUT_DIR}
        docker cp ${CONTAINER_ID}:/workspace/lib/schema/gen/. ${OUT_DIR}
        ;;
      "format")
        # gventsadze.k:
        #  we're copying buf cli behaviour which expects that path to the directory with protobuf files
        #  will be passed as next argument after command, therefore we're taking 2nd argument
        #  without any additional tricks.
        docker cp "${CONTAINER_ID}":/workspace/${2}/. ${2}
        ;;
      "lint")
        ;;
    esac

    # cleanup

    docker rm -f "${CONTAINER_NAME}" &> /dev/null || true
}

##############################################################
# Initialize and validate requirements
##############################################################

SCRIPT_DIR="$(dirname ${BASH_SOURCE[0]})";

Require docker

##############################################################
# Build protocol buffers
##############################################################

cd ${SCRIPT_DIR}

Buf "$@"

echo "Done!"
