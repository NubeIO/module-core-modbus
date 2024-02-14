#/bin/bash

MODULE_NAME="module-core-modbus"
BIOS_CONTAINER="bios"

docker build -t module-builder -f Dockerfile.module --build-arg="GITHUB_TOKEN=$GITHUB_TOKEN" --build-arg="MODULE_NAME=$MODULE_NAME" .
docker run -d --name module-builder module-builder:latest
docker container cp module-builder:/app/$MODULE_NAME .
docker rm -f module-builder
#docker exec $BIOS_CONTAINER sh -c 'rm -rf /data/rubix-os/data/modules/'$MODULE_NAME'/v0.0.0/; mkdir -p /data/rubix-os/data/modules/'$MODULE_NAME'/v0.0.0/'
#docker cp ./$MODULE_NAME $BIOS_CONTAINER:/data/rubix-os/data/modules/$MODULE_NAME/v0.0.0/
