build:
	go build -o bin/kubefs github.com/rahulmedicharla/kubefs

publish:
	git tag -a v$(VERSION) -m "kubefs-cli Release version $(VERSION)"
	git push https://github.com/rahulmedicharla/homebrew-kubefs.git v$(VERSION)
	GITHUB_TOKEN=${GITHUB_TOKEN} goreleaser release --clean
	rm -rf dist

test-publish:
	goreleaser check
	goreleaser release --snapshot --clean
	rm -rf dist

#Allows to re initialize the project from sleeping state
refresh:
	brew install go
	brew install minikube
	brew install helm
	minikube start
	minikube addons enable ingress
	minikube addons enable metrics-server
	minikube stop
# echo "export GOPATH=$HOME/go\nexport GOROOT=$(brew --prefix go)/libexec\nexport PATH=$PATH:$GOPATH/bin\nexport PATH=$PATH:$GOROOT/bin" >> ~/.zprofile
	echo 'alias k=kubectl' >> ~/.zprofile
	go mod tidy
	echo 'alias kubefs=${pwd}/bin/kubefs' >> ~/.zprofile
	echo "restart the terminal to reflect the changes"

push-chart:
	helm package base-chart/deploy
	helm push base-chart/deploy-0.1.0.tgz oci://registry-1.docker.io/rmedicharla

