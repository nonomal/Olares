#!/bin/bash

kubectl get cm -n os-framework authelia-configs -o jsonpath='{.data.configuration\.yaml}' > /tmp/authelia-config.yaml
