## 2024-05-22 - [Stream vs Buffer in GraphQL Client]
**Learning:** `pkg/client` was double-allocating response bodies (once in `io.ReadAll`, once in `json.Unmarshal`). For large GraphQL responses (common in this app), this causes significant GC pressure.
**Action:** Always prefer `json.NewDecoder(r).Decode(&v)` over `io.ReadAll` + `json.Unmarshal` for API responses, unless the raw body is needed for signature verification or logging.

## 2024-05-22 - [Reflection Overhead in Formatting Loops]
**Learning:** Repeatedly calling `reflect.Type.Implements()` inside tight loops (like iterating over table rows) adds significant CPU overhead.
**Action:** Pre-calculate interface implementation status for struct fields outside the loop and pass it down to the writer function. This yielded a ~19% performance improvement in `pkg/format`.
