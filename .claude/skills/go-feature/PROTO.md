# RPC contract

One file per domain under `proto/api/v1/`, package `api.v1`. Follow buf naming rules: service names
end with Service and methods have their own Request/Response messages. Generated Go package paths
come from `buf.gen.yaml`. Put a protovalidate rule on each constrained input field.

```proto
syntax = "proto3";
package api.v1;
import "buf/validate/validate.proto";
service TaskService {
  rpc GetTask(GetTaskRequest) returns (GetTaskResponse) {
    option idempotency_level = NO_SIDE_EFFECTS;
  }
}
message GetTaskRequest {
  string id = 1 [(buf.validate.field).string.uuid = true];
}
message GetTaskResponse { string title = 1; }
```

Use `create_time` / `update_time` for timestamps. Mark reads `NO_SIDE_EFFECTS` so Connect can use
GET. A list that needs paging conventionally takes `page_size`, `page_token` and returns
`next_page_token`; the project chooses its token format. The tree has no paged query example.
Use `bin/tools/buf breaking --against '.git#branch=main'` when changing an existing public contract.
Run `make gen-proto`; implement the generated handler interface without an unimplemented embed.
