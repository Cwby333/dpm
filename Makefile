dpmApp_build:	
	GOOS=linux GOARCH=amd64 go build -o main ./internal/cmd/main.go
	sudo docker build --platform linux/amd64 -t dpm_app:1.0 .
build_exe:
	GOOS=linux GOARCH=amd64 go build -o main ./internal/cmd/main.go
	sudo docker compose cp main app:/app/cmd/main
	sudo docker compose up app
run:
	sudo docker compose up