.PHONY: run stop stop-hard int

# Запуск контейнеров через docker-compose
run:
	docker-compose up -d

# Остановка контейнеров 
stop:
	docker-compose down

# Остановка контейнеров с базой с удалением данных
stop-hard:
	docker-compose down -v

lint:
	golangci-lint run --config .golint.yaml

# Запуск всех юнит-тестов
test:
	go test -v ./...