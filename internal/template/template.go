package template

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var IncludePattern = regexp.MustCompile(`\{\{include\s+"([^"]+)"\}\}`)

var PlaceholderPattern = regexp.MustCompile(`\{\{([a-zA-Z_][a-zA-Z0-9_]*?)(?:\|([^}]*))?\}\}`)

// InclusionContext tracks processed files to detect circular includes
type InclusionContext struct {
	Visited map[string]bool // Absolute paths of files currently being processed
	BaseDir string          // Base directory for resolving relative includes
}

func NewInclusionContext(initialFile string) *InclusionContext {
	return &InclusionContext{
		Visited: make(map[string]bool),
		BaseDir: filepath.Dir(initialFile),
	}
}

func ResolveAbsolutePath(path, baseDir string) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	cleaned := filepath.Clean(path)
	return filepath.Abs(cleaned)
}

// validatePathSecurity ensures the include path doesn't escape the project directory
func validatePathSecurity(absPath string) error {
	projectRoot, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("getting project root: %w", err)
	}
	rel, err := filepath.Rel(projectRoot, absPath)
	if err != nil {
		return fmt.Errorf("resolving path relation: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("include path is outside the project directory")
	}
	return nil
}

// checkCircular verifies no circular dependency exists
func (ctx *InclusionContext) checkCircular(absPath string) error {
	if ctx.Visited[absPath] {
		return fmt.Errorf("circular include detected: %s", absPath)
	}
	return nil
}

// processIncludeFile reads and recursively processes an included file
func (ctx *InclusionContext) processIncludeFile(absPath string) (string, error) {
	ctx.Visited[absPath] = true
	defer delete(ctx.Visited, absPath) // Allow same file in different branches

	includedContent, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("reading included file: %w", err)
	}

	// Process nested includes with updated baseDir
	oldBaseDir := ctx.BaseDir
	ctx.BaseDir = filepath.Dir(absPath)
	defer func() { ctx.BaseDir = oldBaseDir }()

	return ProcessIncludes(string(includedContent), ctx)
}

func ProcessIncludes(content string, ctx *InclusionContext) (string, error) {
	var result strings.Builder
	lastIndex := 0

	for {
		sub := content[lastIndex:]
		idxs := IncludePattern.FindStringSubmatchIndex(sub)
		if idxs == nil {
			result.WriteString(sub)
			break
		}

		// idxs[0], idxs[1] are start/end of full match; idxs[2],idxs[3] are the first capture group
		matchStart := lastIndex + idxs[0]
		matchEnd := lastIndex + idxs[1]
		includePath := sub[idxs[2]:idxs[3]]

		// Write content before match
		result.WriteString(content[lastIndex:matchStart])

		// Resolve path relative to current file's directory
		absPath, err := ResolveAbsolutePath(includePath, ctx.BaseDir)
		if err != nil {
			return "", fmt.Errorf("resolving include path %s: %w", includePath, err)
		}

		// Security check
		if err := validatePathSecurity(absPath); err != nil {
			return "", fmt.Errorf("%s: %w", includePath, err)
		}

		// Check for circular includes
		if err := ctx.checkCircular(absPath); err != nil {
			return "", fmt.Errorf("%s: %w", includePath, err)
		}

		// Process included file
		processedContent, err := ctx.processIncludeFile(absPath)
		if err != nil {
			return "", err
		}

		result.WriteString(processedContent)
		lastIndex = matchEnd
	}

	return result.String(), nil
}

func ReplacePlaceholders(content string, variables map[string]string) (string, error) {
	missingMap := make(map[string]struct{})

	result := PlaceholderPattern.ReplaceAllStringFunc(content, func(match string) string {
		submatches := PlaceholderPattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		varName := submatches[1]
		if value, ok := variables[varName]; ok {
			return value
		}

		if len(submatches) >= 3 && submatches[2] != "" {
			return submatches[2] // Default value
		}

		// No value and no default - track as missing
		missingMap[varName] = struct{}{}
		return match
	})

	if len(missingMap) > 0 {
		missingList := make([]string, 0, len(missingMap))
		for k := range missingMap {
			missingList = append(missingList, k)
		}
		sort.Strings(missingList)
		return "", fmt.Errorf("undefined variables without defaults: %v", missingList)
	}

	return result, nil
}

func ParseVarFlags(args []string) (map[string]string, []string, error) {
	vars := make(map[string]string)
	remaining := []string{}

	i := 0
	for i < len(args) {
		arg := args[i]

		if arg == "--var" || arg == "-v" {
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("--var requires an argument")
			}

			i++
			varDef := args[i]

			// Parse "key=value"
			parts := strings.SplitN(varDef, "=", 2)
			if len(parts) != 2 {
				return nil, nil, fmt.Errorf("invalid --var format: %s (expected key=value)", varDef)
			}

			vars[parts[0]] = parts[1]
		} else {
			remaining = append(remaining, arg)
		}

		i++
	}

	return vars, remaining, nil
}

func GetEnvVariables() map[string]string {
	vars := make(map[string]string)

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			vars[parts[0]] = parts[1]
		}
	}

	return vars
}

func MergeVariables(sources ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, src := range sources {
		for k, v := range src {
			result[k] = v
		}
	}
	return result
}
