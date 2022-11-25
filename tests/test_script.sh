#!/bin/bash


test_input () {
  echo pong $1
  read -r command value
  echo end $value
  exit 0
}

test_env () {
  env | grep DEPLOYER_PODMAN
  exit 0
}

test_volume () {
  cat /test/test_file.txt
  exit 0
}

test_sleep () {
  /usr/bin/sleep $1
  exit 0
}

echo Enter a test and a parameter:
read -r action value
case $action in
  ping)
    test_input $value
    ;;
  env)
    test_env
    ;;
  volume)
    test_volume
    ;;
  sleep)
    test_sleep $value
    ;;
  *)
    echo "no valid input provided, exiting"
    exit 1
    ;;
esac
