.PHONY: run build test lint

run: dist/main.js
	yarn start

build: dist/main.js

dist/main.js: $(wildcard src/**/*.js) node_modules
	yarn build

test: $(wildcard src/**/*.js) node_modules
	yarn test

lint: node_modules
	yarn lint

node_modules: package.json yarn.lock
	yarn install
	touch node_modules
