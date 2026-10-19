# Chat Agent `Ask` Method

## Overview

The `Ask` method is the core of the chat agent's interactive REPL. It sends user input to the LLM, handles tool calls, manages tool call limits, and prevents infinite loops.

## Method Signature

```go
func (a *ChatAgent) Ask(userInput string) (string, error)
```

## Parameters

- `userInput` (string): The user's message or question

## Returns

- `string`: The final answer from the LLM
- `error`: Any error that occurred during processing

## Algorithm

### 1. Initial Setup (lines 251-275)

1. Append user input to message history (line 252-255)
2. Compact message history if needed (line 257)
3. Build tool list and dispatch map from registered plugins (lines 260-267)
4. Initialize counters:
   - `toolCallCount` (atomic): tracks search tool calls per user request
   - `noProgressCount`: counts iterations without tool calls
   - `allInterationRealToolCall` (atomic): counts real tool calls in current iteration
5. Initialize `alreadyDoneToolCalls` map to prevent duplicate tool calls
6. Initialize `lastAnswerContent` to preserve content on loop break

### 2. Main Loop (line 277)

Executes up to `maxAskLoopIteration` (50) iterations:

#### 2.1 Context Preparation (lines 278-282)

- Calculate reserved tokens for output
- Trim messages to fit context window

#### 2.2 LLM Request (lines 289-298)

Send request to OpenAI with:
- Current message history
- All available tools
- Configurable temperature, top_p, reasoning_effort

#### 2.3 Response Handling

##### Case A: LLM returns text (no tools) - line 413

- Return `choice.Message.Content` immediately

##### Case B: LLM requests tool calls - lines 315-410

**Tool Call Processing (lines 328-381):**

1. For each tool call in the response:
   - Create `toolKey` from tool name and arguments (line 329)
   - Check if this tool+args combination was already called in this `Ask` (lines 330-343):
     - **Already called**: Return error to LLM, skip execution, warn user
     - **New tool call**: 
       - Add to `alreadyDoneToolCalls` (line 339)
       - Increment `allInterationRealToolCall` IF search limit not reached (lines 340-342)

2. Split tool calls into two buckets:
   - **Parallel tools** (`Sequential: false`): read-only or network tools — executed concurrently via goroutines.
   - **Sequential tools** (`Sequential: true`): file-mutating tools — executed one at a time after all parallel tools finish.

3. Execute **parallel** tools concurrently:
    - **Search tools** (`IsSearchTool: true`, e.g. search, web_fetch):
      - Check if `toolCallCount >= maxSearchToolCalls` (20)
      - **Limit reached**: Return error, do NOT execute
      - **Limit not reached**:
        - Increment `toolCallCount`
        - Execute tool, return result or error
    - **Other parallel tools** (github_repo, cve, gitlab_fetch_mr, etc.):
      - Execute without search limit check, return result or error

4. Wait for all parallel tools to complete.

5. Execute **sequential** tools one at a time, in the order the LLM listed them:
   - Each tool sees the fully up-to-date filesystem after the previous one finished.
   - Same search-limit and error handling as parallel tools.

6. Append all tool results to message history.

**Progress Tracking (lines 397-401):**

- If `allInterationRealToolCall == 0`: `noProgressCount++`
- If `allInterationRealToolCall > 0`: `noProgressCount = 0`

**Loop Protection (lines 403-408):**

- If `noProgressCount >= noProgressCountMax` (4): 
  - Warn about infinite loop
  - Return last answer content
- Otherwise: `continue` to next iteration

### 3. Maximum Iteration Limit (lines 414-417)

If all 50 iterations are exhausted:
- Warn about infinite loop
- Return last answer content

## Constants

| Constant | Value | Description |
|----------|-------|-------------|
| `maxSearchToolCalls` | 20 | Maximum search tool calls per user request |
| `noProgressCountMax` | 4 | Maximum iterations without progress before breaking |
| `maxAskLoopIteration` | 50 | Maximum iterations per Ask call |
| `reservedOutputFraction` | 5 | 1/5 of context reserved for output |

## Protection Mechanisms

### 1. Duplicate Tool Call Detection

```go
alreadyDoneToolCalls := map[toolRecord]struct{}{}
```

Prevents the LLM from calling the same tool with the same arguments twice in one request.

### 2. Search Tool Call Limit

```go
if atomic.LoadInt64(toolCallCount) >= maxSearchToolCalls {
    result = "error: reached maximum number of search tool calls..."
} else {
    // execute tool
}
```

- Tracks search tool calls (`search`, `web_fetch`, `jira_task_search`)
- Returns error for additional calls (no execution)
- `toolCallCount` is reset for each new user request (local variable in `Ask`)

### 3. Progress Tracking

```go
if atomic.LoadInt64(allInterationRealToolCall) == 0 {
    noProgressCount++
} else {
    noProgressCount = 0
}
```

- Counts iterations without tool calls
- Breaks after 4 consecutive iterations without progress
- Search tool calls exceeding limit still increment `allInterationRealToolCall`

### 4. Sequential Execution for File-Mutating Tools

```go
// LLMTool.Sequential: true
```

Tools marked `Sequential: true` (`file_write`, `file_edit`, `file_edit_multiple`) are never run in parallel with each other or with other tools in the same LLM response batch. They execute strictly one after another, in the order the LLM listed them, after all parallel tools in the same batch have completed. This prevents lost-write races where two concurrent edits both read the original file and the second one overwrites the first.

### 5. Maximum Iteration Limit

```go
for range maxAskLoopIteration { ... }
```

Breaks after 50 iterations regardless of other conditions.

## Tool Categories

### Search Tools (`IsSearchTool: true`)
These tools count toward the per-request search call limit and run in parallel.
- `search`: Unified search tool (calls web_search and rag_search in parallel)
- `web_fetch`: Fetch web pages (including Confluence, JIRA, GitHub)
- `jira_task_search`: Search JIRA issues by JQL

### Lookup Tools (`IsSearchTool: false`, `Sequential: false`)
Read-only or network tools that run in parallel without a search limit.
- `file_read`: Read a local file
- `file_list`: List local files
- `file_exists`: Check if a file exists
- `exec`: Execute a shell command
- `github_repo`: Fetch GitHub repository
- `github_file`: Fetch GitHub file
- `github_tree`: List repository contents
- `github_issue`: Fetch GitHub issue
- `github_milestone`: Fetch GitHub milestone
- `jira_task_get`: Fetch JIRA task by key
- `gitlab_fetch_mr`: Fetch GitLab merge request
- `cve`: Lookup vulnerability from OSV database

### Sequential Tools (`Sequential: true`)
File-mutating tools that run **after all parallel tools finish**, one at a time, in the order the LLM listed them. This guarantees each tool sees the result of the previous write.
- `file_write`: Write a file (create or overwrite)
- `file_edit`: Replace a unique text fragment in a plain-text file
- `file_edit_multiple`: Apply multiple text replacements in a plain-text file

## Performance Notes

- **Parallel tools** are executed concurrently using goroutines and `sync.WaitGroup`.
- **Sequential tools** run one at a time after all parallel tools in the same batch have finished, preserving correct filesystem state.
- Message history is trimmed to fit context window on each iteration.
- Message compaction is triggered automatically when needed.

## Error Handling

1. **LLM request failure**: Returns error immediately
2. **Tool execution failure**: Logs error, continues with other tools
3. **Unknown tool**: Returns "error: unknown tool" to LLM
4. **Search limit reached**: Returns error without executing tool
5. **Duplicate tool call**: Returns error without executing tool

## Security

- Search tool calls are limited to prevent resource exhaustion
- Lookup tools can still loop if arguments change (mitigated by max iterations)
- Tool execution is sandboxed through error handling
