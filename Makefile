.DEFAULT_GOAL := all
.PHONY: all
all: coordinator initializer

.PHONY: protos
protos:
	nix shell nixpkgs#{go,protobuf,protoc-gen-go,protoc-gen-go-grpc} --command bash -c "go generate ./..."

.PHONY: coordinator
coordinator:
	CGO_ENABLED=0 go build -o ./coordinator/coordinator-kbs ./coordinator
	docker buildx build \
		-f ./coordinator/Containerfile \
		-t ghcr.io/katexochen/coordinator-kbs:latest \
		--push \
		./coordinator

.PHONY: initializer
initializer:
	CGO_ENABLED=0 go build -o ./initializer/initializer ./initializer
	docker buildx build \
		-f ./initializer/Containerfile \
		-t ghcr.io/katexochen/initializer:latest \
		--push \
		./initializer


.PHONY: deploy
deploy:
	./tools/genpolicy.sh ./deployment/*.yml
	kubectl apply -f ./deployment/ns.yml
	kubectl apply -f ./deployment/coordinator.yml
	kubectl apply -f ./deployment/initializer.yml

.PHONY: undeploy
undeploy:
	kubectl delete -f ./deployment

.PHONY: clean
clean:
	rm -f ./coordinator/coordinator-kbs
	rm -f ./initializer/initializer
	rm -f ./tools/genpolicy.cache/*.{tar,gz,verify}
