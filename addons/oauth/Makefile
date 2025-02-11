compile:
	docker buildx build -t rmedicharla/auth:latest .
	docker push rmedicharla/auth:latest

build:
	go build -o oauth2 ./main.go
	rm ./oauth2