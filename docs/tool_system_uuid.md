# system_uuid Tool

The `system_uuid` tool generates a cryptographically secure UUID version 4 identifier.

## Overview

The tool provides a unique identifier when the agent needs to create an ID for a document, operation, or other resource.

## Features

- **UUID v4**: Sets the version and RFC 4122 variant bits required by the standard
- **Cryptographically secure**: Uses Go's `crypto/rand` through `github.com/google/uuid`
- **No Parameters**: The tool takes no input parameters
- **Standard format**: Returns a lowercase, hyphenated UUID string in the 8-4-4-4-12 format

## Usage

The tool is invoked by the agent when a UUID is needed. It has no parameters:

```json
{}
```

## Output Format

The tool returns a 36-character UUID v4 string, for example:

```text
550e8400-e29b-41d4-a716-446655440000
```

## Errors

If the secure random source fails, the tool returns a wrapped error. It does not return an empty or fabricated identifier.

## Example

The agent calls `system_uuid` with no parameters:

```json
{
  "tool": "system_uuid"
}
```

The result is a UUID v4 string, such as:

```text
550e8400-e29b-41d4-a716-446655440000
```
