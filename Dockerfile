FROM quay.io/centos/centos:stream8@sha256:100d23534e48465a1e00573a3535f496d4cdf39779cbc8405612d56cb31f299c

COPY tests/test_script.sh /
RUN dnf install net-tools -y

ENTRYPOINT [ "bash", "test_script.sh" ]
