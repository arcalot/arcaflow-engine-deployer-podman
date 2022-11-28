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

test_network () {
  case $1 in
    host)
    /usr/sbin/ifconfig
    exit 0
    ;;
    bridge)
    IP_ADDRESS=`/usr/sbin/ifconfig testif0 | /usr/bin/grep inet | /usr/bin/awk '{ print $2 }'`
    MAC=`/usr/sbin/ifconfig testif0 | /usr/bin/grep ether | /usr/bin/awk '{ print $2 }'`
    echo "$IP_ADDRESS;$MAC"
    exit 0
    ;;
    none)
    IFACE_COUNT=`/usr/sbin/ifconfig -a |/usr/bin/grep flags|/usr/bin/wc -l`
    IFACE=`/usr/sbin/ifconfig -a | /usr/bin/grep flags | /usr/bin/awk '{ gsub (/\:/,"");print $1}'`
    echo "$IFACE_COUNT;$IFACE"
    ;;
  esac

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
  network)
    test_network $value
    ;;
  *)
    echo "no valid input provided, exiting"
    exit 1
    ;;
esac
