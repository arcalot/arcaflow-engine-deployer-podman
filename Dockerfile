FROM quay.io/centos/centos:stream8

COPY tests/test_script.sh /

ENTRYPOINT [ "bash", "test_script.sh" ]