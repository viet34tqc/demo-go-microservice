# Lesson 5 - Protobuf, Buf và generate file pb

Mục tiêu của bài này là hiểu vai trò của file `.proto`, cách project mô tả contract bằng Protobuf, vì sao dùng Buf để generate code Go, và các file `pb.go` sinh ra dùng để làm gì.

Sau bài này bạn sẽ hiểu:

- Vì sao project cần thư mục `proto/`.
- `service`, `message`, `go_package` trong file `.proto` có ý nghĩa gì.
- Vì sao project dùng `buf` thay vì gọi `protoc` thủ công.
- `buf generate proto` sinh ra những file nào trong `gen/go`.
- Khi sửa `proto`, ta cần generate lại code ở bước nào.

## 1. Vì sao cần Protobuf trong project này?

Khi hệ thống có nhiều service, ta cần một cách mô tả request và response sao cho rõ ràng, có version, và dễ generate code cho nhiều ngôn ngữ.

Trong project này, phần contract được đặt ở:

```text
proto/user/v1/user.proto
proto/todo/v1/todo.proto
```

Thay vì chỉ mô tả API bằng text hoặc tự viết struct bằng tay ở từng service, ta định nghĩa contract một lần trong file `.proto`, sau đó generate code dùng lại.

Ý tưởng chính:

```text
.proto
-> Buf generate
-> gen/go/.../*.pb.go
-> service import code đã generate
```

## 2. Cấu trúc thư mục proto

Project đang tổ chức proto như sau:

```text
proto
├── buf.yaml
├── todo
│   └── v1
│       └── todo.proto
└── user
    └── v1
        └── user.proto
```

Mỗi domain có thư mục riêng:

- `user`: contract liên quan đến user.
- `todo`: contract liên quan đến todo.

Mỗi domain dùng version `v1`. Cách đặt version trong path giúp sau này có thể mở rộng sang `v2` mà không phá vỡ code cũ.

## 3. File `.proto` mô tả điều gì?

Ví dụ file: `proto/todo/v1/todo.proto`

```proto
syntax = "proto3";

package todo.v1;

import "google/protobuf/timestamp.proto";

option go_package = "github.com/viet34tqc/demo-go-microservice/gen/go/todo/v1;todov1";
```

Ý nghĩa từng phần:

- `syntax = "proto3"`: dùng cú pháp protobuf version 3.
- `package todo.v1`: namespace logic của contract protobuf.
- `import "google/protobuf/timestamp.proto"`: dùng kiểu thời gian chuẩn của protobuf.
- `option go_package = "...;todov1"`: chỉ ra package import phía Go sau khi generate.

`go_package` rất quan trọng. Nó quyết định code Go sinh ra sẽ thuộc package nào và được import bằng path nào.

Ví dụ trong project này:

```text
github.com/viet34tqc/demo-go-microservice/gen/go/todo/v1
github.com/viet34tqc/demo-go-microservice/gen/go/user/v1
```

## 4. `service` và `message`

Trong protobuf, `message` là cấu trúc dữ liệu. `service` là nhóm các RPC method.

Ví dụ:

```proto
service TodoService {
  rpc CreateTodo(CreateTodoRequest) returns (TodoResponse);
  rpc ListTodos(ListTodosRequest) returns (ListTodosResponse);
  rpc GetTodo(GetTodoRequest) returns (TodoResponse);
  rpc UpdateTodo(UpdateTodoRequest) returns (TodoResponse);
  rpc DeleteTodo(DeleteTodoRequest) returns (DeleteTodoResponse);
}
```

Đoạn này nói rằng `TodoService` có 5 method. Mỗi method nhận vào một `message` request và trả về một `message` response.

Ví dụ một `message`:

```proto
message Todo {
  uint64 id = 1;
  uint64 user_id = 2;
  string title = 3;
  bool completed = 4;
  google.protobuf.Timestamp created_at = 5;
  google.protobuf.Timestamp updated_at = 6;
}
```

Ở đây:

- `uint64`, `string`, `bool` là kiểu dữ liệu protobuf cơ bản.
- `created_at` và `updated_at` dùng `google.protobuf.Timestamp`.
- Các số `= 1`, `= 2`, `= 3` là field number.

Field number là định danh nhị phân của field trong protobuf. Khi contract đã public thì không nên đổi bừa các số này, vì nó ảnh hưởng tới compatibility.

## 5. Vì sao `UpdateTodoRequest` dùng `optional`?

Trong file `todo.proto`:

```proto
message UpdateTodoRequest {
  uint64 user_id = 1;
  uint64 todo_id = 2;
  optional string title = 3;
  optional bool completed = 4;
}
```

Route update thường là partial update. User có thể chỉ muốn đổi `title`, hoặc chỉ muốn đổi `completed`.

`optional` giúp phân biệt:

- field không được gửi lên
- field được gửi lên với giá trị rỗng hoặc `false`

Sau khi generate Go code, các field optional thường trở thành pointer:

```go
Title     *string
Completed *bool
```

Nhờ vậy phía Go biết được field có thật sự xuất hiện trong request hay không.

## 6. Buf đang được cấu hình thế nào?

Project có 2 file Buf quan trọng:

- [proto/buf.yaml](/home/viet/apps/go-microservice/proto/buf.yaml:1)
- [buf.gen.yaml](/home/viet/apps/go-microservice/buf.gen.yaml:1)

`proto/buf.yaml` là cấu hình module và rule:

```yaml
version: v2

lint:
  use:
    - STANDARD

breaking:
  use:
    - FILE
```

Ý nghĩa:

- `lint`: kiểm tra style và rule chuẩn của protobuf.
- `breaking`: giúp phát hiện thay đổi phá vỡ compatibility khi contract tiến hóa.

`buf.gen.yaml` là cấu hình generate:

```yaml
version: v2

plugins:
  - remote: buf.build/protocolbuffers/go
    out: gen/go
    opt:
      - paths=source_relative

  - remote: buf.build/grpc/go
    out: gen/go
    opt:
      - paths=source_relative
```

Ở đây project đang generate bằng 2 plugin:

- `protocolbuffers/go`: sinh file message protobuf.
- `grpc/go`: sinh file service gRPC.

## 7. Vì sao dùng Buf thay vì gọi `protoc` thủ công?

Với repo này, dùng Buf tiện hơn vì:

- Config generate nằm trong file, không cần nhớ command dài.
- Có luôn lint và breaking check.
- Dùng remote plugin nên không cần tự quản lý `protoc-gen-go` và `protoc-gen-go-grpc` trong máy.
- Dễ đồng bộ giữa các máy dev và CI.

Nếu dùng `protoc` trực tiếp, command sẽ dài hơn và dễ sai option:

```bash
protoc -I proto \
  --go_out=gen/go --go_opt=paths=source_relative \
  --go-grpc_out=gen/go --go-grpc_opt=paths=source_relative \
  proto/todo/v1/todo.proto proto/user/v1/user.proto
```

Trong khi với Buf, command ngắn hơn:

```bash
buf generate proto
```

## 8. Generate xong sẽ có gì?

Sau khi generate, project có các file:

```text
gen/go/user/v1/user.pb.go
gen/go/user/v1/user_grpc.pb.go
gen/go/todo/v1/todo.pb.go
gen/go/todo/v1/todo_grpc.pb.go
```

Ý nghĩa:

- `*.pb.go`: struct message, metadata protobuf, helper methods.
- `*_grpc.pb.go`: interface client/server cho gRPC service.

Ví dụ trong `gen/go/todo/v1/todo.pb.go`, `Todo` được generate thành struct Go.

Ví dụ trong `gen/go/todo/v1/todo_grpc.pb.go`, `TodoServiceClient` và `TodoServiceServer` được generate sẵn.

Điểm này rất quan trọng: file `.proto` là nguồn sự thật, còn file trong `gen/go` là code sinh ra từ nguồn đó.

## 9. Khi nào cần generate lại?

Mỗi khi bạn sửa một trong các phần sau trong `.proto`, nên generate lại:

- thêm `message`
- sửa field trong `message`
- thêm `service`
- thêm `rpc`
- đổi kiểu dữ liệu
- thêm `optional`

Quy trình thường là:

```bash
buf lint proto
buf generate proto
go mod tidy
```

`go mod tidy` giúp cập nhật dependency nếu code generate mới cần thêm runtime package.

## 10. Có cần `go get google.golang.org/protobuf` và `google.golang.org/grpc` không?

Không cần `go get` hai package này chỉ để Buf chạy.

Buf chỉ lo phần generate. Nhưng code Go sau khi generate sẽ import runtime package:

- `google.golang.org/protobuf`
- `google.golang.org/grpc`

Vì vậy khi build project, `go.mod` vẫn cần có chúng. Cách làm gọn nhất thường là:

```bash
buf generate proto
go mod tidy
```

## 11. Luồng làm việc đề xuất trong project này

Khi thêm API mới cho user hoặc todo, nên đi theo thứ tự:

```text
1. Sửa hoặc thêm file .proto
2. buf lint proto
3. buf generate proto
4. Import code trong gen/go vào service
5. Viết handler hoặc gRPC implementation thật
```

Làm như vậy giúp contract đi trước implementation. Team sẽ dễ review hơn vì nhìn vào `.proto` là hiểu service đang public dữ liệu gì.

## 12. Tóm tắt

Trong project này:

- Thư mục `proto/` giữ contract của hệ thống.
- Buf là công cụ chính để lint và generate code protobuf.
- `gen/go/` chứa code Go sinh ra từ `.proto`.
- `go_package` quyết định package import phía Go.
- `optional` hữu ích cho các request update partial.

Khi đã quen với cách làm này, bước tiếp theo hợp lý là dùng các contract trong `gen/go` để kết nối service qua gRPC thay vì chỉ đi qua HTTP JSON như các bài trước.
