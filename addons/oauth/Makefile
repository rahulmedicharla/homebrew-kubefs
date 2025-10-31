compile:
	docker buildx build  --platform=linux/amd64,linux/arm64 -t rmedicharla/auth:latest .
	docker push rmedicharla/auth:latest

build:
	go build -o oauth2 .
	rm ./oauth2

clean:
	go mod tidy -v