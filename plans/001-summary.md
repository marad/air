# Plan 001: Request Summary and Output File Support

## Overview

This plan describes the implementation of two new features for AIR:
1. Display request summary showing token usage after each API call
2. Add CLI flags to save output to file and optionally hide the summary

## Objectives

- Show users token consumption (input/output/total tokens) after each request
- Allow users to save AI response to a file using `-o` or `--output` flag
- Allow users to hide the summary display using `--no-summary` flag
- Maintain backward compatibility (summary shown by default)

**Note**: Cost estimation was originally planned but removed to avoid providing misleading information with hard-coded pricing data.

## Current Architecture Analysis

### Entry Point (main.go)
- Parses CLI arguments using custom `template.ParseVarFlags()` 
- Loads template file and processes it through multiple stages
- Calls `ai.CallVertexAI()` which returns only the text response
- Prints output to stdout using `fmt.Println()`
- No metadata is currently captured or displayed

### AI Module (internal/ai/ai.go)
- `CallVertexAI()` returns only `string` (response text) and `error`
- Receives full `GenerateContentResponse` from Vertex AI which contains:
  - `Candidates` - response content
  - `UsageMetadata` - token counts
- `UsageMetadata` structure includes:
  - `PromptTokenCount` (int32) - input tokens
  - `CandidatesTokenCount` (int32) - output tokens  
  - `TotalTokenCount` (int32) - total tokens
- Currently discards metadata after extracting text

### Config Module (internal/config/config.go)
- Handles YAML frontmatter parsing
- Stores configuration like temperature, model, safety settings
- No CLI flag parsing - done separately in main.go

### Template Module (internal/template/template.go)
- Contains `ParseVarFlags()` for parsing `--var` flags
- Returns: (variables map, remaining args, error)
- Uses simple loop-based parsing approach

## Implementation Plan

### Phase 1: Extend AI Module to Return Metadata

**Goal**: Modify ai package to return usage metadata alongside response text

**Changes to internal/ai/ai.go**:

1. Create new result struct:
```
type Response struct {
    Text          string
    InputTokens   int32
    OutputTokens  int32
    TotalTokens   int32
}
```

2. Modify `CallVertexAI()` signature:
```
OLD: func CallVertexAI(ctx, cfg, prompt) (string, error)
NEW: func CallVertexAI(ctx, cfg, prompt) (*Response, error)
```

3. Update `extractText()` to extract both text and metadata:
```
OLD: func extractText(resp) (string, error)
NEW: func extractResponse(resp) (*Response, error)
```

4. Implementation logic for `extractResponse()`:
```
PSEUDOCODE:
- Check resp.Candidates exists and has content
- Extract text from first candidate (existing logic)
- Initialize result struct with text
- IF resp.UsageMetadata is not nil:
    - Set InputTokens = UsageMetadata.PromptTokenCount
    - Set OutputTokens = UsageMetadata.CandidatesTokenCount
    - Set TotalTokens = UsageMetadata.TotalTokenCount
- ELSE:
    - Set all token counts to 0 (graceful degradation)
- Return result struct
```

**Tests to update**: internal/ai/ai_test.go
- Update all test expectations to expect `*Response` instead of `string`
- Add specific tests for metadata extraction
- Test case when UsageMetadata is nil

### Phase 2: Extend CLI Flag Parsing

**Goal**: Add support for `-o`, `--output`, and `--no-summary` flags

**Changes to internal/template/template.go**:

1. Create new struct for CLI options:
```
type CLIOptions struct {
    Variables   map[string]string  // existing --var flags
    OutputFile  string             // -o, --output
    NoSummary   bool               // --no-summary
}
```

2. Modify `ParseVarFlags()` signature:
```
OLD: func ParseVarFlags(args []string) (map[string]string, []string, error)
NEW: func ParseCLIFlags(args []string) (*CLIOptions, []string, error)
```

3. Keep old function as alias for backward compatibility:
```
func ParseVarFlags(args []string) (map[string]string, []string, error) {
    opts, remaining, err := ParseCLIFlags(args)
    if err != nil:
        return nil, nil, err
    return opts.Variables, remaining, nil
}
```

4. Implementation logic for `ParseCLIFlags()`:
```
PSEUDOCODE:
- Initialize opts = CLIOptions{Variables: make(map[string]string)}
- Initialize remaining = empty slice
- Loop through args with index i:
    - IF arg is "--var" or "-v":
        - Check next arg exists
        - Parse next arg as "key=value"
        - Store in opts.Variables
        - Increment i (skip next arg)
    - ELSE IF arg is "-o" or "--output":
        - Check next arg exists
        - Check opts.OutputFile is not already set (error if duplicate)
        - Set opts.OutputFile = next arg
        - Increment i
    - ELSE IF arg is "--no-summary":
        - Set opts.NoSummary = true
    - ELSE:
        - Append arg to remaining
    - Increment i
- Return opts, remaining, nil
```

**Tests to update**: internal/template/template_test.go
- Test parsing `-o filename`
- Test parsing `--output filename`
- Test parsing `--no-summary`
- Test combined flags: `--var x=1 -o out.txt --no-summary`
- Test error: `-o` without filename
- Test error: duplicate `-o` flags
- Test backward compatibility via `ParseVarFlags()`

### Phase 3: Add Cost Calculation Module

**Goal**: Calculate estimated cost based on token usage and model

**New file: internal/cost/cost.go**:

1. Define pricing constants (as of November 2024):
```
PRICING MAP:
  gemini-2.0-flash-001:
    input:  $0.075 per 1M tokens  ($0.000000075 per token)
    output: $0.30 per 1M tokens   ($0.000000300 per token)
  
  gemini-1.5-pro-002:
    input:  $1.25 per 1M tokens   ($0.00000125 per token)
    output: $5.00 per 1M tokens   ($0.00000500 per token)
  
  gemini-1.5-flash-002:
    input:  $0.075 per 1M tokens  ($0.000000075 per token)
    output: $0.30 per 1M tokens   ($0.000000300 per token)
  
  [similar for other models]
```

2. Create pricing struct:
```
type ModelPricing struct {
    InputCostPerToken  float64
    OutputCostPerToken float64
}
```

3. Create function to calculate cost:
```
func CalculateCost(model string, inputTokens, outputTokens int32) float64
```

4. Implementation logic:
```
PSEUDOCODE:
- Get pricing for model from map
- IF model not found:
    - Return 0.0 (unknown cost)
- Calculate: inputCost = inputTokens * pricing.InputCostPerToken
- Calculate: outputCost = outputTokens * pricing.OutputCostPerToken
- Return: inputCost + outputCost
```

5. Create formatting function:
```
func FormatCost(cost float64) string
```

6. Implementation logic:
```
PSEUDOCODE:
- IF cost < 0.000001:
    - Return "< $0.000001"
- IF cost < 0.01:
    - Format with 6 decimal places
- ELSE:
    - Format with 4 decimal places
- Return formatted string with $ prefix
```

**Tests to create**: internal/cost/cost_test.go
- Test cost calculation for each supported model
- Test with zero tokens
- Test with unknown model
- Test cost formatting for various amounts
- Test edge cases (very small, very large costs)

### Phase 4: Add Summary Display Module

**Goal**: Format and display request summary

**New file: internal/summary/summary.go**:

1. Create summary struct:
```
type RequestSummary struct {
    Model         string
    InputTokens   int32
    OutputTokens  int32
    TotalTokens   int32
    EstimatedCost float64
}
```

2. Create function to build summary:
```
func BuildSummary(model string, response *ai.Response) *RequestSummary
```

3. Implementation logic:
```
PSEUDOCODE:
- Calculate cost using cost.CalculateCost(model, response.InputTokens, response.OutputTokens)
- Return RequestSummary with all fields populated
```

4. Create formatting function:
```
func (s *RequestSummary) Format() string
```

5. Implementation logic:
```
PSEUDOCODE:
- Build multi-line string with format:
  
  ---
  Request Summary
  Model: {model}
  Input tokens: {inputTokens}
  Output tokens: {outputTokens}
  Total tokens: {totalTokens}
  Estimated cost: {formattedCost}
  ---
  
- Use cost.FormatCost() for cost formatting
- Return formatted string
```

6. Create display function:
```
func Display(summary *RequestSummary, writer io.Writer)
```

7. Implementation logic:
```
PSEUDOCODE:
- Write formatted summary to writer (typically os.Stderr)
- Ensures summary doesn't interfere with stdout output
```

**Tests to create**: internal/summary/summary_test.go
- Test BuildSummary with various models and token counts
- Test Format() output structure
- Test Display() writes to correct writer
- Test formatting with zero tokens
- Test formatting with unknown model (cost = 0)

### Phase 5: Integrate in Main

**Goal**: Wire up all components in main.go

**Changes to main.go**:

1. Update imports:
```
Add:
- "air/internal/cost"
- "air/internal/summary"
- "io"
```

2. Update CLI parsing:
```
OLD: cliVars, args, err := template.ParseVarFlags(os.Args[1:])
NEW: cliOpts, args, err := template.ParseCLIFlags(os.Args[1:])
```

3. Update variable merging:
```
OLD: variables := template.MergeVariables(envVars, cfg.Variables, cliVars)
NEW: variables := template.MergeVariables(envVars, cfg.Variables, cliOpts.Variables)
```

4. Update AI call:
```
OLD: result, err := ai.CallVertexAI(ctx, cfg, finalMarkdown)
     // result is string
NEW: response, err := ai.CallVertexAI(ctx, cfg, finalMarkdown)
     // response is *ai.Response
```

5. Update output handling:
```
OLD: output := result
     if cfg.ResponseSchema != nil:
         output = schema.FormatResponse(result)
     fmt.Println(output)

NEW: output := response.Text
     if cfg.ResponseSchema != nil:
         output = schema.FormatResponse(response.Text)
     
     // Write output to file or stdout
     if cliOpts.OutputFile != "":
         err := writeOutputToFile(cliOpts.OutputFile, output)
         if err != nil:
             fatalf(ExitFileError, "Error writing output: %v", err)
     else:
         fmt.Println(output)
     
     // Show summary if not disabled
     if !cliOpts.NoSummary:
         model := cfg.ModelOrDefault()
         s := summary.BuildSummary(model, response)
         summary.Display(s, os.Stderr)
```

6. Add helper function:
```
func writeOutputToFile(filename, content string) error
```

7. Implementation logic:
```
PSEUDOCODE:
- Open or create file with write permissions (0644)
- IF error: return wrapped error
- Defer file.Close()
- Write content to file
- IF error: return wrapped error
- Return nil
```

**Tests to update**: integration_test.go
- Test default behavior (shows summary)
- Test --no-summary flag (hides summary)
- Test -o flag (saves to file)
- Test combined: -o with --no-summary
- Test output file creation
- Test output file error handling
- Verify summary goes to stderr, not stdout
- Verify backward compatibility

### Phase 6: Update Documentation

**Changes to README.md**:

1. Add new section "Output Options" before "Troubleshooting":

```markdown
## Output Options

### Saving Output to File

Save the AI response to a file instead of displaying it:

```bash
./air template.md -o output.txt
./air template.md --output response.json
```

The file will be created or overwritten if it exists.

### Request Summary

After each request, AIR displays a summary with token usage and estimated cost:

```
---
Request Summary
Model: gemini-2.0-flash-001
Input tokens: 1234
Output tokens: 567
Total tokens: 1801
Estimated cost: $0.000274
---
```

To hide the summary:

```bash
./air template.md --no-summary
```

The summary is printed to stderr, so it won't interfere with piping output.

### Combining Options

You can combine multiple options:

```bash
./air template.md --var name=Alice -o result.txt --no-summary
```
```

2. Update "Usage" section to mention new flags:

```markdown
## Usage

Basic usage:

```bash
./air prompt_template.md
```

With options:

```bash
# Save output to file
./air prompt.md -o output.txt

# Hide request summary
./air prompt.md --no-summary

# Pass variables
./air prompt.md --var name=Alice --var task=coding

# Combine options
./air prompt.md --var x=1 -o out.txt --no-summary
```
```

3. Add to troubleshooting section:

```markdown
**"Error writing output"**
- Check file path is valid
- Ensure you have write permissions for the directory
- Verify disk space is available
```

**Changes to docs/config-reference.md** (if exists):

1. Add CLI flags reference section documenting:
   - `-o, --output <file>` - Save output to file
   - `--no-summary` - Hide request summary
   - `--var, -v <key=value>` - Set template variable

**New file: examples/with_output_file.md**:

Create example demonstrating output file usage.

## Testing Strategy

### Unit Tests

1. **internal/ai/ai_test.go**:
   - Test Response struct creation
   - Test metadata extraction with valid UsageMetadata
   - Test metadata extraction when UsageMetadata is nil
   - Test backward compatibility of response handling

2. **internal/template/template_test.go**:
   - Test parsing each new flag individually
   - Test parsing combined flags
   - Test error cases (missing arguments, duplicates)
   - Test backward compatibility of ParseVarFlags()

3. **internal/cost/cost_test.go**:
   - Test cost calculation for each model
   - Test with zero tokens
   - Test with unknown model
   - Test cost formatting edge cases

4. **internal/summary/summary_test.go**:
   - Test summary building
   - Test summary formatting
   - Test summary display to custom writer

### Integration Tests

1. **integration_test.go**:
   - Test full workflow with -o flag
   - Test full workflow with --no-summary flag
   - Test full workflow with both flags
   - Test summary appears on stderr
   - Test output file creation and content
   - Verify token counts are non-zero for real requests
   - Verify cost calculation is reasonable

### Manual Testing

1. Test with different models to verify pricing
2. Test with various template sizes (different token counts)
3. Test error handling for file write failures
4. Test summary formatting with very large/small costs
5. Verify backward compatibility with existing templates

## Rollout Plan

### Step 1: Implement and Test Core Features
- Implement phases 1-3 (metadata, flags, cost)
- Write and pass all unit tests
- Code review

### Step 2: Implement Display and Integration
- Implement phases 4-5 (summary display, main integration)
- Write and pass integration tests
- Manual testing

### Step 3: Documentation
- Implement phase 6 (documentation updates)
- Review all docs for accuracy
- Create examples

### Step 4: Final Validation
- Run full test suite
- Manual testing with various scenarios
- Performance check (ensure no regression)
- Review all error messages for clarity

## Backward Compatibility

This implementation maintains full backward compatibility:

- Default behavior unchanged (output to stdout)
- Summary is new feature, shown by default but unobtrusive
- No existing functionality removed or changed
- All existing templates and commands continue to work
- ParseVarFlags() preserved as alias

Users who want old behavior (no summary) can use `--no-summary` flag.

## Edge Cases and Error Handling

1. **Missing UsageMetadata**: Show summary with "N/A" or zero values
2. **Unknown model for pricing**: Show cost as "unknown" or "$0.00 (pricing not available)"
3. **File write errors**: Clear error message with exit code 3 (ExitFileError)
4. **Duplicate output flags**: Error with message "multiple output files specified"
5. **Missing output filename**: Error with message "-o/--output requires a filename"
6. **Summary formatting for zero cost**: Show "$0.000000" or "< $0.000001"
7. **Very large costs**: Format with appropriate precision (e.g., "$123.4567")

## Cost Calculation Notes

- Costs are estimated based on published pricing (as of implementation date)
- Actual costs may vary slightly due to:
  - Pricing changes by Google Cloud
  - Regional pricing differences
  - Commitment discounts
  - Promotional credits
- Recommend adding comment in code and docs that costs are estimates
- Consider adding flag or env var for custom pricing in future

## Future Enhancements (Out of Scope)

These are NOT part of this implementation but noted for future consideration:

1. JSON output format for summary (`--summary-format json`)
2. Append mode for output file (`-a` or `--append`)
3. Save summary to separate file (`--summary-output`)
4. Custom pricing configuration file
5. Show running totals across multiple requests (session tracking)
6. Historical cost tracking and reporting
7. Budget alerts/warnings
8. Token usage visualization

## Implementation Checklist

- [ ] Phase 1: Extend AI module to return metadata
  - [ ] Create Response struct
  - [ ] Modify CallVertexAI signature
  - [ ] Update extractText to extractResponse
  - [ ] Update tests
- [ ] Phase 2: Extend CLI flag parsing
  - [ ] Create CLIOptions struct
  - [ ] Implement ParseCLIFlags
  - [ ] Add backward compatibility alias
  - [ ] Update tests
- [ ] Phase 3: Add cost calculation module
  - [ ] Create cost.go with pricing data
  - [ ] Implement CalculateCost
  - [ ] Implement FormatCost
  - [ ] Write tests
- [ ] Phase 4: Add summary display module
  - [ ] Create summary.go
  - [ ] Implement BuildSummary
  - [ ] Implement Format
  - [ ] Implement Display
  - [ ] Write tests
- [ ] Phase 5: Integrate in main
  - [ ] Update imports
  - [ ] Update CLI parsing
  - [ ] Update AI call handling
  - [ ] Add file writing logic
  - [ ] Add summary display logic
  - [ ] Update integration tests
- [ ] Phase 6: Update documentation
  - [ ] Update README.md
  - [ ] Update config reference (if exists)
  - [ ] Create examples
- [ ] Final validation
  - [ ] All tests pass
  - [ ] Manual testing complete
  - [ ] Documentation reviewed
  - [ ] Code reviewed

## Success Criteria

Implementation is complete when:

1. All unit tests pass
2. All integration tests pass
3. Manual testing confirms:
   - Summary displays correctly with token counts and cost
   - `-o` flag saves output to specified file
   - `--no-summary` flag hides summary
   - Flags can be combined correctly
   - Error handling works as expected
4. Documentation is complete and accurate
5. Code review approved
6. Backward compatibility verified (existing templates work unchanged)

## Estimated Effort

- Phase 1: 2-3 hours (implementation + tests)
- Phase 2: 2-3 hours (implementation + tests)
- Phase 3: 1-2 hours (implementation + tests)
- Phase 4: 1-2 hours (implementation + tests)
- Phase 5: 2-3 hours (integration + tests)
- Phase 6: 1-2 hours (documentation)
- Testing and validation: 2-3 hours

**Total**: 11-18 hours
