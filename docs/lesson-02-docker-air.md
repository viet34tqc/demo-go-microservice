# Lesson 2 - Setup Docker và Air cho môi trường dev

Mục tiêu của bài này là chạy toàn bộ microservice stack bằng Docker Compose và dùng Air để tự rebuild service khi bạn sửa code.

Sau bài này bạn sẽ hiểu:

- Vì sao cần Docker khi học microservice.
- Cách dùng `docker-compose.dev.yml`.
- Cách tạo Docker image dev dùng chung.
- Cách Air hot reload từng service.
- Cách dùng Makefile để rút gọn lệnh.

## 1. Vì sao cần Docker?

Ở Lesson 1, bạn có thể chạy từng service bằng `go run`. Cách đó tốt để bắt đầu, nhưng khi số service tăng lên thì bạn phải tự mở nhiều terminal, tự chạy PostgreSQL, tự set biến môi trường và tự nhớ port.

Docker Compose giúp gom mọi thứ vào một lệnh:

```text
postgres
gateway
user-service
todo-service
```

Mỗi service chạy trong container riêng nhưng vẫn có thể gọi nhau qua network của Docker Compose.

## 2. File quan trọng trong project

```text
.docker/Dockerfile.dev          image dev dùng chung cho các service Go
docker-compose.dev.yml          stack dev có bind mount và hot reload
cmd/gateway/.air.toml           config Air cho gateway
cmd/user-service/.air.toml      config Air cho user-service
cmd/todo-service/.air.toml      config Air cho todo-service
Makefile                        lệnh tắt cho Docker Compose
```

## 3. Dockerfile dev

File: `.docker/Dockerfile.dev`

```dockerfile
FROM golang:1.25-alpine

WORKDIR /app

RUN apk add --no-cache git \
    && go install github.com/air-verse/air@latest
```

Ý nghĩa:

- `golang:1.25-alpine`: image có Go, nhẹ hơn image Debian.
- `WORKDIR /app`: thư mục làm việc trong container.
- `apk add git`: cần Git để Go tải một số dependency.
- `go install github.com/air-verse/air@latest`: cài Air vào image.

Image này chỉ dành cho development. Production sẽ dùng Dockerfile riêng của từng service.

## 4. Docker Compose dev

File: `docker-compose.dev.yml`

Các service chính:

```text
postgres      -> database
gateway       -> port 8080
user-service  -> port 8081
todo-service  -> port 8082
```

PostgreSQL:

```yaml
postgres:
  image: postgres:15-alpine
  container_name: mini-maestro-postgres
  environment:
    POSTGRES_USER: postgres
    POSTGRES_PASSWORD: postgres
    POSTGRES_DB: demo_mircoservice
  ports:
    - '5432:5432'
```

Lưu ý: tên database hiện tại trong project là `demo_mircoservice`. Đây là tên đang được dùng trong config và compose, nên giữ đồng bộ khi học. Sau này bạn có thể đổi thành `demo_microservice` nếu muốn sửa typo, nhưng cần đổi ở tất cả nơi liên quan.

Một service Go trong dev compose có dạng:

```yaml
user-service:
  build:
    context: .
    dockerfile: .docker/Dockerfile.dev
  working_dir: /app
  volumes:
    - .:/app
    - go-mod-cache:/go/pkg/mod
    - go-build-cache:/root/.cache/go-build
  ports:
    - '8081:8081'
  environment:
    - PORT=8081
    - DB_HOST=postgres
    - DB_PORT=5432
    - DB_USER=postgres
    - DB_PASS=postgres
    - DB_NAME=demo_mircoservice
  command: air -c cmd/user-service/.air.toml
```

Ý nghĩa:

- `build`: build image từ `.docker/Dockerfile.dev`.
- `working_dir: /app`: container chạy trong thư mục project.
- `.:/app`: mount code từ máy host vào container.
- `go-mod-cache`: cache dependency Go để chạy nhanh hơn.
- `go-build-cache`: cache build Go.
- `ports`: map port container ra máy host.
- `environment`: biến môi trường cho service.
- `command`: thay vì chạy binary cố định, container chạy Air.

## 5. Air hot reload hoạt động như thế nào?

Air theo dõi file Go. Khi bạn sửa code, Air sẽ:

1. phát hiện file thay đổi,
2. chạy `go build`,
3. dừng process cũ,
4. chạy binary mới.

Ví dụ config của gateway:

```toml
root = "."
tmp_dir = "/tmp/air/gateway"

[build]
cmd = "go build -o /tmp/air/gateway/gateway ./cmd/gateway"
entrypoint = ["/tmp/air/gateway/gateway"]
include_dir = ["cmd/gateway"]
include_ext = ["go", "mod", "sum"]
include_file = ["go.mod", "go.sum"]
poll = true
poll_interval = 500
```

Điểm quan trọng:

- `cmd`: lệnh build service.
- `entrypoint`: binary được chạy sau khi build xong.
- `include_dir`: chỉ watch folder của service đó.
- `include_file`: vẫn watch `go.mod` và `go.sum`.
- `poll = true`: hữu ích trên Docker Desktop Windows vì file event đôi khi không ổn định qua bind mount.

Nhờ `include_dir`, sửa `cmd/user-service` sẽ không restart gateway.

## 6. Chạy stack dev

Build image dev:

```bash
docker compose -f docker-compose.dev.yml build
```

Chạy foreground:

```bash
docker compose -f docker-compose.dev.yml up
```

Chạy background:

```bash
docker compose -f docker-compose.dev.yml up -d
```

Xem container:

```bash
docker compose -f docker-compose.dev.yml ps
```

Xem logs:

```bash
docker compose -f docker-compose.dev.yml logs -f
```

Dừng stack:

```bash
docker compose -f docker-compose.dev.yml down
```

Dừng và xóa volume:

```bash
docker compose -f docker-compose.dev.yml down -v
```

Lệnh `down -v` sẽ xóa volume database, nghĩa là dữ liệu PostgreSQL trong môi trường dev cũng mất.

## 7. Dùng Makefile

Makefile giúp lệnh ngắn hơn:

```makefile
COMPOSE_DEV := docker compose -f docker-compose.dev.yml
```

Các lệnh thường dùng:

```bash
make dev          # chạy dev stack foreground
make dev-up       # chạy dev stack background
make dev-build    # build image dev
make dev-down     # dừng stack
make dev-logs     # xem logs
make dev-ps       # xem container
make dev-clean    # dừng stack và xóa volume
```

Nếu máy bạn chưa có `make`, có thể dùng trực tiếp lệnh `docker compose`.

## 8. Kiểm tra các service

Gateway:

```bash
curl http://localhost:8080/health
```

User service:

```bash
curl http://localhost:8081/health
```

Todo service:

```bash
curl http://localhost:8082/health
```

Kết quả mong đợi đều có dạng:

```json
{
  "service": "gateway",
  "status": "ok"
}
```

Trường `service` sẽ khác nhau theo service bạn gọi.

## 9. Thử hot reload

1. Chạy:

```bash
make dev
```

2. Mở `cmd/gateway/main.go`.
3. Đổi response `/health`, ví dụ thêm field:

```go
"version": "dev",
```

4. Lưu file.
5. Quan sát logs, Air sẽ rebuild gateway.
6. Gọi lại:

```bash
curl http://localhost:8080/health
```

Nếu response có field mới, hot reload đã hoạt động.

## 10. Lỗi thường gặp

Docker Desktop chưa chạy:

```text
Cannot connect to the Docker daemon
```

Cách xử lý: mở Docker Desktop rồi chạy lại lệnh.

Port đã bị chiếm:

```text
bind: address already in use
```

Cách xử lý: tắt process đang dùng port `8080`, `8081`, `8082` hoặc đổi port trong compose.

Database chưa healthy:

```text
connection refused
```

Cách xử lý: đợi PostgreSQL start xong, kiểm tra logs bằng:

```bash
docker compose -f docker-compose.dev.yml logs -f postgres
```

Air không rebuild khi sửa file trên Windows:

- Đảm bảo `.air.toml` có `poll = true`.
- Đảm bảo code nằm trong folder được mount vào Docker.
- Rebuild lại image nếu vừa đổi Dockerfile:

```bash
make dev-build
```

## 11. Bài tập nhỏ

1. Chạy `make dev`.
2. Gọi `/health` của cả 3 service.
3. Sửa response `/health` của gateway và quan sát Air rebuild.
4. Chạy `make dev-down`.
5. Chạy `make dev-clean` và giải thích vì sao dữ liệu database bị xóa.

Sau Lesson 2, bạn đã có môi trường dev đủ tốt để tiếp tục học routing qua gateway, authentication giữa service và tách database theo service.
