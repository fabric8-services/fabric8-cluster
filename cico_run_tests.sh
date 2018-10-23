#!/bin/bash

. cico_setup.sh

cico_setup;

generate_client_setup fabric8-services fabric8-cluster-client;

run_tests_without_coverage;
