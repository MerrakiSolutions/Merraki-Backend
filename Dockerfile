# ==========================================
# MERRAKI BACKEND - PRODUCTION BUILD
# ==========================================

FROM node:22-alpine AS builder

WORKDIR /app

RUN corepack enable \
    && corepack prepare pnpm@10.30.0 --activate

COPY package.json pnpm-lock.yaml ./

RUN pnpm install --frozen-lockfile

COPY . .

RUN pnpm build


# ==========================================
# PRODUCTION IMAGE
# ==========================================

FROM node:22-alpine AS production

WORKDIR /app

ENV NODE_ENV=production

RUN corepack enable \
    && corepack prepare pnpm@10.30.0 --activate

COPY package.json pnpm-lock.yaml ./

RUN pnpm install --prod --frozen-lockfile \
    && pnpm store prune

COPY --from=builder /app/dist ./dist

# Copy Drizzle SQL migrations
COPY --from=builder /app/src/db/migrations ./dist/db/migrations

USER node

EXPOSE 8000

CMD ["node", "dist/app.js"]