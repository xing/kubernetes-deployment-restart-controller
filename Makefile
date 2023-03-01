.PHONY: install clean test image push-image

IMAGE := xingse/kubernetes-deployment-restart-controller
BRANCH = $(shell git rev-parse --abbrev-ref HEAD)

all: kubernetes-deployment-restart-controller

kubernetes-deployment-restart-controller: $(shell find . -name "*.go") $(wildcard go.*)
	go build

clean:
	go clean ./...

test:
	@go test -v ./...

image:
	docker build -t $(IMAGE) .

push-image:
	docker push $(IMAGE)

release: image
ifneq ($(BRANCH),master)
	$(error release only works from master, currently on '$(BRANCH)')
endif
	$(MAKE) perform-release

TAG = $(shell docker run --rm $(IMAGE) --version | grep -oE "kubernetes-deployment-restart-controller [^ ]+" | cut -d ' ' -f2)

perform-release:
	git tag $(TAG)
	git push origin $(TAG)
	git push origin master
