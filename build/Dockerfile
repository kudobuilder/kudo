ARG GOLANG_VERSION=1.16
FROM golang:${GOLANG_VERSION}
ARG KUBEBUILDER_VERSION=2.3.1

# Download and install Kubebuilder

# download the release, extract and move it in one step, to reduce layer size
RUN curl --fail -L -O https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${KUBEBUILDER_VERSION}/kubebuilder_${KUBEBUILDER_VERSION}_linux_amd64.tar.gz && \
    tar -zxvf kubebuilder_${KUBEBUILDER_VERSION}_linux_amd64.tar.gz && \
    rm -f kubebuilder_${KUBEBUILDER_VERSION}_linux_amd64.tar.gz && \
    mv kubebuilder_${KUBEBUILDER_VERSION}_linux_amd64 /usr/local/kubebuilder

# update your PATH to include /usr/local/kubebuilder/bin
ENV PATH $PATH:/usr/local/kubebuilder/bin
