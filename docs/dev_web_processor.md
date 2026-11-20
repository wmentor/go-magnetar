# Web Processor Package

The `internal/http/processor` package provides a store for web processors that handle URL processing and content extraction.

## Overview

The `processor` package implements a thread-safe store for registering and executing web processor functions. It's designed to work with the `plugin.WebProcessorFunc` interface and supports:

- **Thread-safe operations**: Uses read-write mutexes for concurrent access
- **Chained processing**: Supports multiple processors that are executed sequentially
- **Error handling**: Properly handles and wraps errors
- **Skip behavior**: Supports `plugin.ErrNoWebProcessor` to skip processors

## Usage

### Creating a Store

```go
store := processor.NewStore()
```

### Adding Processors

```go
store.AddProcessorFunc(func(ctx context.Context, url string) (string, error) {
    // Process the URL and return content
    content, err := fetchContent(url)
    if err != nil {
        return "", fmt.Errorf("failed to fetch: %w", err)
    }
    return content, nil
})
```

### Processing URLs

```go
ctx := context.Background()
result, err := store.Process(ctx, "https://example.com")
if err != nil {
    log.Fatalf("Failed to process URL: %v", err)
}
fmt.Println(result)
```

## Thread Safety

The store is designed for concurrent use:

- `AddProcessorFunc` uses a write lock
- `Process` uses a read lock
- Multiple goroutines can safely call these methods concurrently

## Processor Chain

Processors are executed in the order they were added:

1. Each processor is called sequentially
2. If a processor returns `plugin.ErrNoWebProcessor`, it's skipped
3. If a processor returns an error, it's wrapped and returned
4. If a processor returns success, the result is returned immediately

## Testing

The package includes comprehensive tests covering:

- Basic store operations
- Concurrent access
- Context cancellation
- Error handling
- Multiple processor chaining
- Various URL formats
- Real HTTP requests

Run tests with:

```bash
go test ./internal/http/processor/...
```

Check coverage:

```bash
go test ./internal/http/processor/... -cover
```

## Implementation Details

### Store Structure

```go
type store struct {
    mtx        sync.RWMutex
    processors []plugin.WebProcessorFunc
}
```

- `mtx`: Read-write mutex for thread safety
- `processors`: Slice of registered processor functions

### Default Allocation Size

The store is pre-allocated with capacity for 8 processors by default.

```go
const defaultAllocSize = 8
```
