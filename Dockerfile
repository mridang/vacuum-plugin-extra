# syntax=docker/dockerfile:1.7
# The builder always runs on the host's native architecture (BUILDPLATFORM),
# regardless of which TARGETPLATFORM we are producing. Go cross-compiles to the
# requested target on its own, so no QEMU emulation is involved — the arm64
# image is built at native speed on an amd64 host. Base image pinned by digest;
# the readable `1.27-alpine` tag is kept for reviewers.
FROM --platform=$BUILDPLATFORM golang:1.27-alpine@sha256:70bd7ddea8b6cbe5d6943cde279d6b67feb10cc924fb9100d1d8757c1575b280 AS builder

# Docker buildx injects TARGETOS and TARGETARCH automatically ("linux" and
# "amd64" or "arm64").
ARG TARGETOS
ARG TARGETARCH

# upx: final-binary compressor (it is architecture-aware and compresses a
# cross-built binary fine).
RUN apk add --no-cache upx

WORKDIR /app

# Fetch the modules first so this layer is cached until go.mod or go.sum change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# A fixed SOURCE_DATE_EPOCH (2010-01-01 UTC) removes wall-clock time from any
# build step that honours it, so repeated builds of the same commit are
# bit-for-bit identical.
ENV SOURCE_DATE_EPOCH=1262304000

# Cross-compile a static binary, then compress. `-s -w` drops the symbol table
# and DWARF data and `-trimpath` drops the build paths. `upx --no-time` keeps
# the compressed wrapper free of a build timestamp; with the input binary
# already deterministic, the UPX output is reproducible too.
RUN CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" \
    go build -trimpath -ldflags "-s -w" -o /vacuum-extra ./cmd/vacuum-extra \
 && (upx --lzma --best --no-time /vacuum-extra \
       || echo "WARNING: upx compression failed, shipping uncompressed binary")

# Runtime stage: the published container image. Last stage, so a plain
# `docker build` defaults to it.
FROM scratch AS runtime
# OCI image-spec labels so registries (and downstream scanners) can link the
# image back to its source, license, and project metadata. Keep these in sync
# with the repository metadata on GitHub.
LABEL org.opencontainers.image.source="https://github.com/mridang/vacuum-plugin-extra"
LABEL org.opencontainers.image.url="https://github.com/mridang/vacuum-plugin-extra"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.title="vacuum-extra"
LABEL org.opencontainers.image.description="Extra lint rules for the vacuum OpenAPI linter"
COPY --from=builder /vacuum-extra /vacuum-extra
# Drop root. The scratch image has no /etc/passwd, so we reference the UID:GID
# numerically. 65532 is the conventional non-root UID used by distroless.
USER 65532:65532
ENTRYPOINT ["/vacuum-extra"]
