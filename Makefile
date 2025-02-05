build:
	go build -o bin/kubefs .

publish:
	git tag -a v$(VERSION) -m "Release version $(VERSION)"
	git push origin v$(VERSION) --repo=https://github.com/rahulmedicharla/homebrew-kubefs.git