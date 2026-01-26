## 2024-05-22 - [Stream vs Buffer in GraphQL Client]
**Learning:** `pkg/client` was double-allocating response bodies (once in `io.ReadAll`, once in `json.Unmarshal`). For large GraphQL responses (common in this app), this causes significant GC pressure.
**Action:** Always prefer `json.NewDecoder(r).Decode(&v)` over `io.ReadAll` + `json.Unmarshal` for API responses, unless the raw body is needed for signature verification or logging.

## 2025-05-23 - [Reflection Interface Checks Cache]
**Learning:** Repeatedly calling `reflect.Type.Implements` in a tight loop (e.g., printing table rows) has measurable CPU overhead (~13% in benchmarks).
**Action:** Pre-calculate interface satisfaction for struct fields once before iterating over rows, and pass the result to the formatting function.
