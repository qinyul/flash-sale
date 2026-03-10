.PHONY: dev deploy down logs-app logs-loadtest clean test-integration

# Integration tests
test-integration:
	docker-compose up --build --abort-on-container-exit test
	docker-compose rm -fsv test db_test

# Deployment: Runs DB, App, and Loadtest
benchmark:
	docker-compose up --build --abort-on-container-exit
	docker-compose down -v

# Development: Runs only DB and App
dev:
	docker-compose build db app --progress=plain 
	docker-compose up db app

# Stop all services
down:
	docker-compose down

# View application logs
logs-app:
	docker-compose logs -f app

clean:
	docker-compose down -v --remove-orphans