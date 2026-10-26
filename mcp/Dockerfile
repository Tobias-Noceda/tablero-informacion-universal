# ============================================================================
# BUILD STAGE
# ============================================================================
FROM node:24-slim AS builder

# The pnpm that wrote pnpm-lock.yaml; --frozen-lockfile fails on another major.
RUN corepack enable && corepack prepare pnpm@11.21.0 --activate
ENV CI=true

WORKDIR /app

COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY tsconfig.json xmcp.config.ts xmcp.svg ./
COPY src ./src
RUN pnpm build

# Keep only production dependencies for the runtime image
RUN pnpm prune --prod

# ============================================================================
# PRODUCTION STAGE
# ============================================================================
FROM node:24-slim

ENV NODE_ENV=production
WORKDIR /app

COPY --from=builder /app/package.json ./
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/dist ./dist

USER node

EXPOSE 3001

# Requires API_URL: the backend's origin (the server adds /api)
CMD ["node", "dist/http.js"]
