FROM oven/bun AS builder

ENV NODE_ENV production
WORKDIR /app
COPY package.json bun.lockb .
RUN bun install
COPY . .
RUN bun run build

FROM nginx:1.21.0-alpine as production
ENV NODE_ENV production
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
