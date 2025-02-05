build:
	go build -o bin/kubefs .

publish:
	git tag -a v$(VERSION) -m "kubefs-cli Release version $(VERSION)"
	git push https://github.com/rahulmedicharla/homebrew-kubefs.git v$(VERSION)
	GITHUB_TOKEN=${GITHUB_TOKEN} goreleaser release --clean

#Allows to re initialize the project from sleeping state
refresh:
	brew install go
	brew install minikube
	brew install helm
	minikube start
	minikube addons enable ingress
	minikube addons enable metrics-server
	minikube stop
	echo "export GOPATH=$HOME/go\nexport GOROOT=$(brew --prefix go)/libexec\nexport PATH=$PATH:$GOPATH/bin\nexport PATH=$PATH:$GOROOT/bin" >> ~/.zprofile
	go mod tidy
	echo "restart the terminal to reflect the changes"


