FROM ubuntu:15.10

MAINTAINER Rancher Labs, Inc.

RUN apt-get update && apt-get install -y software-properties-common
RUN add-apt-repository ppa:gluster/glusterfs-3.7 && apt-get update && apt-get install -y curl glusterfs-client
RUN mkdir -p /var/lib/rancher/convoy-agent && \
    curl -sSL -o convoy.tar.gz https://github.com/rancher/convoy/releases/download/v0.4.1/convoy.tar.gz && tar -xvzf convoy.tar.gz && mv convoy/convoy /var/lib/rancher/convoy-agent && \
    curl -sSL -o share-mnt https://github.com/rancher/runc/releases/download/share-mnt-v0.0.2/share-mnt && chmod u+x share-mnt && mv share-mnt /var/lib/rancher/convoy-agent

ENV PATH /var/lib/rancher/convoy-agent:$PATH

COPY convoy-agent /var/lib/rancher/convoy-agent
COPY launch /

ENTRYPOINT ["/launch"]
