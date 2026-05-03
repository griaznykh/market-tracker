#!/usr/bin/env bash

##############################################################
# Safeguards
##############################################################

set -eo pipefail

##############################################################
# Init databases
##############################################################

psql -v ON_ERROR_STOP=1 --username "${POSTGRES_USER}" --dbname "postgres" <<-EOSQL
    CREATE DATABASE ${CORE_DB_NAME};
    GRANT ALL PRIVILEGES ON DATABASE ${CORE_DB_NAME} TO ${POSTGRES_USER};
EOSQL
