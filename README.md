# paracletus-app
Application developed using Paracletus

Run integration tests:
```go test . --tags integration --count=1```
See code coverage in browser
```go test -coverprofile=coverage.out . --tags integration```
```go tool cover -html=coverage.out```