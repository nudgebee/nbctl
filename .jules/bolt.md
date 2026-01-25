## 2024-05-22 - [Stream vs Buffer in GraphQL Client]
**Learning:** `pkg/client` was double-allocating response bodies (once in `io.ReadAll`, once in `json.Unmarshal`). For large GraphQL responses (common in this app), this causes significant GC pressure.
**Action:** Always prefer `json.NewDecoder(r).Decode(&v)` over `io.ReadAll` + `json.Unmarshal` for API responses, unless the raw body is needed for signature verification or logging.

## 2024-05-23 - [Reflection Overhead in Tight Loops]
**Learning:** `reflect.Type.Implements` is expensive when called repeatedly in tight loops (e.g., iterating over table rows).
**Action:** Pre-calculate interface implementation status for fields outside the loop and pass it to the printer function. This yielded ~50% speedup for tabular output.
