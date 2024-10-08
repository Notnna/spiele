FROM node:20-alpine AS builder-client

WORKDIR /app

RUN npm install -g pnpm

COPY client/package.json client/pnpm-lock.yaml ./
RUN pnpm install
COPY client/ ./
RUN pnpm run build

FROM golang:1.23.2-alpine AS builder-server

WORKDIR /app

COPY server/ ./
RUN go build -o main .

FROM scratch AS runner

COPY --from=builder-server /app/main /app/main
COPY --from=builder-client /app/dist /app/dist

COPY server/data/categories.json /app/data/categories.json

EXPOSE 8080

ENTRYPOINT ["/app/main"]
