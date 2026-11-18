# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM node:22-alpine3.20 AS frontend
ARG BUILD_HASH=dev-build
WORKDIR /app
RUN apk add --no-cache git
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm,sharing=locked \
    npm ci --force --no-audit --no-fund --prefer-offline \
        --onnxruntime-node-install-cuda=skip \
        --fetch-retries=5 \
        --fetch-retry-mintimeout=15000 \
        --fetch-retry-maxtimeout=120000 \
        --fetch-timeout=300000
COPY . .
ENV APP_BUILD_HASH=${BUILD_HASH}
RUN npm run build

FROM python:3.11.14-slim-bookworm
ARG BUILD_HASH=dev-build
ARG UID=0
ARG GID=0
ARG USE_PERMISSION_HARDENING=false

ENV PYTHONUNBUFFERED=1 \
    ENV=prod \
    PORT=8080 \
    HOME=/root \
    UV_LINK_MODE=copy \
    USE_OLLAMA_DOCKER=false \
    USE_CUDA_DOCKER=false \
    USE_SLIM_DOCKER=true \
    SCARF_NO_ANALYTICS=true \
    DO_NOT_TRACK=true \
    ANONYMIZED_TELEMETRY=false

WORKDIR /app/backend

RUN if [ "$UID" -ne 0 ]; then \
      if [ "$GID" -ne 0 ]; then addgroup --gid "$GID" app; fi; \
      adduser --uid "$UID" --gid "$GID" --home "$HOME" --disabled-password --no-create-home app; \
    fi; \
    mkdir -p /app/backend/data; \
    chown -R "$UID:$GID" /app "$HOME"

COPY --chown=$UID:$GID backend/requirements.txt ./requirements.txt
RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends curl jq build-essential python3-dev; \
    pip3 install --no-cache-dir uv; \
    uv pip install --system -r requirements.txt --no-cache-dir; \
    pip3 uninstall -y uv; \
    apt-get purge -y --auto-remove build-essential python3-dev; \
    rm -rf /var/lib/apt/lists/* /root/.cache/pip

COPY --chown=$UID:$GID --from=frontend /app/build /app/build
COPY --chown=$UID:$GID --from=frontend /app/package.json /app/package.json
COPY --chown=$UID:$GID backend/ .

RUN if [ "$USE_PERMISSION_HARDENING" = "true" ]; then \
      set -eux; \
      chgrp -R 0 /app /root || true; \
      chmod -R g+rwX /app /root || true; \
      find /app -type d -exec chmod g+s {} + || true; \
      find /root -type d -exec chmod g+s {} + || true; \
    fi

EXPOSE 8080
HEALTHCHECK CMD curl --silent --fail http://localhost:${PORT:-8080}/health | jq -ne 'input.status == true' || exit 1
USER $UID:$GID
ENV WEBUI_BUILD_VERSION=${BUILD_HASH} DOCKER=true
CMD ["bash", "start.sh"]
