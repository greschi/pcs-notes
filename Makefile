.PHONY: build docker-build docker-run docker-stop clean trivy-scan

BINARY_NAME=notes-app
CONTAINER_NAME=trixie-nginx-container
IMAGE_NAME=debian-trixie-nginx

build:
	go build -o $(BINARY_NAME) main.go

docker-build: build
	docker build -t $(IMAGE_NAME) .

docker-run: docker-build
	@echo "Stopping existing container (if any)..."
	-docker stop $(CONTAINER_NAME)
	-docker rm $(CONTAINER_NAME)
	@echo "Running container..."
	docker run -d \
		--name $(CONTAINER_NAME) \
		--privileged \
		--security-opt seccomp=unconfined \
		--cgroupns=host \
		-p 8080:80 \
		--volume /sys/fs/cgroup:/sys/fs/cgroup:rw \
		$(IMAGE_NAME)
	@echo "Container is running on port 8080!"

docker-stop:
	-docker stop $(CONTAINER_NAME)
	-docker rm $(CONTAINER_NAME)

trivy-scan:
	@echo "Running Trivy vulnerability scan on $(IMAGE_NAME)..."
	trivy image $(IMAGE_NAME) || docker run --rm -v /var/run/docker.sock:/var/run/docker.sock aquasec/trivy image $(IMAGE_NAME)

clean:
	rm -f $(BINARY_NAME)
