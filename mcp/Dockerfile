FROM ghcr.io/pnpm/pnpm:12 AS builder

ENV CI=true
WORKDIR /app

COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY tsconfig.json xmcp.config.ts xmcp.svg ./
COPY src ./src

RUN pnpm build
RUN pnpm prune --prod

FROM node:24-slim

ENV NODE_ENV=production
WORKDIR /app
USER node

EXPOSE 3001

COPY --from=builder /app/package.json ./
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/dist ./dist

CMD ["node", "dist/http.js"]
