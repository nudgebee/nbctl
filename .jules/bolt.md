## 2024-05-22 - [Stream vs Buffer in GraphQL Client]
**Learning:** `pkg/client` was double-allocating response bodies (once in `io.ReadAll`, once in `json.Unmarshal`). For large GraphQL responses (common in this app), this causes significant GC pressure.
**Action:** Always prefer `json.NewDecoder(r).Decode(&v)` over `io.ReadAll` + `json.Unmarshal` for API responses, unless the raw body is needed for signature verification or logging.

## 2024-05-22 - [Optimizing CLI Output Formatting]
**Learning:** In `pkg/format`, replacing `fmt.Fprint(w, "\t")` and `fmt.Fprintln(w)` with `w.Write(tabBytes)` and `w.Write(newlineBytes)` inside loops yielded a ~20% speedup. Also, using `reflect.Value.Bytes()` for `json.RawMessage` fields avoids interface boxing allocations.
**Action:** For tight loops writing to `io.Writer`, prefer `w.Write` or `io.WriteString` over `fmt.Fprint`, and access underlying bytes of slice types directly via reflection when possible.
