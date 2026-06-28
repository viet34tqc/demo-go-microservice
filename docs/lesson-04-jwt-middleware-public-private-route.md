# Lesson 4 - JWT middleware, public route và private route

Mục tiêu của bài này là hiểu cách gateway phân biệt route public và private, cách JWT middleware kiểm tra token, và vì sao gateway thêm `X-User-ID` trước khi chuyển request sang service phía sau.

Sau bài này bạn sẽ hiểu:

- Public route là gì và route nào không cần token.
- Private route là gì và route nào bắt buộc có token.
- JWT middleware trong gateway kiểm tra `Authorization: Bearer <token>` như thế nào.
- Vì sao `user-service` và `gateway` phải dùng cùng `JWT_SECRET`.
- Vì sao `todo-service` đọc user từ header `X-User-ID` thay vì tự parse JWT.

## 1. Public route và private route là gì?

Public route là route client có thể gọi mà chưa cần đăng nhập. Trong project này, public route là nhóm auth:

```text
POST /api/auth/register
POST /api/auth/login
```

Hai route này phải public vì user mới chưa có token. Nếu bắt token ở đây thì user không thể đăng ký hoặc đăng nhập.

Private route là route chỉ cho phép user đã đăng nhập gọi. Client phải gửi JWT trong header:

```text
Authorization: Bearer <token>
```

Trong project này, private route là:

```text
GET    /api/users/me
POST   /api/todos
GET    /api/todos
GET    /api/todos/:id
PUT    /api/todos/:id
DELETE /api/todos/:id
```

## 2. Gateway chia route như thế nào?

File: `cmd/gateway/main.go`

```go
jwtMiddleware := middleware.NewJWTMiddleware(cfg.JWTSecret)
userProxy := proxy.NewReverseProxy(cfg.UserServiceURL)
todoProxy := proxy.NewReverseProxy(cfg.TodoServiceURL)

api := r.Group("/api")

// Public auth routes
api.Any("/auth/*path", userProxy)

// Private routes
private := api.Group("")
private.Use(jwtMiddleware.RequireAuth())

private.Any("/users/*path", userProxy)
private.Any("/todos", todoProxy)
private.Any("/todos/*path", todoProxy)
```

Ý tưởng chính:

- Tất cả public API đi qua prefix `/api`.
- `/api/auth/*path` được đăng ký trước và không dùng JWT middleware.
- `private := api.Group("")` tạo một nhóm route cũng nằm dưới `/api`.
- `private.Use(jwtMiddleware.RequireAuth())` gắn middleware cho toàn bộ route private.
- Những route đăng ký sau dòng `Use(...)` sẽ phải qua middleware trước khi proxy sang service nội bộ.

Vì vậy request tới `/api/auth/login` đi thẳng tới `user-service`, còn request tới `/api/todos` phải có token hợp lệ trước.

## 3. Luồng đăng ký và đăng nhập

Khi client gọi:

```text
POST /api/auth/register
POST /api/auth/login
```

Gateway chỉ proxy sang `user-service`:

```text
Client
  -> Gateway /api/auth/login
  -> user-service /auth/login
  -> user-service kiểm tra email/password
  -> user-service tạo JWT
  -> Gateway trả response lại client
```

Token được tạo ở `cmd/user-service/internal/util/jwt.go`:

```go
type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}
```

JWT trong project có claim quan trọng là `user_id`. Claim này cho biết token thuộc về user nào.

## 4. JWT middleware trong gateway làm gì?

File: `cmd/gateway/internal/middleware/jwt.go`

Middleware `RequireAuth()` chạy theo các bước:

```text
1. Đọc Authorization header.
2. Kiểm tra format Bearer <token>.
3. Parse token bằng JWT secret.
4. Chỉ chấp nhận thuật toán HS256.
5. Kiểm tra token còn valid.
6. Lấy user_id từ claims.
7. Gắn user_id vào Gin context.
8. Set header X-User-ID để service phía sau dùng.
```

Code quan trọng:

```go
authHeader := c.GetHeader("Authorization")
```

Header phải có dạng:

```text
Authorization: Bearer eyJ...
```

Gateway parse token bằng secret:

```go
token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, jwt.ErrTokenSignatureInvalid
	}

	return []byte(m.secret), nil
}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
```

Sau khi token hợp lệ, gateway lấy `user_id`:

```go
userIDFloat, ok := claims["user_id"].(float64)
```

`jwt.MapClaims` decode số JSON thành `float64`, nên code cần convert về string trước khi gắn vào header:

```go
userID := strconv.FormatUint(uint64(userIDFloat), 10)
c.Set("user_id", userID)
c.Request.Header.Set("X-User-ID", userID)
```

## 5. Vì sao gateway set `X-User-ID`?

`todo-service` không tự parse JWT. Nó chỉ cần biết user hiện tại là ai để query đúng todo.

Gateway là nơi kiểm tra token ở biên hệ thống. Sau khi token hợp lệ, gateway chuyển danh tính user sang service phía sau bằng internal header:

```text
X-User-ID: 5
```

File: `cmd/todo-service/middleware/user_middleware.go`

```go
userIDHeader := c.GetHeader("X-User-ID")
```

Sau đó todo handler dùng user id này để chỉ thao tác trên todo của user hiện tại:

```go
todos, err := h.repo.FindAllByUserID(userID)
```

Cách này giúp `todo-service` không cần biết JWT secret và không phải duplicate logic parse JWT. Trong kiến trúc này, gateway chịu trách nhiệm xác thực request từ client, còn service phía sau tin vào header do gateway đã thêm.

Riêng `GET /api/users/me` hơi khác một chút. Gateway vẫn verify JWT trước, sau đó proxy sang `user-service`. Vì reverse proxy giữ lại request header, `Authorization` vẫn được chuyển tiếp, nên `user-service` tiếp tục dùng middleware riêng để parse token và set `userID` trong Gin context:

```go
protected := r.Group("/")
protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))
{
	protected.GET("/users/me", authHandler.Me)
}
```

Nói ngắn gọn:

- `user-service` tạo token và vẫn biết cách parse token cho route của chính nó.
- `todo-service` nhận user id đã được gateway xác thực qua `X-User-ID`.

## 6. Vì sao `JWT_SECRET` phải giống nhau?

`user-service` là nơi tạo token:

```text
email/password đúng -> tạo JWT bằng JWT_SECRET
```

`gateway` là nơi verify token cho private route:

```text
Authorization: Bearer <token> -> verify bằng JWT_SECRET
```

Nếu hai service dùng secret khác nhau, token login vẫn được tạo thành công nhưng gateway sẽ reject khi gọi private route:

```json
{"error":"invalid or expired token"}
```

Trong `docker-compose.dev.yml`, cả gateway và user-service đang dùng cùng biến:

```yaml
- JWT_SECRET=${JWT_SECRET:-dev-secret-change-me}
```

Khi chạy production, cũng cần đảm bảo gateway và user-service nhận cùng một `JWT_SECRET`.

## 7. Route mapping sau khi có JWT middleware

| Client gọi gateway | Public/private | Gateway xử lý | Service nhận |
| --- | --- | --- | --- |
| `POST /api/auth/register` | Public | Proxy thẳng | `POST /auth/register` |
| `POST /api/auth/login` | Public | Proxy thẳng | `POST /auth/login` |
| `GET /api/users/me` | Private | Verify JWT rồi proxy | `GET /users/me` |
| `POST /api/todos` | Private | Verify JWT, set `X-User-ID`, proxy | `POST /todos` |
| `GET /api/todos` | Private | Verify JWT, set `X-User-ID`, proxy | `GET /todos` |
| `GET /api/todos/1` | Private | Verify JWT, set `X-User-ID`, proxy | `GET /todos/1` |
| `PUT /api/todos/1` | Private | Verify JWT, set `X-User-ID`, proxy | `PUT /todos/1` |
| `DELETE /api/todos/1` | Private | Verify JWT, set `X-User-ID`, proxy | `DELETE /todos/1` |

Gateway vẫn bỏ prefix `/api` trước khi gửi sang service nội bộ. Phần này đã học ở Lesson 3.

## 8. Test bằng curl

Chạy stack dev:

```bash
make dev
```

Đăng ký user qua gateway:

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Demo User","email":"demo@example.com","password":"secret123"}'
```

Hoặc đăng nhập:

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@example.com","password":"secret123"}'
```

Response sẽ có `token`. Lưu token vào biến shell:

```bash
TOKEN="paste-token-here"
```

Gọi private route không có token:

```bash
curl http://localhost:8080/api/todos
```

Kết quả mong đợi:

```json
{"error":"missing authorization header"}
```

Gọi private route có token:

```bash
curl http://localhost:8080/api/todos \
  -H "Authorization: Bearer $TOKEN"
```

Tạo todo:

```bash
curl -X POST http://localhost:8080/api/todos \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"title":"Learn JWT middleware"}'
```

Nếu token hợp lệ, gateway sẽ set `X-User-ID`, todo-service sẽ lưu todo theo user hiện tại.

## 9. Lỗi thường gặp

Thiếu header:

```json
{"error":"missing authorization header"}
```

Sai format header:

```json
{"error":"invalid authorization header format"}
```

Đúng format nhưng token sai, hết hạn hoặc khác secret:

```json
{"error":"invalid or expired token"}
```

Token không có `user_id`:

```json
{"error":"missing user_id claim"}
```

Nếu gặp lỗi khi gọi todo-service trực tiếp ở port `8082`, nhớ rằng todo-service yêu cầu `X-User-ID`. Trong luồng thật, client nên gọi qua gateway ở port `8080` để gateway tự verify JWT và thêm header này.
