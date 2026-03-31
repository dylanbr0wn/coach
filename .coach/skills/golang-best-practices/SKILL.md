---
name: golang-best-practices
description: Enforces Go best practices for naming, error handling, concurrency, project layout, testing, and API design. Distilled from Effective Go, Google Go Style Guide, Twelve Go Best Practices, and golang-standards/project-layout.
allowed-tools:
  - Read
  - Write
  - Bash
---

# Go Best Practices

## When to Use

Use this skill when:
- Writing new Go code (files, packages, functions, types)
- Reviewing or refactoring existing Go code
- Setting up a new Go project or restructuring an existing one
- Designing Go APIs, interfaces, or package boundaries
- Writing or improving Go tests
- Debugging concurrency issues or designing concurrent systems
- Making decisions about error handling strategies

## Instructions

### 1. Project Layout

Follow the standard Go project layout based on project size:

**Small projects** (libraries, simple CLIs): a single `main.go` + `go.mod` is fine. Do not over-structure.

**Growing projects** should adopt this structure as needed:

| Directory | Purpose |
|-----------|---------|
| `cmd/` | Main applications. Each subdirectory = one executable. Keep `main.go` minimal — import and call code from `internal/` or `pkg/`. |
| `internal/` | Private code the Go compiler prevents external imports of. Use `internal/app/` for application logic, `internal/pkg/` for shared private libraries. |
| `pkg/` | Public library code safe for external consumption. Only use when you intentionally want others to import it. |
| `api/` | OpenAPI specs, protobuf definitions, JSON schemas. |
| `configs/` | Configuration file templates and defaults. |
| `scripts/` | Build, install, and analysis scripts. |
| `build/` | Packaging (`build/package/`) and CI (`build/ci/`) configs. |
| `deployments/` | Docker, Kubernetes, Helm, Terraform configs. |
| `test/` | External test apps and test data. Go ignores dirs starting with `.` or `_`. |
| `docs/` | Design documents and user guides. |
| `tools/` | Supporting tools for the project. |
| `examples/` | Usage examples for libraries. |

**Never create a `/src` directory.** This is a Java convention that conflicts with Go's workspace structure.

### 2. Naming

**Packages:**
- Use lowercase, single-word names. No underscores or mixedCaps.
- Name packages after what they provide, not what they contain.
- Never name a package `util`, `helper`, `common`, or `base` — these are uninformative and cause import conflicts.

**Exported identifiers:**
- Visibility is controlled by capitalization: `Uppercase` = exported, `lowercase` = unexported.
- Avoid repeating the package name: `json.Encoder` not `json.JSONEncoder`.
- No `Get` prefix on getters: `Owner()` not `GetOwner()`. Setters use `Set` prefix: `SetOwner()`.
- Interfaces with one method use the method name + `-er`: `Reader`, `Writer`, `Stringer`.

**Functions and methods:**
- Do not repeat the receiver type, parameter types, or return types in the name.
- Functions returning a value get noun-like names: `JobName()`.
- Functions performing an action get verb-like names: `WriteDetail()`.
- Use `MixedCaps` or `mixedCaps`, never underscores.
- Shorter is better: `MarshalIndent` not `MarshalWithIndentation`.

**Variables:**
- Do not shadow standard package names (e.g., avoid `url := "..."` in functions that need `net/url`).
- When reassigning an outer-scope variable inside a block, use `=` not `:=` to avoid creating a new shadowed variable.

### 3. Error Handling

**Return errors early to avoid nesting:**

```go
// Good: handle error first, keep happy path unindented
f, err := os.Open(name)
if err != nil {
    return err
}
defer f.Close()
// ... use f
```

```go
// Bad: deeply nested success path
f, err := os.Open(name)
if err == nil {
    // nested code...
    if err == nil {
        // more nesting...
    }
}
```

**Error message conventions:**
- Prefix with operation or package name: `"image: unknown format"`.
- Use `fmt.Errorf("doing X: %w", err)` to wrap errors with context.
- Custom error types should implement `Error() string` and include enough context for diagnosis.

**Custom error types for rich context:**

```go
type PathError struct {
    Op   string
    Path string
    Err  error
}

func (e *PathError) Error() string {
    return e.Op + " " + e.Path + ": " + e.Err.Error()
}
```

**Error handling patterns:**
- Use the comma-ok idiom: `val, ok := cache[key]`.
- Use `errors.Is()` and `errors.As()` to check wrapped errors.
- Create helper types to reduce repetitive error-checking (e.g., a `binWriter` that tracks the first error).
- Use `panic` only for truly unrecoverable errors. Libraries should return errors, not panic.
- Use `recover` in deferred functions to convert panics to errors at API boundaries (e.g., HTTP handlers).

### 4. Functions and Methods

**Multiple return values:**
- Return `(result, error)` pairs. Never use out-parameters.
- Use named return values for documentation, but avoid bare returns in long functions.

**Defer for cleanup:**
- Always `defer` resource cleanup immediately after acquiring the resource.
- Deferred calls execute in LIFO order.
- Arguments to deferred calls are evaluated at defer time, not at execution time.

```go
f, err := os.Open(name)
if err != nil {
    return err
}
defer f.Close()
```

**Receivers — pointer vs value:**
- Use pointer receivers when the method modifies the receiver or the struct is large.
- Use value receivers for small, immutable types.
- Be consistent: if one method needs a pointer receiver, make all methods pointer receivers.

**Function adapters for cross-cutting concerns:**

```go
func errorHandler(f func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if err := f(w, r); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            log.Printf("handling %q: %v", r.RequestURI, err)
        }
    }
}
```

### 5. Interfaces

**Design for behavior, not types:**
- Accept interfaces, return concrete types.
- Depend on the smallest interface you need: accept `io.Writer` not `*os.File`.
- One- or two-method interfaces are preferred for flexibility.

**Keep independent packages independent:**
- Use interfaces to decouple packages rather than importing concrete types across package boundaries.

**Compile-time interface checks:**

```go
var _ json.Marshaler = (*MyType)(nil)
```

**Embedding for composition:**
- Embed interfaces in interfaces to compose behavior: `ReadWriter` embeds `Reader` + `Writer`.
- Embed structs in structs to promote methods — but the receiver remains the inner type.

### 6. Concurrency

> "Do not communicate by sharing memory; share memory by communicating."

**Expose synchronous APIs.** Let callers decide on concurrency:

```go
// Good: synchronous function
func Process(job Job) error { ... }

// Caller adds concurrency as needed
go func() { errc <- Process(job) }()
```

**Use goroutines to manage state:**

```go
type Server struct{ quit chan bool }

func (s *Server) run() {
    for {
        select {
        case <-s.quit:
            fmt.Println("shutting down")
            s.quit <- true
            return
        case <-time.After(time.Second):
            fmt.Println("working")
        }
    }
}
```

**Prevent goroutine leaks:**
- Unbuffered channels can block goroutines forever if no one reads/writes.
- Use buffered channels when the number of senders is known.
- Use a `quit`/`done` channel or `context.Context` for cancellation when the count is unknown.
- Always ensure every goroutine has a path to termination.

**Channel patterns:**
- Use `for range` on channels to consume until closed.
- Use `select` with `default` for non-blocking operations.
- Use buffered channels as semaphores to limit concurrency.

**Parallelization:**
- Use `runtime.NumCPU()` or `runtime.GOMAXPROCS(0)` to size worker pools.
- Fan-out work to goroutines, collect results via a shared channel.

### 7. Testing

**Use interfaces to enable test doubles:**

```go
type Store interface {
    Get(key string) (string, error)
}

// In tests, provide a stub:
type stubStore struct{ data map[string]string }
func (s *stubStore) Get(key string) (string, error) {
    v, ok := s.data[key]
    if !ok { return "", errors.New("not found") }
    return v, nil
}
```

**Test double naming:**
- Put test helpers in a `*test` package: `creditcardtest` for `creditcard`.
- Single double = simple name: `creditcardtest.Stub`.
- Multiple behaviors = descriptive names: `AlwaysCharges`, `AlwaysDeclines`.

**Test variable naming:**
- Prefix test doubles to distinguish from production: `spyCC` not `cc`.
- Use `got, want` pattern for assertions.

**Error messages:**
- Format as: `t.Errorf("Func(args) = %v, want %v", got, want)` — show the call, actual result, and expected result.

**Table-driven tests:**

```go
tests := []struct {
    name  string
    input string
    want  string
}{
    {"empty", "", ""},
    {"hello", "hello", "HELLO"},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got := Transform(tt.input)
        if got != tt.want {
            t.Errorf("Transform(%q) = %q, want %q", tt.input, got, tt.want)
        }
    })
}
```

### 8. Code Organization

**File ordering within a package:**
1. License header and build tags
2. Package documentation comment
3. `package` declaration
4. Import statements (standard library group, then third-party group, separated by blank line)
5. Most significant types and their methods first
6. Helper types and functions last

**File splitting:**
- Avoid very long files. Split by responsibility.
- Separate code and tests: `handler.go` and `handler_test.go`.
- Use `doc.go` for package-level documentation in multi-file packages.

**Make packages `go get`-able:**
- Separate reusable library code from executables using `cmd/` for binaries.

### 9. Data Structures

**Slices over arrays.** Arrays are values (copying semantics) and size is part of the type. Slices are references and far more flexible.

**Design for useful zero values.** Types like `sync.Mutex` and `bytes.Buffer` work correctly without explicit initialization. Strive for this in your own types.

**Composite literals:**

```go
// Preferred: named fields, any order, missing = zero value
return &Config{
    Timeout: 30 * time.Second,
    Retries: 3,
}
```

**Maps:**
- Use comma-ok idiom to distinguish missing keys from zero values: `v, ok := m[key]`.
- Use `delete(m, key)` to remove entries.
- A `map[string]bool` is a simple set implementation.

### 10. Documentation

- Document all exported identifiers.
- Start comments with the identifier name: `// Reader reads from an underlying source.`
- Package comments go above the `package` declaration; use `doc.go` for long descriptions.
- Use `godoc` conventions — comments are rendered as plain text with blank-line-separated paragraphs.

### 11. Formatting

- Always use `gofmt` (or `goimports`). Do not fight the formatter.
- Use tabs for indentation.
- No line length limit, but wrap long lines and indent continuations with an extra tab.
- No parentheses in `if`, `for`, `switch` conditions.
- Opening brace on the same line as the control structure — never on the next line.

## Constraints

- Do not add a `/src` directory to Go projects.
- Do not use `Get` prefix on getter methods.
- Do not name packages `util`, `helper`, `common`, or `base`.
- Do not expose concurrent APIs when a synchronous design suffices — let callers add concurrency.
- Do not use `panic` in library code for recoverable errors — return errors instead.
- Do not use bare returns in functions longer than a few lines.
- Do not skip `gofmt` — all Go code must be formatted.
- Do not create goroutines without a clear termination path.
- Do not import concrete types across package boundaries when an interface would decouple them.
- Do not repeat the package name in exported identifiers.
