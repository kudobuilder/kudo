ARG GOLANG_VERSION=1.13
FROM golang:${GOLANG_VERSION}


# Download and install Kubebuilder

# download the release
RUN curl -L -O https://github.com/kubernetes-sigs/kubebuilder/releases/download/v2.1.0/kubebuilder_2.1.0_linux_amd64.tar.gz

# extract the archive
RUN tar -zxvf kubebuilder_2.1.0_linux_amd64.tar.gz
RUN mv kubebuilder_2.1.0_linux_amd64 kubebuilder && mv kubebuilder /usr/local/

# update your PATH to include /usr/local/kubebuilder/bin
ENV PATH $PATH:/usr/local/kubebuilder/bin