ARG GOLANG_VERSION=1.10
FROM golang:${GOLANG_VERSION}


# Download and install Kubebuilder

# download the release
RUN curl -L -O https://github.com/kubernetes-sigs/kubebuilder/releases/download/v1.0.8/kubebuilder_1.0.8_linux_amd64.tar.gz

# extract the archive
RUN tar -zxvf kubebuilder_1.0.8_linux_amd64.tar.gz
RUN mv kubebuilder_1.0.8_linux_amd64 kubebuilder && mv kubebuilder /usr/local/

# update your PATH to include /usr/local/kubebuilder/bin
ENV PATH $PATH:/usr/local/kubebuilder/bin