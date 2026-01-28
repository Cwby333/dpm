dpmApp_build:	
	GOOS=linux GOARCH=amd64 go build -o main ./internal/cmd/main.go
	sudo docker build --platform linux/amd64 -t dpm_app:1.0 .

run:
	sudo docker compose up