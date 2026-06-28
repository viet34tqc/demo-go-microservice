# Lesson 3 - Gateway reverse proxy

Mục tiêu của bài này là hiểu cách `gateway` nhận request public từ client rồi chuyển tiếp request đó tới service nội bộ phù hợp.

Sau bài này bạn sẽ hiểu:

- Vì sao client chỉ cần gọi gateway ở port `8080`.
- `r.Any("/api/auth/*path", userProxy)` nghĩa là gì.
- `httputil.ReverseProxy` chuyển request sang service khác như thế nào.
- Vì sao gateway bỏ prefix `/api` trước khi gửi request vào service nội bộ.

## 1. Gateway đang làm gì?

Trong project này có 3 service chính:

```text
gateway       port 8080
user-service  port 8081
todo-service  port 8082
```

Client không cần gọi thẳng `user-service` hoặc `todo-service`. Client gọi gateway:

```text
POST http://localhost:8080/api/auth/login
GET  http://localhost:8080/api/users/me
GET  http://localhost:8080/api/todos
```

Gateway nhìn path request rồi quyết định chuyển sang service nào.

## 2. Route proxy trong gateway

File: `cmd/gateway/main.go`

```go
userProxy := proxy.NewReverseProxy(cfg.UserServiceURL)
todoProxy := proxy.NewReverseProxy(cfg.TodoServiceURL)

r.Any("/api/auth/*path", userProxy)
r.Any("/api/users/*path", userProxy)

r.Any("/api/todos", todoProxy)
r.Any("/api/todos/*path", todoProxy)
```

`userProxy` và `todoProxy` là các Gin handler. Chúng được tạo khi gateway start, nhưng chưa xử lý request ngay. Khi request thật đi vào và match route, Gin mới gọi handler tương ứng.

`r.Any` nghĩa là route nhận mọi HTTP method:

```text
GET
POST
PUT
PATCH
DELETE
```

Gateway không cần tự hiểu chi tiết method đó làm gì. Gateway chỉ giữ nguyên method và chuyển tiếp sang service phía sau.

`*path` là catch-all wildcard của Gin. Nó bắt phần path còn lại sau prefix.

Ví dụ:

```text
/api/auth/login       match /api/auth/*path
/api/users/me         match /api/users/*path
/api/todos/123        match /api/todos/*path
```

Riêng `/api/todos` được đăng ký riêng vì catch-all `/api/todos/*path` không match path không có dấu `/` phía sau.

## 3. Mapping route

Gateway public API:

| Client gọi gateway | Gateway chuyển tới | Service xử lý |
| --- | --- | --- |
| `POST /api/auth/register` | `POST /auth/register` | `user-service` |
| `POST /api/auth/login` | `POST /auth/login` | `user-service` |
| `GET /api/users/me` | `GET /users/me` | `user-service` |
| `POST /api/todos` | `POST /todos` | `todo-service` |
| `GET /api/todos` | `GET /todos` | `todo-service` |
| `GET /api/todos/123` | `GET /todos/123` | `todo-service` |
| `PUT /api/todos/123` | `PUT /todos/123` | `todo-service` |
| `DELETE /api/todos/123` | `DELETE /todos/123` | `todo-service` |

Điểm quan trọng là service nội bộ không biết prefix `/api`. Prefix này chỉ là public API convention của gateway.

## 4. Reverse proxy hoạt động thế nào?

File: `cmd/gateway/internal/proxy/proxy.go`

Gateway dùng Go standard library:

```go
net/http/httputil.ReverseProxy
```

Luồng xử lý khi client gọi:

```text
Client
  -> Gateway route
  -> Gin gọi userProxy hoặc todoProxy
  -> ReverseProxy rewrite request
  -> ReverseProxy gửi request sang service nội bộ
  -> Service trả response
  -> Gateway trả response đó lại cho client
```

Trong `Rewrite`, proxy đổi URL outbound sang target service:

```go
req.SetURL(targetURL)
```

Sau đó gateway bỏ prefix `/api`:

```go
req.Out.URL.Path = strings.TrimPrefix(req.In.URL.Path, "/api")
```

Ví dụ:

```text
Inbound  path: /api/auth/login
Outbound path: /auth/login
```

## 5. Service URL lấy từ đâu?

File: `cmd/gateway/config/config.go`

Gateway đọc service URL từ biến môi trường:

```text
USER_SERVICE_URL
TODO_SERVICE_URL
```

Khi chạy bằng Docker Compose dev:

```yaml
environment:
  - USER_SERVICE_URL=http://user-service:8081
  - TODO_SERVICE_URL=http://todo-service:8082
```

Trong Docker Compose, các service gọi nhau bằng tên service:

```text
http://user-service:8081
http://todo-service:8082
```

Tên `user-service` và `todo-service` là DNS name nội bộ do Docker Compose tạo.

## 6. Khi nào gateway trả 502?

Nếu gateway không gọi được service phía sau, proxy trả:

```json
{"error":"bad_gateway","message":"upstream service unavailable"}
```

Một vài nguyên nhân thường gặp:

- Service phía sau chưa chạy.
- Sai `USER_SERVICE_URL` hoặc `TODO_SERVICE_URL`.
- Container không cùng Docker Compose network.
- Service bị panic hoặc listen sai port.

Khi gặp lỗi này, nên xem logs:

```bash
make dev-logs
```

Hoặc:

```bash
docker compose -f docker-compose.dev.yml logs -f
```

Ở Lesson 4, gateway sẽ được nâng cấp thêm JWT middleware để tách public route như `/api/auth/login` khỏi private route như `/api/users/me` và `/api/todos`.
