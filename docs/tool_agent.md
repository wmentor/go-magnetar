# Agent Tool

The `agent` tool allows you to invoke a subagent by name and process a message with it. This is useful when you need to delegate a task to a specialized agent that has its own prompt and available tools.

## Signature

```
agent(agent: string, message: string) -> string
```

## Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `agent` | string | The name of the agent to invoke. Must be a valid agent registered in the system. |
| `message` | string | The message to process by the agent. This should contain the task or question to delegate. |

## Usage

The `agent` tool is called automatically by the chat agent when it determines that a subagent is needed to handle a request. You don't need to manually invoke this tool; the agent will use it when appropriate.

### Example

If you ask a question that requires domain-specific knowledge or a specialized workflow, the agent might delegate the task to a subagent:

```text
User: Can you analyze this GitHub repository for security issues?
Agent: [Uses agent tool to delegate to security-analyzer agent]
```

## How It Works

1. The agent determines that a subagent is needed to handle the request
2. The `agent` tool is invoked with the target agent name and the message
3. The subagent processes the message using its own prompt and available tools
4. The subagent returns a response
5. The main agent receives the response and incorporates it into the final answer

## Subagents

Subagents are defined in the configuration and can have:
- Custom prompts
- Specific tool sets
- Unique configurations

To create a subagent, define it in your configuration file under the `agents` section.

## Error Handling

The `agent` tool handles various error scenarios:

- **Invalid agent name**: Returns an error if the specified agent doesn't exist
- **Empty agent name**: Returns an error if the agent name is empty
- **Empty message**: Returns an error if the message is empty
- **Agent execution error**: Returns an error if the subagent fails to execute

## Best Practices

1. **Use specific agent names**: Ensure the agent name matches exactly with the registered agents
2. **Provide clear messages**: The message should clearly describe the task to delegate
3. **Check agent availability**: Verify that the target agent is available and properly configured
4. **Handle responses appropriately**: The agent tool returns a string response that should be processed by the main agent

## Configuration

The `agent` tool is enabled by default and doesn't require specific configuration. However, you can control the available subagents through the configuration file.

See [configuration.md](./configuration.md) for complete configuration options.
