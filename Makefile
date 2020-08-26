build:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o solios-x-device-plugin main.go
	docker build -t verisilicon/solios-x-device-plugin:0.4 .
