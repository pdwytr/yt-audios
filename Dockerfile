FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.25-alpine AS backend
WORKDIR /app
COPY go.mod ./
COPY *.go ./
COPY --from=frontend /app/frontend/dist ./frontend/dist
RUN go build -o yt-audios .

FROM alpine:3.21
RUN apk add --no-cache python3 py3-pip ffmpeg \
    && pip3 install --break-system-packages yt-dlp
WORKDIR /app
COPY --from=backend /app/yt-audios .
COPY users.csv ./
COPY channels.csv ./
EXPOSE 8080
VOLUME ["/app/data"]
ENV DATA_DIR=/app/data
ENV USERS_FILE=/app/users.csv
ENV CHANNELS_FILE=/app/channels.csv
ENV YT_COOKIES=""
CMD ["./yt-audios"]
