#!/bin/bash

setsid sleep 5 && /travels-api >/dev/null 2>&1 < /dev/null &

/entrypoint.sh mysqld