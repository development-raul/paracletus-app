BINARY_NAME=paracletusApp

build-app:
	@echo "Building Paracletus..."
	@go build -o build/${BINARY_NAME} .
	@echo "Paracletus built!"

run: build-app
	@echo "Starting Paracletus..."
	@./build/${BINARY_NAME} &
	@echo "Paracletus started!"

clean:
	@echo "Cleaning..."
	@go clean
	@rm build/${BINARY_NAME}
	@echo "Cleaned!"

start_compose:
	docker-compose up -d

stop_compose:
	docker-compose down

test:
	@echo "Testing..."
	@go test ./...
	@echo "Done!"

start: run

stop:
	@echo "Stopping Paracletus..."
	@-pkill -SIGTERM -f "./build/${BINARY_NAME}"
	@echo "Stopped Paracletus!"

restart: stop start