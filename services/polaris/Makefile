.PHONY: run build test lint deps clean

run: deps
	yarn start

build: dist/main.js

dist/main.js: $(wildcard src/**/*.js) deps
	yarn build

test: deps
	yarn test

lint: deps
	yarn lint

deps:
	yarn install --frozen-lockfile --check-files

clean:
	rm -rf dist coverage node_modules test-results
