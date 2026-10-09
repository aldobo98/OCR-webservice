IMAGE_NAME := ocr-react-webservice
IMAGE_TAG := latest
PLATFORM := linux/amd64
DOCKERHUB_USERNAME := 

build:
	docker buildx build --platform=$(PLATFORM) -t $(DOCKERHUB_USERNAME)/$(IMAGE_NAME):$(IMAGE_TAG) --load .

build-push:
	docker buildx build --platform=$(PLATFORM) -t $(DOCKERHUB_USERNAME)/$(IMAGE_NAME):$(IMAGE_TAG) --push .