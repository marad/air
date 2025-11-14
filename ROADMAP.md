# AIR - Implementation Roadmap

This document outlines the step-by-step implementation plan for AIR (AI Requester), organized into milestones that each deliver working functionality.

---

## Milestone 1: Podstawowa funkcjonalność - "Hello World z pliku"

**Goal**: Get basic end-to-end flow working - read a markdown file and send it to Vertex AI.

### Tasks:
1. **CLI - czytanie pliku** 
   - Accept filename as command-line argument
   - Read file contents
   - Handle file not found errors

2. **Parsowanie YAML frontmatter**
   - Extract YAML frontmatter from markdown (between `---` delimiters)
   - Parse YAML into config struct
   - Separate frontmatter from markdown content

3. **Ładowanie .env**
   - Load `.env` file from current directory on startup
   - Set environment variables for `GOOGLE_CLOUD_PROJECT` and `GOOGLE_CLOUD_LOCATION`
   - Graceful handling if `.env` doesn't exist

4. **Integracja z AI**
   - Use markdown content (without frontmatter) as prompt
   - Send to Vertex AI using existing `callVertexAI` logic
   - Handle API errors gracefully

5. **Wyświetlanie odpowiedzi**
   - Print AI response to stdout
   - Clean formatting

**Success Criteria**: `./air template.md` reads a markdown file, sends it to Vertex AI, and displays the response.

---

## Milestone 2: Konfigurowalność - "Parametry generowania"

**Goal**: Allow users to control AI generation parameters via YAML frontmatter.

### Tasks:
6. **Konfiguracja parametrów AI**
   - Read `temperature`, `topP`, `maxTokens` from YAML frontmatter
   - Use parsed values in AI request
   - Fall back to sensible defaults if not specified

7. **Konfiguracja safety settings**
   - Support safety settings configuration in YAML
   - Map YAML values to protobuf safety enums
   - Keep current BLOCK_NONE defaults if not configured

8. **Wybór modelu**
   - Allow `model` field in YAML frontmatter
   - Support different Gemini model versions
   - Default to current gemini-2.0-flash-001

**Success Criteria**: Users can control temperature, safety, and model selection via YAML frontmatter.

---

## Milestone 3: Templating - "Składanie promptów z kawałków"

**Goal**: Enable building complex prompts from reusable template pieces.

### Tasks:
9. **File inclusion**
   - Design and implement file inclusion syntax
   - Support relative and absolute paths
   - Prevent circular includes (detect and error)
   - Handle missing included files gracefully

10. **Placeholder replacement**
    - Design and implement placeholder syntax
    - Support default values for placeholders
    - Replace all placeholders in template content

11. **Przekazywanie zmiennych**
    - Accept variables via command-line flags
    - Read variables from environment variables
    - Support variables section in YAML frontmatter
    - Define precedence order (CLI > frontmatter > env)

**Success Criteria**: Users can build prompts from multiple files with dynamic placeholders.

---

## Milestone 4: Structured Output - "JSON Schema"

**Goal**: Support structured JSON output with schema validation.

### Tasks:
12. **Parsowanie schema z YAML**
    - Parse `responseSchema` section from YAML frontmatter
    - Support nested objects and arrays
    - Validate schema structure before use

13. **Konwersja do protobuf Schema**
    - Map YAML schema types to `aiplatformpb.Type` enums
    - Build protobuf Schema structure recursively
    - Handle properties, items, and nested definitions

14. **Walidacja odpowiedzi**
    - Verify AI response matches expected schema
    - Provide clear error messages for schema mismatches
    - Pretty-print JSON responses

**Success Criteria**: Users can specify output schema in YAML and receive validated JSON responses.

---

## Milestone 5: Polish - "Produkcyjna jakość"

**Goal**: Production-ready code quality, testing, and documentation.

### Tasks:
15. **Refaktor struktury**
    - Split code into packages: `internal/config/`, `internal/template/`, `internal/ai/`, `internal/schema/`
    - Move logic from main.go into appropriate packages
    - Clean separation of concerns

16. **Error handling**
    - Meaningful error messages with context
    - Proper exit codes for different error types
    - User-friendly suggestions in error messages

17. **Testowanie**
    - Unit tests for config parsing
    - Unit tests for template processing (includes, placeholders)
    - Unit tests for schema conversion
    - Integration tests with mock AI responses

18. **Dokumentacja i przykłady**
    - Create `examples/` directory with sample templates
    - Document all YAML frontmatter options
    - Update README with usage examples
    - Add troubleshooting guide

**Success Criteria**: Well-tested, documented tool ready for production use.

---

## Current Status

- [x] PoC implementation (basic Vertex AI call)
- [x] Milestone 1 - Completed
- [ ] Milestone 2 - Not Started
- [ ] Milestone 3 - Not Started
- [ ] Milestone 4 - Not Started
- [ ] Milestone 5 - Not Started

---

## Notes

- Each milestone delivers working, usable functionality
- Milestones can be developed incrementally
- Project is usable after Milestone 1, with increasing capability at each stage
- Milestones 3-4 are independent and order could be swapped based on priority
