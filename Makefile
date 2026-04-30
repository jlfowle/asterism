.PHONY: dev test lint build clean render-deploy

SERVICES := cluster pfsense polaris unifi

dev:
	bash scripts/dev-platform.sh

test:
	@set -e; \
	for service in $(SERVICES); do \
		$(MAKE) -C services/$$service test; \
	done

lint:
	@set -e; \
	for service in $(SERVICES); do \
		$(MAKE) -C services/$$service lint; \
	done

build:
	@set -e; \
	for service in $(SERVICES); do \
		$(MAKE) -C services/$$service build; \
	done

render-deploy:
	./scripts/update-deploy.sh
	kustomize build deploy > /dev/null

clean:
	@set -e; \
	for service in $(SERVICES); do \
		$(MAKE) -C services/$$service clean; \
	done
