# Simulated cloud VM: a Linux container that runs Docker and hosts a
# self-hosted GitHub Actions runner. It stands in for something like an
# EC2 instance, but costs nothing because it runs on your own machine
# under your own already-installed Docker. See deploy/README.md.
FROM ubuntu:22.04

ARG RUNNER_VERSION=2.319.1

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    curl \
    ca-certificates \
    git \
    jq \
    sudo \
    docker.io \
    docker-compose-v2 \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -m runner && usermod -aG docker runner
WORKDIR /home/runner

RUN curl -o actions-runner.tar.gz -L \
    https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz \
    && tar xzf ./actions-runner.tar.gz \
    && rm actions-runner.tar.gz \
    && chown -R runner:runner /home/runner

USER runner

COPY --chown=runner:runner entrypoint.sh /home/runner/entrypoint.sh
USER root
RUN chmod +x /home/runner/entrypoint.sh
USER runner

ENTRYPOINT ["/home/runner/entrypoint.sh"]
