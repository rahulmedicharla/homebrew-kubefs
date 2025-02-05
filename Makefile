build:
	go build -o bin/kubefs .

publish:
	git tag -a v$(VERSION) -m "kubefs-cli Release version $(VERSION)"
	git push https://github.com/rahulmedicharla/homebrew-kubefs.git $(VERSION)
	GITHUB_TOKEN=${GITHUB_TOKEN} goreleaser release --clean --rm-dist
