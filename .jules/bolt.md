## 2024-05-22 - [Stream vs Buffer in GraphQL Client]
**Learning:** `pkg/client` was double-allocating response bodies (once in `io.ReadAll`, once in `json.Unmarshal`). For large GraphQL responses (common in this app), this causes significant GC pressure.
**Action:** Always prefer `json.NewDecoder(r).Decode(&v)` over `io.ReadAll` + `json.Unmarshal` for API responses, unless the raw body is needed for signature verification or logging.

## 2024-05-22 - [fmt.Fprint Overhead in Tight Loops]
**Learning:** Replacing `fmt.Fprint(w, "\t")` and `fmt.Fprintln(w)` with direct `w.Write` calls using pre-allocated byte slices improved table formatting throughput by ~22%. `fmt`'s reflection and parsing overhead is significant in hot loops, even for constant strings.
**Action:** For tight loops writing static separators (tabs, newlines), use `io.Writer.Write` with package-level byte slices instead of `fmt` functions.
