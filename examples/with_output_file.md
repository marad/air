---
model: gemini-2.0-flash-001
temperature: 0.7
maxTokens: 500
responseMimeType: application/json
responseSchema:
  type: object
  properties:
    title:
      type: string
    summary:
      type: string
    keywords:
      type: array
      items:
        type: string
  required:
    - title
    - summary
    - keywords
---

# Document Analysis Task

Please analyze the following topic and provide a structured response:

**Topic**: {{topic|Artificial Intelligence in Healthcare}}

Create a brief analysis with:
1. A concise title
2. A 2-3 sentence summary
3. 3-5 relevant keywords

# Usage Examples

This example demonstrates saving output to a file with structured JSON response.

## Basic usage - output to stdout with summary:
```bash
./air examples/with_output_file.md
```

## Save output to file:
```bash
./air examples/with_output_file.md -o analysis.json
```

## Save output with custom topic:
```bash
./air examples/with_output_file.md --var topic="Climate Change" -o climate_analysis.json
```

## Save output without summary:
```bash
./air examples/with_output_file.md -o result.json --no-summary
```

## Pipe output (summary goes to stderr, doesn't interfere):
```bash
./air examples/with_output_file.md | jq '.keywords'
```

This will show the summary on your terminal (stderr) but only pipe the JSON response to jq.
