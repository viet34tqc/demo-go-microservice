# Lesson 1 - Setup project Go microservice từ đầu

Mục tiêu của bài này là dựng nền móng cho một project microservice bằng Go. Sau bài này bạn sẽ hiểu vì sao project chia thành nhiều service, cách đặt cấu trúc thư mục, cách khởi tạo module Go, cách tạo HTTP server cơ bản và cách chạy từng service riêng lẻ.

## 1. Microservice là gì trong project này?

Microservice là cách chia một hệ thống lớn thành nhiều service nhỏ. Mỗi service phụ trách một nghiệp vụ rõ ràng và có thể chạy độc lập.

Trong project này ta đang chia thành:

```text
gateway       -> cổng vào hệ thống
user-service  -> xử lý user, register, login, JWT
todo-service  -> xử lý todo của từng user
postgres      -> database dùng chung trong môi trường học
```

Ở giai đoạn học, ta để các service trong cùng một repository để dễ quan sát. Cách này thường gọi là monorepo.

## 2. Chuẩn bị môi trường

Cài các công cụ sau:

- Go
- Git
- Docker Desktop, dùng ở Lesson 2
- Một công cụ gọi API như Postman, Insomnia hoặc curl

Kiểm tra Go:

```bash
go version
```

Kiểm tra Git:

```bash
git --version
```

## 3. Khởi tạo project

Tạo thư mục project:

```bash
mkdir go-microservice
cd go-microservice
```

Khởi tạo Git:

```bash
git init
```

Khởi tạo Go module:

```bash
go mod init github.com/viet34tqc/demo-go-microservice
```

Trong Go, module path là định danh gốc để import package nội bộ. Ví dụ project này import handler của todo service bằng:

```go
github.com/viet34tqc/demo-go-microservice/cmd/todo-service/handler
```

## 4. Cài dependency ban đầu

Project đang dùng Gin để làm HTTP framework:

```bash
go get github.com/gin-gonic/gin
```

Các phần user và todo cần database nên có thêm GORM và PostgreSQL driver:

```bash
go get gorm.io/gorm
go get gorm.io/driver/postgres
```

User service dùng JWT và hash password:

```bash
go get github.com/golang-jwt/jwt/v5
go get golang.org/x/crypto
```

Sau khi cài dependency, Go sẽ cập nhật:

```text
go.mod
go.sum
```

## 5. Cấu trúc thư mục

Cấu trúc hiện tại:

```text
.
├── cmd
│   ├── gateway
│   │   └── main.go
│   ├── user-service
│   │   ├── main.go
│   │   └── internal
│   │       ├── config
│   │       ├── db
│   │       ├── handler
│   │       ├── middleware
│   │       ├── model
│   │       └── util
│   └── todo-service
│       ├── main.go
│       ├── handler
│       ├── middleware
│       ├── repository
│       └── internal
│           ├── config
│           ├── db
│           └── model
├── docs
├── go.mod
├── go.sum
├── docker-compose.dev.yml
├── docker-compose.yml
└── Makefile
```

Ý nghĩa:

- `cmd/<service-name>/main.go`: entrypoint của từng service.
- `internal/`: code chỉ dùng nội bộ trong service đó.
- `handler/`: nhận HTTP request, validate input, trả response.
- `repository/`: thao tác database.
- `model/`: định nghĩa dữ liệu.
- `config/`: đọc biến môi trường.
- `middleware/`: xử lý logic nằm giữa request và handler, ví dụ auth.
- `docs/`: tài liệu học và ghi chú setup.

File `main.go` ở root hiện không phải entrypoint chính của hệ thống. Các service thật nằm trong `cmd/...`.

## 6. Tạo service đầu tiên: gateway

Gateway là service đơn giản nhất. Nó có nhiệm vụ làm cổng vào hệ thống. Ở bước đầu, ta chỉ tạo endpoint kiểm tra service còn sống hay không.

File: `cmd/gateway/main.go`

```go
package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "gateway",
			"status":  "ok",
		})
	})

	r.Run(":" + port)
}
```

Chạy gateway:

```bash
go run ./cmd/gateway
```

Kiểm tra:

```bash
curl http://localhost:8080/health
```

Kết quả mong đợi:

```json
{
  "service": "gateway",
  "status": "ok"
}
```

## 7. Tạo user-service

User service phụ trách:

- Đăng ký user: `POST /auth/register`
- Đăng nhập: `POST /auth/login`
- Lấy thông tin user hiện tại: `GET /users/me`
- Health check: `GET /health`

Service này cần database nên cấu trúc nhiều hơn gateway:

```text
cmd/user-service
├── main.go
└── internal
    ├── config
    ├── db
    ├── handler
    ├── middleware
    ├── model
    └── util
```

Luồng xử lý cơ bản:

```text
HTTP request
-> Gin router
-> middleware nếu route cần bảo vệ
-> handler
-> database qua GORM
-> JSON response
```

Các biến môi trường user-service đang dùng:

```text
PORT hoặc USER_SERVICE_PORT
DB_HOST
DB_PORT
DB_USER
DB_PASS
DB_NAME
JWT_SECRET
```

Nếu không set biến môi trường, project có giá trị mặc định trong `internal/config/config.go`.

Chạy local cần có PostgreSQL đang chạy:

```bash
go run ./cmd/user-service
```

## 8. Tạo todo-service

Todo service phụ trách:

- Tạo todo: `POST /todos`
- Lấy danh sách todo: `GET /todos`
- Lấy chi tiết todo: `GET /todos/:id`
- Cập nhật todo: `PUT /todos/:id`
- Xóa todo: `DELETE /todos/:id`
- Health check: `GET /health`

Service này có thêm `repository` để tách logic database ra khỏi handler:

```text
cmd/todo-service
├── main.go
├── handler
├── middleware
├── repository
└── internal
    ├── config
    ├── db
    └── model
```

Todo service đang yêu cầu user id trước khi truy cập `/todos`. Trong project hiện tại, middleware đọc user id từ request context/header tùy cách bạn triển khai tiếp theo. Ý tưởng chính là: todo luôn thuộc về một user cụ thể.

Các biến môi trường todo-service đang dùng:

```text
PORT hoặc TODO_SERVICE_PORT
DB_HOST
DB_PORT
DB_USER
DB_PASS
DB_NAME
```

Chạy local cần có PostgreSQL đang chạy:

```bash
go run ./cmd/todo-service
```

## 9. Build từng service

Build gateway:

```bash
go build -o bin/gateway ./cmd/gateway
```

Build user-service:

```bash
go build -o bin/user-service ./cmd/user-service
```

Build todo-service:

```bash
go build -o bin/todo-service ./cmd/todo-service
```

Build toàn bộ project:

```bash
go build ./...
```

## 10. Nguyên tắc đặt code khi học microservice

Giữ `main.go` mỏng:

- đọc config
- kết nối database
- tạo router
- đăng ký route
- start server

Đưa nghiệp vụ vào handler, service, repository hoặc package riêng:

- handler xử lý HTTP
- repository xử lý database
- middleware xử lý auth, user id, logging
- util chứa helper nhỏ như JWT, password hash

Không nên để tất cả logic trong `main.go`, vì sau này mỗi service sẽ lớn lên rất nhanh.

## 11. Bài tập nhỏ

1. Chạy `go run ./cmd/gateway`, gọi `GET /health`.
2. Đổi port gateway bằng biến môi trường `PORT=8090`.
3. Thêm endpoint `GET /version` cho gateway.
4. Chạy `go build ./...` để kiểm tra toàn bộ project còn build được.

Sau Lesson 1, bạn đã có nền móng project Go microservice. Lesson 2 sẽ đưa các service này vào Docker Compose và dùng Air để hot reload khi code thay đổi.
