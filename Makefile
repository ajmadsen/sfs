all: sfs static/js/project.js

sfs: *.go
	go build

static/js/project.js: src/main.coffee
	coffee -b -j $@ -c $<

.PHONY: src/main.coffee
