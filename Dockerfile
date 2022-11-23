FROM quay.io/centos/centos:stream8

COPY test/test_script.sh /

ENTRYPOINT [ "bash", "test_script.sh" ]