# AIR - AI Requester

AIR is a tool that helps you in the most basic thing when working with LLMs - prompt creation and
tuning.

It basically sends the request to AI and shows its response.

## Usage 

The usage simple - just invoke air with your prompt template:

```bash
> air prompt_template.md
# ai response
```

## Prompt Templates

Prompts are simple markdown files. Air uses the templating engine that let's you split the prompt
into multiple files - simulating prompt generation from the template.

It allows:
- including other files
- replacing named placeholders with concrete values

## Templating Features

### File Inclusion

Include external files in your templates:

```markdown
{{include "fragments/header.md"}}
Main content here
{{include "fragments/footer.md"}}
```

Includes support:
- Relative paths (resolved from current file's directory)
- Absolute paths
- Nested includes (includes can contain includes)
- Circular dependency detection

### Variables and Placeholders

Use placeholders with default values:

```markdown
Hello {{name|User}}!
Your task: {{task}}
```

Variables can be provided via:

1. **CLI flags** (highest priority):
   ```bash
   ./air template.md --var name=Alice --var task=coding
   ```

2. **YAML frontmatter**:
   ```yaml
   ---
   variables:
     name: Bob
     task: writing
   ---
   ```

3. **Environment variables** (lowest priority):
   ```bash
   export NAME=Charlie
   ./air template.md
   ```

Default values: Use `{{variable|default_value}}` syntax.

## Configuration

While prompt is a simple markdown file, you can add YAML frontmatter in the beginning to modify how
the request is going to behave.

### Generation parameters and safety settings

You can provide the basic generation parameters as simple YAML values:

```yaml
---
temperature: 0.2
topP: 0.95
maxTokens: 8192
model: gemini-1.5-pro-002
responseMimeType: application/json
---
```

**Available options:**
- `temperature` (float32, 0.0-2.0): Controls randomness (0.0 = deterministic, higher = more creative)
- `topP` (float32, 0.0-1.0): Nucleus sampling parameter
- `maxTokens` (int32): Maximum response length
- `model` (string): AI model to use. [Supported models](https://docs.cloud.google.com/vertex-ai/generative-ai/docs/learn/model-versions)
- `responseMimeType` (string): Response format, usually `application/json` or `text/plain`

**Safety Settings:**
Configure content filtering:

```yaml
---
safetySettings:
  hate_speech: BLOCK_LOW_AND_ABOVE
  dangerous_content: BLOCK_MEDIUM_AND_ABOVE
  sexually_explicit: BLOCK_ONLY_HIGH
  harassment: BLOCK_NONE
---
```

**Available categories:** `hate_speech`, `dangerous_content`, `sexually_explicit`, `harassment`

**Thresholds:** `BLOCK_NONE`, `BLOCK_ONLY_HIGH`, `BLOCK_MEDIUM_AND_ABOVE`, `BLOCK_LOW_AND_ABOVE`

### Support for `.env`

On startup `air` also reads the environment variables from the `.env` in current directory. This
allows you to set `GOOGLE_CLOUD_PROJECT` and `GOOGLE_CLOUD_LOCATION` locally to the project you are
working on.

### Output Schema Configuration

You can provide the expected response schema within the YAML frontmatter. When specified, AIR will:

- Request structured JSON output from the AI model
- Validate the response against the schema
- Pretty-print the JSON response for readability

Example:

```yaml
---
responseSchema:
  type: object
  properties:
    name:
      type: string
    age:
      type: integer
  required:
    - name
    - age
---
```

This should produce a response like:

```json
{
  "name": "Alice",
  "age": 30
}
```

If the response doesn't match the schema, a warning will be printed to stderr, but the response is still returned.

## Troubleshooting

### Common Issues

**"GOOGLE_CLOUD_PROJECT environment variable not set"**
- Set your Google Cloud project ID: `export GOOGLE_CLOUD_PROJECT=your-project-id`
- Or add it to `.env` file: `GOOGLE_CLOUD_PROJECT=your-project-id`

**"unsupported model"**
- Check supported models in the configuration section above
- Use a valid model name like `gemini-2.0-flash-001`

**"invalid safety threshold" or "unknown harm category"**
- Verify safety settings use correct categories and thresholds (see configuration section)

**"undefined variables without defaults"**
- Provide all required variables via CLI (`--var key=value`), YAML frontmatter, or environment variables
- Or add default values in placeholders: `{{variable|default}}`

**"circular include detected"**
- Check your `{{include}}` directives for loops
- Ensure included files don't include each other

**"include path is outside the project directory"**
- Include paths must be within the project root
- Use relative paths from the template file's directory

**Exit Codes:**
- 0: Success
- 2: Invalid command-line arguments
- 3: File reading errors
- 4: Configuration parsing/validation errors
- 5: Template processing errors
- 6: AI API errors

### Getting Help

For more examples, see the `examples/` directory. Each file demonstrates different features.
