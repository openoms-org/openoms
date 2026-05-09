# ── Dependencies stage ───────────────────────────────────────────
FROM node:22-bookworm-slim AS deps

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci --ignore-scripts

# ── Build stage ──────────────────────────────────────────────────
FROM node:22-bookworm-slim AS builder

WORKDIR /app

COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Build-time env vars baked into the Next.js client bundle
ARG NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL}

ARG DASHBOARD_API_PROXY_URL
ENV DASHBOARD_API_PROXY_URL=${DASHBOARD_API_PROXY_URL}

ARG NEXT_PUBLIC_INPOST_GEOWIDGET_TOKEN
ENV NEXT_PUBLIC_INPOST_GEOWIDGET_TOKEN=${NEXT_PUBLIC_INPOST_GEOWIDGET_TOKEN}

ARG NEXT_PUBLIC_SENTRY_DSN
ENV NEXT_PUBLIC_SENTRY_DSN=${NEXT_PUBLIC_SENTRY_DSN}

ARG NEXT_PUBLIC_SENTRY_ENVIRONMENT
ENV NEXT_PUBLIC_SENTRY_ENVIRONMENT=${NEXT_PUBLIC_SENTRY_ENVIRONMENT}

ARG SENTRY_AUTH_TOKEN
ARG SENTRY_ORG
ARG SENTRY_PROJECT

# next.config.ts already has output: "standalone"
RUN npx next build

# ── Runtime stage ────────────────────────────────────────────────
FROM gcr.io/distroless/nodejs22-debian12:nonroot AS runner

WORKDIR /app

ENV NODE_ENV=production

# Copy only the standalone build output. UID/GID 65532 is the distroless nonroot user.
COPY --from=builder --chown=65532:65532 /app/public ./public
COPY --from=builder --chown=65532:65532 /app/.next/standalone ./
COPY --from=builder --chown=65532:65532 /app/.next/static ./.next/static

USER 65532:65532

EXPOSE 3000

ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

CMD ["server.js"]
