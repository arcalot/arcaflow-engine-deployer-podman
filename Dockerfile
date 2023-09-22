FROM quay.io/centos/centos:stream8@sha256:039dfede2e3ab9093411ac1054697eeefa6272ab57092f6c804b53cf937b8bb2

COPY tests/test_script.sh /
RUN dnf install net-tools -y

ENTRYPOINT [ "bash", "test_script.sh" ]