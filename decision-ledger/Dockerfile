ARG NODE_VERSION=22

FROM node:${NODE_VERSION}-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY tsconfig.json ./
COPY src ./src
RUN npm run build

FROM node:${NODE_VERSION}-alpine AS runtime
ENV NODE_ENV=production
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --omit=dev && npm cache clean --force
COPY --chown=root:root /app/dist ./dist
COPY --chown=root:root migrations ./migrations
COPY --chmod=0555 docker/entrypoint.sh ./docker/entrypoint.sh
USER node
ENTRYPOINT ["sh", "/app/docker/entrypoint.sh"]
CMD ["node", "dist/index.js"]