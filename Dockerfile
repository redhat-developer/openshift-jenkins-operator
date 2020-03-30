FROM registry.access.redhat.com/ubi8/ubi:latest

ARG GO_PACKAGE_PATH=github.com/redhat-developer/openshift-jenkins-operator
ENV LANG=en_US.utf8
ENV GOPATH /tmp/go
ENV GIT_COMMITTER_NAME devtools
ENV GIT_COMMITTER_EMAIL devtools@redhat.com
ENV PATH=:$GOPATH/bin:/tmp/goroot/go/bin:$PATH

ENV GO_VERSION=1.14.1 \
    GO_DOWNLOAD_SITE=https://dl.google.com/go/ \
    OPERATOR_SDK_VERSION=v0.16.0 \
    OPERATOR_SDK_DOWNLOAD_SITE=https://github.com/operator-framework/operator-sdk/releases/download/ \
    KUBECTL_VERSION=v1.14.3 \
    KUBECTL_DOWNLOAD_SITE=https://storage.googleapis.com/kubernetes-release/release/

ENV GO_DIST=go$GO_VERSION.linux-amd64.tar.gz \
    OPERATOR_SDK_DIST=operator-sdk-OPERATOR_SDK_VERSION-x86_64-linux-gnu


WORKDIR /tmp
RUN mkdir -p $GOPATH/bin &&  mkdir -p /tmp/goroot

RUN echo $GO_DOWNLOAD_SITE/$GO_DIST && \
    curl -Lo $GO_DIST $GO_DOWNLOAD_SITE/$GO_DIST && \
    tar -C /tmp/goroot -xzf $GO_DIST
RUN curl -Lo kubectl $KUBECTL_DOWNLOAD_SITE/$KUBECTL_VERSION/bin/linux/amd64/kubectl && \
    chmod +x kubectl && mv kubectl $GOPATH/bin/
RUN curl -Lo operator-sdk $OPERATOR_SDK_DOWNLOAD_SITE/$OPERATOR_SDK_VERSION/$OPERATOR_SDK_DIST && \
    chmod +x operator-sdk && mv operator-sdk $GOPATH/bin/
RUN mkdir -p ${GOPATH}/src/${GO_PACKAGE_PATH}/

WORKDIR ${GOPATH}/src/${GO_PACKAGE_PATH}


ENTRYPOINT [ "/bin/bash" ]

