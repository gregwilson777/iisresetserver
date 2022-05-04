
compile:
	docker run -e GOOS=windows -e GOARCH=amd64 -v $$(pwd):/app -w /app golang:1.18.1 go build -ldflags="-s -w" -o qamanager.exe
