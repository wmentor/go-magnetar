# FAQ

## Why do we only support OpenAI?

We only support OpenAI because of their dominant position in the market, model quality, and open API specification. OpenAI provides:

1. **High-quality models** — GPT-4 and other models are set as the industry standard
2. **Stable API** — OpenAI API has well-documented and stable interfaces
3. **Wide compatibility** — Many other providers (Groq, Together AI, Groq, etc.) offer OpenAI-compatible APIs
4. **Ecosystem** — Large user base and tooling for OpenAI API

In your configuration, you can specify `base_url` for any OpenAI-compatible API:

```yaml
llm:
  api_key: $env:OPENAI_API_KEY
  base_url: https://api.openai.com/v1  # or any other compatible endpoint
```

This allows you to use any compatible provider without code changes — just specify a different `base_url` in the configuration.

## How to run downloaded binary on macOS?

If macOS blocks the downloaded release binary with the error "cannot be opened because the developer cannot be verified", you need to remove the quarantine attribute.

### Using Terminal (command line method)

Remove the quarantine extended attribute using `xattr`:

```bash
xattr -d com.apple.quarantine go-magnetar
```

After running this command, the binary should execute normally:

```bash
./go-magnetar
```
