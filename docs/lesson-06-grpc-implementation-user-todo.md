# Lesson 6 - Thực thi gRPC cho user-service và todo-service

Mục tiêu của bài này là ghi lại phần code change để biến contract Protobuf đã generate thành gRPC server chạy thật trong `user-service` và `todo-service`.

Sau bài này bạn sẽ hiểu:

- gRPC implementation nằm ở đâu trong từng service.
- Vì sao tách thêm service layer trước khi viết gRPC server.
- Request protobuf được map sang input nội bộ như thế nào.
- Error nội bộ được đổi sang gRPC status code ra sao.
- Cách khởi động gRPC server song song với HTTP server hiện có.

## 1. Bối cảnh trước khi code change

Ở bài trước, project đã có contract Protobuf và code generate:

```text
proto/user/v1/user.proto
proto/todo/v1/todo.proto
gen/go/user/v1/user.pb.go
gen/go/user/v1/user_grpc.pb.go
gen/go/todo/v1/todo.pb.go
gen/go/todo/v1/todo_grpc.pb.go
```

Các file `*_grpc.pb.go` chỉ sinh ra interface, registration function và client stub. Chúng chưa có business logic thật.

Vì vậy bước tiếp theo là tự viết implementation cho các interface sau:

```text
userpb.UserServiceServer
todopb.TodoServiceServer
```

## 2. Tổng quan code change

Các phần chính được thêm vào:

```text
cmd/user-service/internal/service/user_service.go
cmd/user-service/internal/grpc/server.go
cmd/todo-service/service/todo_service.go
cmd/todo-service/internal/grpc/server.go
```

Hai file `main.go` của service cũng được cập nhật để start gRPC server:

```text
cmd/user-service/main.go
cmd/todo-service/main.go
```

Luồng tổng quan:

```text
.proto
-> buf generate proto
-> gen/go/.../*_grpc.pb.go
-> internal grpc server implement generated interface
-> service layer xử lý business logic
-> GORM thao tác database
```

## 3. Vì sao cần service layer?

Trước gRPC, HTTP handler có thể gọi repository hoặc database trực tiếp. Khi thêm gRPC, nếu copy logic từ HTTP handler sang gRPC handler thì code sẽ bị lặp.

Vì vậy code change thêm service layer:

- `UserService`: xử lý register, login, get me.
- `TodoService`: xử lý create, list, get, update, delete todo.

gRPC server chỉ còn nhiệm vụ:

- nhận protobuf request
- validate phần liên quan đến transport nếu cần
- gọi service layer
- map output sang protobuf response
- map error sang `status.Error`

Cách tách này giúp sau này HTTP handler cũng có thể dùng lại service layer, thay vì mỗi transport tự giữ một bản business logic riêng.

## 4. Triển khai gRPC cho user-service

File chính:

```text
cmd/user-service/internal/grpc/server.go
```

Struct server:

```go
type UserGRPCServer struct {
	userpb.UnimplementedUserServiceServer

	userService *service.UserService
}
```

`UnimplementedUserServiceServer` được embed để implementation tương thích forward-compatible với gRPC generated code. Khi proto thêm RPC mới, server vẫn compile được và method chưa implement sẽ trả `Unimplemented`.

Các RPC đang được implement:

```text
Register(RegisterRequest) -> AuthResponse
Login(LoginRequest)       -> AuthResponse
GetMe(GetMeRequest)       -> UserResponse
```

### Register

Luồng xử lý:

```text
RegisterRequest
-> service.RegisterInput
-> UserService.Register
-> AuthResponse(token, user)
```

`UserService.Register` chịu trách nhiệm:

- trim `name`, `email`, `password`
- lowercase email
- kiểm tra input rỗng
- kiểm tra email đã tồn tại
- hash password bằng bcrypt
- tạo user trong database
- generate JWT token

### Login

Luồng xử lý:

```text
LoginRequest
-> service.LoginInput
-> UserService.Login
-> AuthResponse(token, user)
```

`UserService.Login` chịu trách nhiệm:

- trim và lowercase email
- tìm user theo email
- so sánh password bằng bcrypt
- generate JWT token nếu credential hợp lệ

### GetMe

Luồng xử lý:

```text
GetMeRequest(user_id)
-> userIDFromProto
-> UserService.GetMe
-> UserResponse(user)
```

`userIDFromProto` kiểm tra `uint64` từ protobuf có vượt quá range của `uint` trong Go runtime hiện tại không. Nếu vượt range, gRPC trả `InvalidArgument`.

## 5. Map lỗi user-service sang gRPC status code

File `cmd/user-service/internal/grpc/server.go` có hàm `mapUserError`.

Mapping hiện tại:

```text
ErrInvalidInput        -> InvalidArgument
ErrEmailAlreadyExists  -> AlreadyExists
ErrInvalidCredentials  -> Unauthenticated
ErrUserNotFound        -> NotFound
Lỗi khác               -> Internal
```

Ý nghĩa:

- Client sai input thì nhận `InvalidArgument`.
- Đăng ký email trùng thì nhận `AlreadyExists`.
- Login sai email hoặc password thì nhận `Unauthenticated`.
- User không tồn tại thì nhận `NotFound`.
- Lỗi database hoặc lỗi không xác định được che bằng `Internal`.

## 6. Triển khai gRPC cho todo-service

File chính:

```text
cmd/todo-service/internal/grpc/server.go
```

Struct server:

```go
type TodoGRPCServer struct {
	todopb.UnimplementedTodoServiceServer

	todoService *service.TodoService
}
```

Các RPC đang được implement:

```text
CreateTodo(CreateTodoRequest) -> TodoResponse
ListTodos(ListTodosRequest)   -> ListTodosResponse
GetTodo(GetTodoRequest)       -> TodoResponse
UpdateTodo(UpdateTodoRequest) -> TodoResponse
DeleteTodo(DeleteTodoRequest) -> DeleteTodoResponse
```

### CreateTodo

Luồng xử lý:

```text
CreateTodoRequest(user_id, title)
-> service.CreateTodoInput
-> TodoService.CreateTodo
-> TodoResponse(todo)
```

`TodoService.CreateTodo` trim title, kiểm tra `user_id` và `title`, sau đó tạo todo mới với `completed = false`.

### ListTodos

Luồng xử lý:

```text
ListTodosRequest(user_id)
-> TodoService.ListTodos
-> []TodoOutput
-> repeated Todo
-> ListTodosResponse
```

Kết quả được query theo `user_id` và sort `id DESC`.

Trong gRPC server, slice nội bộ được convert sang protobuf:

```go
protoTodos := make([]*todopb.Todo, 0, len(todos))
for i := range todos {
	protoTodos = append(protoTodos, toProtoTodo(&todos[i]))
}
```

### GetTodo

Luồng xử lý:

```text
GetTodoRequest(user_id, todo_id)
-> TodoService.GetTodo
-> TodoResponse(todo)
```

Service kiểm tra todo có tồn tại không, sau đó kiểm tra todo có thuộc đúng `user_id` không. Nếu todo thuộc user khác thì trả lỗi forbidden.

### UpdateTodo

Luồng xử lý:

```text
UpdateTodoRequest(user_id, todo_id, title, completed)
-> service.UpdateTodoInput
-> TodoService.UpdateTodo
-> TodoResponse(todo)
```

Service kiểm tra input, load todo, kiểm tra quyền sở hữu, cập nhật `title` và `completed`, rồi lưu lại bằng GORM.

Lưu ý: contract hiện tại dùng `optional string title` và `optional bool completed`, nhưng implementation đang đọc bằng `req.GetTitle()` và `req.GetCompleted()`. Cách này biến field không gửi thành zero value (`""` hoặc `false`). Nếu muốn hỗ trợ partial update đúng nghĩa, bước tiếp theo nên kiểm tra presence của optional field trong generated struct.

### DeleteTodo

Luồng xử lý:

```text
DeleteTodoRequest(user_id, todo_id)
-> TodoService.DeleteTodo
-> DeleteTodoResponse(success = true)
```

Service kiểm tra input, load todo, kiểm tra quyền sở hữu, sau đó delete record.

## 7. Map lỗi todo-service sang gRPC status code

File `cmd/todo-service/internal/grpc/server.go` có hàm `mapTodoError`.

Mapping hiện tại:

```text
ErrInvalidInput  -> InvalidArgument
ErrTodoNotFound  -> NotFound
ErrForbidden     -> PermissionDenied
Lỗi khác         -> Internal
```

Ý nghĩa:

- Thiếu `user_id`, `todo_id`, hoặc title không hợp lệ thì nhận `InvalidArgument`.
- Todo không tồn tại thì nhận `NotFound`.
- Todo tồn tại nhưng thuộc user khác thì nhận `PermissionDenied`.
- Lỗi database hoặc lỗi không xác định được che bằng `Internal`.

## 8. Convert model nội bộ sang protobuf

gRPC response không trả trực tiếp model GORM. Thay vào đó code đi qua output struct:

```text
model.User -> service.UserOutput -> userpb.User
model.Todo -> service.TodoOutput -> todopb.Todo
```

User convert:

```go
func toProtoUser(user service.UserOutput) *userpb.User {
	return &userpb.User{
		Id:    uint64(user.ID),
		Email: user.Email,
		Name:  user.Name,
	}
}
```

Todo convert:

```go
func toProtoTodo(todo *service.TodoOutput) *todopb.Todo {
	return &todopb.Todo{
		Id:        uint64(todo.ID),
		UserId:    uint64(todo.UserID),
		Title:     todo.Title,
		Completed: todo.Completed,
	}
}
```

Việc convert riêng giúp boundary giữa database model, business output và protobuf contract rõ ràng hơn.

## 9. Khởi động gRPC server trong main.go

Cả hai service vẫn giữ HTTP server Gin hiện có. gRPC server được start song song bằng goroutine.

User service:

```go
userService := service.NewUserService(database, cfg.JWTSecret)
go startUserGRPCServer(userService)
```

Todo service:

```go
todoService := todoservice.NewTodoService(database)
go startTodoGRPCServer(todoService)
```

Trong hàm start gRPC:

```text
net.Listen
-> grpc.NewServer
-> Register...ServiceServer
-> grpcServer.Serve
```

Ví dụ user-service:

```go
userpb.RegisterUserServiceServer(
	grpcServer,
	grpcserver.NewUserGRPCServer(userService),
)
```

Ví dụ todo-service:

```go
todopb.RegisterTodoServiceServer(
	grpcServer,
	grpcserver.NewTodoGRPCServer(todoService),
)
```

## 10. Port khi chạy bằng Docker Compose dev

HTTP port hiện tại:

```text
gateway      -> 8080
user-service -> 8081
todo-service -> 8082
```

gRPC listener trong code hiện tại:

```text
user-service -> :50051
todo-service -> :50051
```

Hai service chạy trong hai container khác nhau nên cùng listen `:50051` bên trong container không bị xung đột. Nhưng nếu expose ra host để test bằng `grpcurl`, mapping port trong `docker-compose.dev.yml` cần khớp với port mà process trong container đang listen.

Nếu giữ todo-service listen `:50051`, mapping hợp lý từ host thường là:

```yaml
ports:
  - '50052:50051'
```

Nghĩa là:

```text
localhost:50052 trên máy host
-> port 50051 bên trong container todo-service
```

Hoặc có thể đổi code todo-service listen `:50052`, miễn là code và compose thống nhất với nhau.

## 11. Cách verify sau khi code change

Chạy lint và generate lại nếu vừa sửa proto:

```bash
buf lint proto
buf generate proto
```

Chạy test/build Go:

```bash
go test ./...
```

Chạy stack dev:

```bash
make dev
```

Nếu dùng `grpcurl`, cần bật server reflection thì mới list service trực tiếp được. Hiện code chưa bật reflection, nên khi test có thể gọi bằng proto file:

```bash
grpcurl -plaintext \
  -import-path proto \
  -proto user/v1/user.proto \
  -d '{"email":"a@example.com","password":"secret","name":"Alice"}' \
  localhost:50051 \
  user.v1.UserService/Register
```

Ví dụ gọi todo-service từ host nếu map `50052:50051`:

```bash
grpcurl -plaintext \
  -import-path proto \
  -proto todo/v1/todo.proto \
  -d '{"user_id":1,"title":"Learn gRPC"}' \
  localhost:50052 \
  todo.v1.TodoService/CreateTodo
```

## 12. Tóm tắt

Code change gRPC lần này gồm 4 ý chính:

- Contract nằm trong `.proto`, code generate nằm trong `gen/go`.
- Business logic được gom vào service layer để gRPC server không phải xử lý database trực tiếp.
- gRPC server implement generated interface, map protobuf request sang input nội bộ và map output ngược lại.
- Error nội bộ được đổi sang gRPC status code rõ nghĩa cho client.

Sau bước này, `user-service` và `todo-service` đã có thể phục vụ cả HTTP API cũ và gRPC API mới trong cùng một process.
