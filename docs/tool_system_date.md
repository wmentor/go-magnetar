# system_date Tool

The `system_date` tool returns the current system date in the "Mon Jan _2 15:04:05 MST 2006" format.

## Overview

The tool provides the current date to the LLM when it needs to know the current date for time-sensitive queries or to provide context in responses.

## Features

- **Current Date**: Returns the exact current date in Unix date format
- **No Parameters**: The tool takes no parameters and always returns the current date
- **Format**: Uses Go's `time.UnixDate` format

## Usage

The tool is invoked by the LLM when it needs to know the current date. The tool has no parameters:

```json
{}
```

## Output Format

The tool returns the current date in the "Mon Jan _2 15:04:05 MST 2006" format:

```
Mon Oct 26 14:30:45 MSK 2026
```

## Implementation Details

The tool uses Go's `time.Now().Format(time.UnixDate)` to get the current date. This ensures consistent behavior across all platforms and time zones.

## Examples

### Getting current date

When the LLM needs to know the current date for a query like "What day is today?", it will call:

```json
{
  "tool": "system_date"
}
```

And receive a response like:

```
Mon Oct 26 14:30:45 MSK 2026
```
