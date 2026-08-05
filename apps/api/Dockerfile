FROM debian:12-slim AS build

ARG TARGETOS=linux
ARG TARGETARCH

ENV DEBIAN_FRONTEND=noninteractive
ENV MISE_EXPERIMENTAL=1
ENV MISE_INSTALL_PATH=/usr/local/bin/mise
ENV PATH=/root/.local/bin:/usr/local/bin:${PATH}

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl xz-utils \
    && rm -rf /var/lib/apt/lists/*

RUN curl https://mise.run | sh

WORKDIR /repo

COPY mise.toml mise.lock ./
COPY apps/api/mise.toml apps/api/go.mod apps/api/go.sum ./apps/api/

RUN mise trust /repo/mise.toml \
    && mise trust /repo/apps/api/mise.toml \
    && mise install go

COPY apps/api ./apps/api

RUN cd /repo/apps/api \
    && CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH:-amd64} mise :build

FROM gcr.io/distroless/static-debian12

COPY --from=build /repo/apps/api/bin/api /api

EXPOSE 4000

ENTRYPOINT ["/api"]
