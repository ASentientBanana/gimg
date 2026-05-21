

run:
	go run .

build:
	go build -o gimg main.go

windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -o gimg.exe main.go

