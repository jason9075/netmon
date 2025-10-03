IMAGE ?= netmon:local

.PHONY: build run stop logs clean

build:
	docker compose build

run: build
	docker compose up -d

stop:
	docker compose down

logs:
	docker compose logs -f

clean:
	rm -rf data
