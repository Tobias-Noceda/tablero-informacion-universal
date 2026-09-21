# ============================================================================
# BUILD STAGE
# ============================================================================
FROM node:24-slim AS builder

# Keep in sync with packageManagerDependencies.pnpm.version in pnpm-lock.yaml.
# package.json only says ^12.3.4, so an unpinned pnpm would resolve a newer 12.x and
# --frozen-lockfile would fail with ERR_PNPM_FROZEN_LOCKFILE_WITH_OUTDATED_LOCKFILE
RUN corepack enable && corepack prepare pnpm@12.4.2 --activate
ENV CI=true

WORKDIR /app

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile

COPY tsconfig.json ./
COPY src ./src
RUN pnpm run build:dev

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

EXPOSE 3000

# Requires MONGODB_URI and REDIS_URL
CMD ["node", "dist/server.js"]
