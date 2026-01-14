## 2025-05-23 - Avoid fmt.Fprint in Tight Loops
**Learning:** Replacing `fmt.Fprint(w, str)` with `io.WriteString(w, str)` and `fmt.Fprint(w, "\t")` with `w.Write(tabBytes)` in tight loops (like table rendering) can yield significant performance improvements (~20% faster). `fmt`'s reflection overhead is costly per call.
**Action:** Prefer `io.WriteString` or `w.Write` for static strings inside loops. Pre-allocate byte slices for common separators like tabs and newlines.
