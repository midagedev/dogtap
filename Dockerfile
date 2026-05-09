FROM node:24-alpine AS web
WORKDIR /src/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS backend
ARG DOGTAP_VERSION=dev
ARG DOGTAP_COMMIT=none
ARG DOGTAP_DATE=unknown
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
RUN CGO_ENABLED=0 go build \
  -trimpath \
  -ldflags "-s -w -X main.version=${DOGTAP_VERSION} -X main.commit=${DOGTAP_COMMIT} -X main.date=${DOGTAP_DATE}" \
  -o /out/dogtap ./cmd/dogtap

FROM alpine:3.22
RUN adduser -D -H dogtap && mkdir -p /data && chown dogtap:dogtap /data
USER dogtap
COPY --from=backend /out/dogtap /usr/local/bin/dogtap
EXPOSE 8080 8126 4317 4318
ENTRYPOINT ["dogtap"]
CMD ["serve"]
