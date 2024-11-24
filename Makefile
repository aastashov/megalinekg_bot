init:
	cp config.example.yml config.yml

build-docker:
	 docker build --build-arg RELEASE=1 -t megalinebot:latest .

run-docker:
	docker run -d -v $(shell pwd)/config.yml:/app/config.yml --name megalinebot megalinebot:latest
