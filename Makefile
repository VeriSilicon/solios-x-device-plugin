.PHONY: build deploy

build:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o solios main.go

deploy:
	helm install solios deploy/helm/solios
