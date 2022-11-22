#!/bin/bash


test_input () {
  echo This is what input was received: \"$1\"
  exit 0
}

test_env () {
  env | grep DEPLOYER_PODMAN
  exit 0
}

echo Enter a test and a parameter:
read -r action value
case $action in
  input)
    test_input $value
    ;;
  env)
    test_env
    ;;
  *)
    echo "no valid input provided, exiting"
    exit 1
    ;;
esac
