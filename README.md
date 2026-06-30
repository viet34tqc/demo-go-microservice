# Demo Go Microservice

Tài liệu học microservice bằng Go, đi từ setup project cơ bản tới chạy nhiều service bằng Docker Compose và Air.

## Danh sách bài học

1. [Lesson 1 - Setup project từ đầu](docs/lesson-01-setup-project.md)
2. [Lesson 2 - Setup Docker và Air](docs/lesson-02-docker-air.md)
3. [Lesson 3 - Gateway reverse proxy](docs/lesson-03-gateway-proxy.md)
4. [Lesson 4 - JWT middleware, public route và private route](docs/lesson-04-jwt-middleware-public-private-route.md)
5. [Lesson 5 - Protobuf, Buf và generate file pb](docs/lesson-05-protobuf-buf-generate-pb.md)

## Project hiện tại

Project đang có 3 service chính:

- `gateway`: cổng vào hệ thống, chạy ở port `8080`.
- `user-service`: đăng ký, đăng nhập, lấy thông tin user, chạy ở port `8081`.
- `todo-service`: quản lý todo theo user, chạy ở port `8082`.

Stack chính:

- Go `1.25`
- Gin
- GORM
- PostgreSQL
- Docker Compose
- Air hot reload

## Chạy nhanh môi trường dev

```bash
make dev
```

Hoặc không dùng Make:

```bash
docker compose -f docker-compose.dev.yml up
```
