FROM quay.io/centos/centos:stream8@sha256:f24005786295703fc65e5cd74ab90497a05479fac780790a43eab5729f9e098f

COPY tests/test_script.sh /
RUN dnf install net-tools -y

ENTRYPOINT [ "bash", "test_script.sh" ]