# GitHub repositories are now mirror only

Due to recent changes in GitHub's direction, development has migrated to: https://git.ribas89.co.uk/

Have a issue/question? https://link.ribas89.co.uk/ask


Thank you for your support ❤️!


# Smoi

<img src="static/static/favicon.svg" alt="smoi" width="160">

## Purpose

The upstream distribution bundles local inference, embeddings, audio models, document processing, vector databases, and optional service dependencies. smoi is built for deployments where another service handles model inference. The goal is a smaller Docker image, lower startup overhead, and a web interface for authentication, conversation history, streaming, tool use, image inputs, and model selection.

## Removed or replaced

- Local PyTorch, torchvision, torchaudio, Transformers, Sentence Transformers, Accelerate, ONNX Runtime, Faster Whisper, ColBERT, and associated inference dependencies are excluded from the Docker runtime.
- The eager local embeddings initialization and vector database startup were removed or deferred.
- Optional vector database backends, browser automation, and development/test packages are excluded from the Docker runtime.
- Server-side chat PDF export and bundled PDF fonts were removed.
- Web fonts were replaced with system fonts.
- Emoji SVG files were replaced with native Unicode emojis.
- The onboarding slideshow, bundled background images, and upstream changelog UI were removed.
- The default UI name, icons, favicon, and splash assets use smoi branding.
- The backend-unavailable page no longer displays upstream README or Discord links.

Local RAG/vector search, local speech recognition/synthesis, OCR, and unsupported optional integrations are not included in the slim Docker image. File and image attachment uploads, file storage, and remote model requests remain in the code. Uploading a PDF does not guarantee local PDF text extraction when retrieval and document-processing features are disabled. The frontend still contains some browser-side optional dependencies; these changes primarily reduce the **backend runtime image**. Docker builds use `backend/requirements.txt`; upstream Python packaging metadata and `uv.lock` are not used by this Docker build and may still install optional dependencies in other installation methods.

## Docker Compose

Create a `compose.yaml`:

```yaml
services:
  smoi:
    image: ghcr.io/ribas89/smoi:latest
    container_name: smoi
    restart: unless-stopped
    ports:
      - "9196:9196"
    env_file:
      - .env
    environment:
      PORT: "9196"
    volumes:
      - smoi-data:/app/backend/data

volumes:
  smoi-data:
```

Create a `.env` alongside the compose file:

```dotenv
WEBUI_SECRET_KEY=replace-with-a-long-random-secret
OPENAI_API_BASE_URL=http://your-completion-gateway:9195/v1
OPENAI_API_KEY=your-gateway-key
DEFAULT_MODELS=your-model-id

OAUTH_PROVIDER_NAME=Authentik
OAUTH_CLIENT_ID=your-client-id
OAUTH_CLIENT_SECRET=your-client-secret
OPENID_PROVIDER_URL=https://your-authentik-host/application/o/your-provider/.well-known/openid-configuration
OPENID_REDIRECT_URI=https://your-smoi-host/oauth/oidc/callback
OAUTH_SCOPES=openid email profile
ENABLE_OAUTH_SIGNUP=true
DEFAULT_USER_ROLE=user
```

Start or update the container:

```bash
docker compose pull
docker compose up -d
```

Set the OpenID redirect URI to the public URL used to access smoi. Use the same persistent data volume and secret when upgrading an existing installation. Set the container's published ports and reverse proxy to match your deployment. If the completion gateway runs on the host, use an address reachable from inside the container.

## Lite defaults

The source enables remote OpenAI-compatible connectivity and authentication, uses one Uvicorn worker, bypasses local embedding/retrieval, and disables Ollama, web search, image generation, local code execution, memories, automations, calendar, notes, channels, and nonessential task-generation features by default.

The following environment variables can be set explicitly when needed:

```dotenv
WEBUI_AUTH=true
ENABLE_OPENAI_API=true
ENABLE_OLLAMA_API=false
UVICORN_WORKERS=1
BYPASS_EMBEDDING_AND_RETRIEVAL=true
ENABLE_RAG_HYBRID_SEARCH=false
ENABLE_WEB_SEARCH=false
ENABLE_IMAGE_GENERATION=false
ENABLE_CODE_EXECUTION=false
ENABLE_CODE_INTERPRETER=false
ENABLE_MEMORIES=false
ENABLE_AUTOMATIONS=false
ENABLE_CALENDAR=false
ENABLE_NOTES=false
ENABLE_CHANNELS=false
ENABLE_TITLE_GENERATION=true
```

These are source defaults; they do not have to be duplicated in `.env`. Previously saved admin settings can override defaults. Re-enabling an excluded local ML feature requires rebuilding with its dependencies.

## Troubleshooting

A backend-unavailable page indicates that the frontend could not load application configuration. Check `docker compose logs --tail=100 smoi` and confirm that `GET /api/config` returns HTTP 200. For OAuth issues, check the OpenID provider URL and redirect URI.

## Upstream reference

For upstream configuration and feature documentation, see [Open WebUI at the pinned commit](https://github.com/open-webui/open-webui/tree/f51d2b026f1b0e7283b15f093412be8b67d24770).

The original `LICENSE`, `LICENSE_NOTICE`, and `LICENSE_HISTORY` remain applicable. Review branding and attribution conditions before distributing a rebranded fork. smoi is not endorsed by Open WebUI Inc.

Created by [Timothy Jaeryang Baek](https://github.com/tjbck).
