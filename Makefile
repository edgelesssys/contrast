.DEFAULT_GOAL := all
.PHONY: all
all: undeploy coordinator initializer deploy

.PHONY: coordinator
coordinator:
	nix run .#push-coordinator

.PHONY: initializer
initializer:
	nix run .#push-initializer

.PHONY: deploy
deploy:
	nix run .#cli -- generate -m data/manifest.json -p tools -s genpolicy-msft.json ./deployment/{coordinator,initializer}.yml
	kubectl apply -f ./deployment/ns.yml
	kubectl apply -f ./deployment/coordinator.yml
	kubectl apply -f ./deployment/initializer.yml
	kubectl apply -f ./deployment/portforwarder.yml

.PHONY: undeploy
undeploy:
	-kubectl delete -f ./deployment

.PHONY: clean
clean:
	rm -f ./tools/genpolicy.cache/*.{tar,gz,verify}
