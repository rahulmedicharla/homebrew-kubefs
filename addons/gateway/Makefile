compile:
	docker buildx build --platform=linux/amd64,linux/arm64 -t rmedicharla/gateway:latest .
	docker push rmedicharla/gateway:latest

build:
	go build -o gateway .
	rm ./gateway

clean:
	go mod tidy -v