#!/bin/sh

if [ ! -d "/tmp/check" ]; then
  mkdir /tmp/check
  echo `date` > /tmp/check/first_`date +%Y%m%d_%H%M%S`
  chmod 755 /workspace/.devcontainer/build.sh
  yarn
else
  echo `date` > /tmp/check/second_`date +%Y%m%d_%H%M%S`
fi

exec "$@"