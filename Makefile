APP        := graphserv
DOCKER_REPO := rajeshkurup77/graphserv
TAG        := latest
SWAG       := $(shell go env GOPATH)/bin/swag

.PHONY: build clean run test swagger docker-build docker-push

swagger:
	$(SWAG) init --parseDependency --parseInternal

build: swagger
	CGO_ENABLED=0 go build -o $(APP) .

clean:
	rm -f $(APP)

run: build
	./$(APP)

test:
	go test ./...

docker-build:
	docker build -t $(DOCKER_REPO):$(TAG) .

docker-push: docker-build
	docker push $(DOCKER_REPO):$(TAG)
