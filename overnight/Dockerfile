FROM rust:1.89-bookworm AS builder

ARG NODE_MAJOR=20

RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    pkg-config \
    libssl-dev \
    build-essential \
  && curl -fsSL https://deb.nodesource.com/setup_${NODE_MAJOR}.x | bash - \
  && apt-get install -y --no-install-recommends nodejs \
  && npm i -g npm@latest \
  && apt-get clean \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace/overnight

COPY package*.json ./
RUN npm ci

COPY . ./
RUN npm run build
RUN cargo build --release --manifest-path src-rust/Cargo.toml --bin overnight-bridge

FROM node:20-bookworm-slim AS runtime

WORKDIR /app
ENV NODE_ENV=production
ENV PORT=1420
ENV BRIDGE_BIN=/usr/local/bin/overnight-bridge

RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    ca-certificates \
    libssl3 \
  && apt-get clean \
  && rm -rf /var/lib/apt/lists/*

COPY package*.json ./
RUN npm ci --omit=dev

COPY --from=builder /workspace/overnight/dist ./dist
COPY --from=builder /workspace/overnight/server ./server
COPY --from=builder /workspace/overnight/src-rust/target/release/overnight-bridge /usr/local/bin/overnight-bridge

EXPOSE 1420

CMD ["node", "server/index.js"]
