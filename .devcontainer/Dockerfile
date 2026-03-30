FROM mcr.microsoft.com/devcontainers/javascript-node:1-22-bookworm

ARG GO_VERSION=1.23.8
ARG KUBECTL_VERSION=v1.32.2
ARG KUSTOMIZE_VERSION=v5.6.0
ARG COSIGN_VERSION=v2.4.1
ARG TRIVY_VERSION=0.69.3
ARG SYFT_VERSION=v1.20.0
ARG OC_VERSION=4.18.13

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        git \
        gnupg \
        jq \
        make \
        unzip \
        xz-utils \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

RUN ARCH="$(dpkg --print-architecture)" \
    && case "${ARCH}" in \
        amd64) TOOL_ARCH="amd64"; GO_ARCH="amd64"; TRIVY_ARCH="64bit"; OC_ARCHIVE="openshift-client-linux-${OC_VERSION}.tar.gz" ;; \
        arm64) TOOL_ARCH="arm64"; GO_ARCH="arm64"; TRIVY_ARCH="ARM64"; OC_ARCHIVE="openshift-client-linux-arm64-${OC_VERSION}.tar.gz" ;; \
        *) echo "Unsupported architecture: ${ARCH}" && exit 1 ;; \
      esac \
    && curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" -o /tmp/go.tgz \
    && rm -rf /usr/local/go \
    && tar -C /usr/local -xzf /tmp/go.tgz \
    && ln -sf /usr/local/go/bin/go /usr/local/bin/go \
    && ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt \
    && rm /tmp/go.tgz \
    && curl -fsSL "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/${TOOL_ARCH}/kubectl" -o /usr/local/bin/kubectl \
    && chmod +x /usr/local/bin/kubectl \
    && curl -fsSL "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/${KUSTOMIZE_VERSION}/kustomize_${KUSTOMIZE_VERSION}_linux_${TOOL_ARCH}.tar.gz" -o /tmp/kustomize.tgz \
    && tar -C /usr/local/bin -xzf /tmp/kustomize.tgz kustomize \
    && rm /tmp/kustomize.tgz \
    && curl -fsSL "https://github.com/sigstore/cosign/releases/download/${COSIGN_VERSION}/cosign-linux-${TOOL_ARCH}" -o /usr/local/bin/cosign \
    && chmod +x /usr/local/bin/cosign \
    && curl -fsSL "https://github.com/aquasecurity/trivy/releases/download/v${TRIVY_VERSION}/trivy_${TRIVY_VERSION}_Linux-${TRIVY_ARCH}.tar.gz" -o /tmp/trivy.tgz \
    && tar -C /usr/local/bin -xzf /tmp/trivy.tgz trivy \
    && rm /tmp/trivy.tgz \
    && curl -fsSL "https://github.com/anchore/syft/releases/download/${SYFT_VERSION}/syft_${SYFT_VERSION#v}_linux_${TOOL_ARCH}.tar.gz" -o /tmp/syft.tgz \
    && tar -C /usr/local/bin -xzf /tmp/syft.tgz syft \
    && rm /tmp/syft.tgz \
    && curl -fsSL "https://mirror.openshift.com/pub/openshift-v4/clients/ocp/${OC_VERSION}/${OC_ARCHIVE}" -o /tmp/oc.tgz \
    && tar -C /tmp -xzf /tmp/oc.tgz oc kubectl \
    && mv /tmp/oc /usr/local/bin/oc \
    && mv /tmp/kubectl /usr/local/bin/oc-kubectl \
    && chmod +x /usr/local/bin/oc /usr/local/bin/oc-kubectl \
    && rm -f /tmp/oc.tgz

RUN corepack enable \
    && corepack prepare yarn@1.22.22 --activate

ENV PATH="/usr/local/go/bin:${PATH}"
