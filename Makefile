build:
	go build -a -installsuffix cgo -o solios-x-device-plugin main.go
docker:
	docker build -t verisilicon/solios-x-device-plugin:latest .
push:
	docker push verisilicon/solios-x-device-plugin:latest
