.PHONY: build dev-backend dev-frontend clean

build:
	cd frontend && npm run build
	go build -o yt-audios .

dev-backend:
	go run main.go

dev-frontend:
	cd frontend && npm run dev

clean:
	rm -rf frontend/dist yt-audios
