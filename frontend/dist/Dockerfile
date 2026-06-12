# ============================================================================
# BUILD STAGE
# ============================================================================
FROM node:22-alpine AS builder

# Install pnpm
RUN corepack enable && corepack prepare pnpm@11.6.0 --activate
ENV PNPM_CONFIG_MINIMUM_RELEASE_AGE=0
ENV CI=true

WORKDIR /app

# Copy dependency files (leverages Docker layer caching)
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./

# Install dependencies (production + dev, needed for build)
# --allow-scripts: permits build scripts to run (needed for esbuild and other native modules)
# --frozen-lockfile: ensures reproducible builds across machines
RUN pnpm install --frozen-lockfile

# Copy source code
COPY . .

# Build the SPA
# The adapter-static will output to 'build' directory by default
RUN pnpm run build

# ============================================================================
# PRODUCTION STAGE
# ============================================================================
FROM nginx:stable-alpine

# Labels for metadata
LABEL maintainer="Your Organization"
LABEL description="SvelteKit Static SPA running on Nginx"

# Install curl for health checks (lightweight)
RUN apk add --no-cache curl

# Remove default nginx config
RUN rm -f /etc/nginx/conf.d/default.conf

# Copy custom nginx configuration (from root, not dist/)
COPY dist/nginx.conf /etc/nginx/conf.d/default.conf

# Copy built static files from builder stage
# Assuming SvelteKit build output is in 'build' directory
COPY --from=builder /app/build /usr/share/nginx/html

# Create non-root user for security (nginx already runs as www-data, but explicit is good)
# nginx:x:101:102:nginx user:/var/cache/nginx:/sbin/nologin
# We rely on nginx's built-in unprivileged user

# Expose port
EXPOSE 80

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost/index.html || exit 1

# Start nginx in foreground mode
CMD ["nginx", "-g", "daemon off;"]