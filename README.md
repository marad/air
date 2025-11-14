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

### Gneration parameters and safety settings

You can provide the basic generation parameters as simple YAML values:

```yaml
---
temperature: 0.2
topP: 0.95
maxTokens: 8192
---
```

TODO: safety settings configuration

### Support for `.env`

On startup `air` also reads the environment variables from the `.env` in current directory. This
allows you to set `GOOGLE_CLOUD_PROJECT` and `GOOGLE_CLOUD_LOCATION` locally to the project you are
working on.

### Output Schema Configuration

You can also provide the expected response schema  within the YAML. For example this schema:

```yaml
responseSchema:
  type: array
  items: 
    type: object
    properties: 
      field_one: string 
      field_two: integer
```

Should make the response conform to a JSON in the form:

```json 
[
  { "field_one": "foo", "field_two": 42 }
  // ...
]
```


