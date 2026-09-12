wire:
	@echo "Wiring dependencies..."
	@wire ./...

run:
	@echo "Running the app..."
	@go run main.go conjure serve
