## 2024-05-22 - [Stream vs Buffer in GraphQL Client]
**Learning:** `pkg/client` was double-allocating response bodies (once in `io.ReadAll`, once in `json.Unmarshal`). For large GraphQL responses (common in this app), this causes significant GC pressure.
**Action:** Always prefer `json.NewDecoder(r).Decode(&v)` over `io.ReadAll` + `json.Unmarshal` for API responses, unless the raw body is needed for signature verification or logging.

## 2025-05-23 - [Reflection Overhead & Safety in Formatting]
**Learning:** Repeatedly calling `reflect.Type.Implements` in tight loops (like table rendering) caused 28% overhead. Worse, relying on `v.Interface()` fallback for unhandled types caused panics for structs with unexported fields (e.g., `time.Time`).
**Action:** Pre-calculate interface implementation status (like `fmt.Stringer`) for struct fields outside of data loops. This optimizes performance and allows early dispatch to safe formatters (like `fmt`), avoiding dangerous reflection on unexported fields.
