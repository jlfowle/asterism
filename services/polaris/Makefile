.PHONY: run build test lint deps clean verify-registry

run: deps verify-registry
	yarn start

build: verify-registry dist/main.js

dist/main.js: $(wildcard src/**/*.js) deps
	yarn build

test: deps verify-registry
	yarn test

lint: deps verify-registry
	yarn lint

verify-registry:
	../../scripts/generate-microfrontends-registry.sh --check

deps:
	yarn install --frozen-lockfile --check-files

clean:
	rm -rf dist coverage node_modules test-results
