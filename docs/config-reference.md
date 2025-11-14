# Configuration Reference

This document describes all configuration options available in AIR prompt templates via YAML frontmatter.

## Variables

### variables (map, optional)
Define variables for placeholder replacement.

Example:
```yaml
---
variables:
  author: Alice
  version: "1.0"
  status: draft
---

Document by {{author}}, version {{version}}
Status: {{status|unknown}}
```

### Variable Sources

Variables are resolved in this order (highest to lowest priority):

1. **CLI flags**: `--var name=value`
2. **Frontmatter**: `variables:` section in YAML
3. **Environment variables**: System environment

### Placeholder Syntax

- Basic: `{{variable_name}}`
- With default: `{{variable_name|default_value}}`

Variable names:
- Must start with letter or underscore
- Can contain letters, numbers, underscores
- Case-sensitive

### File Inclusion

Include external files:

```markdown
{{include "path/to/file.md"}}
```

Features:
- Relative paths resolved from current file's directory
- Absolute paths supported
- Nested includes allowed
- Circular includes detected and rejected
- Included files can contain includes and placeholders

## Generation Parameters

### temperature (float, optional)
Controls randomness in the response. Values between 0.0 and 2.0.

- 0.0: Most deterministic
- 1.0: Balanced creativity
- 2.0: Most creative

Default: 0.0

### topP (float, optional)
Controls diversity via nucleus sampling. Values between 0.0 and 1.0.

Default: 0.95

### maxTokens (int, optional)
Maximum number of tokens to generate.

Default: 8192

## Model Selection

### model (string, optional)
Specify which Gemini model to use.

Supported models:
- `gemini-2.0-flash-001` (default)
- `gemini-1.5-pro-002`
- `gemini-1.5-pro-001`
- `gemini-1.5-flash-002`
- `gemini-1.5-flash-001`

## Safety Settings

### safetySettings (map, optional)
Configure content safety filters.

Example:
```yaml
safetySettings:
  hate_speech: BLOCK_LOW_AND_ABOVE
  dangerous_content: BLOCK_MEDIUM_AND_ABOVE
  sexually_explicit: BLOCK_NONE
  harassment: BLOCK_ONLY_HIGH
```

Available categories:
- `hate_speech`
- `dangerous_content`
- `sexually_explicit`
- `harassment`

Threshold options:
- `BLOCK_NONE`
- `BLOCK_ONLY_HIGH`
- `BLOCK_MEDIUM_AND_ABOVE`
- `BLOCK_LOW_AND_ABOVE`

Default: All categories set to `BLOCK_NONE`

## Response Configuration

### responseMimeType (string, optional)
Specify the response format.

- `application/json` for JSON responses
- `text/plain` for plain text (default)

### responseSchema (object, optional)
Define expected JSON response structure for schema validation.

Example:
```yaml
responseSchema:
  type: object
  properties:
    title:
      type: string
    content:
      type: string
    tags:
      type: array
      items:
        type: string
```