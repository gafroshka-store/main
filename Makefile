# Все поднятия приложения, запуск тестов и тд - ЗДЕСЬ
.PHONY: run stop stop-hard run-lint

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
	golangci-lint run --config .golangci.yml

tests:
	go test -v -cover ./...