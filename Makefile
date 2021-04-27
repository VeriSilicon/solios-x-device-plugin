build:
	go build -a -installsuffix cgo -o solios-x-device-plugin main.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build -a -installsuffix cgo -o solios-x-device-plugin-arm64 main.go
docker:
	docker build -t verisilicon/solios-x-device-plugin:latest .
push:
	docker push verisilicon/solios-x-device-plugin:latest
