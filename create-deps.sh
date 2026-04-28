#!/bin/bash
for i in $(seq 1 50); do
NAME=$(printf "pg-test-%02d" "$i")
./build/_output/orch-cli create deployment-package "$NAME" "1.0.0" --application-reference nfd:0.18.1 --kind normal --default-profile-name deployment-profile-1 || break
done
