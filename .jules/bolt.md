## 2025-05-23 - [Direct Writer vs String Concatenation]
**Learning:** Constructing intermediate string slices and joining them in loops (e.g., for table rows) generates significant garbage collection pressure.
**Action:** When writing formatted output (like tables), prefer writing directly to the `io.Writer` (e.g., using `fmt.Fprint`) instead of building string buffers, especially inside loops.
