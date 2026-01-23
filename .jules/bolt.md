## 2024-05-22 - [Stream vs Buffer in GraphQL Client]
**Learning:** `pkg/client` was double-allocating response bodies (once in `io.ReadAll`, once in `json.Unmarshal`). For large GraphQL responses (common in this app), this causes significant GC pressure.
**Action:** Always prefer `json.NewDecoder(r).Decode(&v)` over `io.ReadAll` + `json.Unmarshal` for API responses, unless the raw body is needed for signature verification or logging.

## 2024-05-22 - [Reflect Type Check Hoisting]
**Learning:** Repeatedly calling `reflect.Value.Type().Implements()` in tight loops (e.g. table rendering) is expensive.
**Action:** Pre-calculate `isStringerOrError` (or other interface checks) based on the static `reflect.Type` of struct fields outside the loop. Pass this boolean to the inner writer function. This reduces CPU time by ~15-20% for large tables.
