package template

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var IncludePattern = regexp.MustCompile(`\{\{include\s+"([^"]+)"\}\}`)

var PlaceholderPattern = regexp.MustCompile(`\{\{([a-zA-Z_][a-zA-Z0-9_]*?)(?:\|([^}]*))?\}\}`)

// InclusionContext tracks processed files to detect circular includes
type InclusionContext struct {
	Visited map[string]bool
	BaseDir string
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
	return filepath.Abs(path)
}

func ProcessIncludes(content string, ctx *InclusionContext) (string, error) {
	var result strings.Builder
	lastIndex := 0

	for {
		matches := IncludePattern.FindStringSubmatch(content[lastIndex:])
		if matches == nil {
			result.WriteString(content[lastIndex:])
			break
		}

		// Calculate absolute position
		matchStart := lastIndex + strings.Index(content[lastIndex:], matches[0])
		matchEnd := matchStart + len(matches[0])

		// Write content before match
		result.WriteString(content[lastIndex:matchStart])

		includePath := matches[1]

		// Resolve path (relative to current file's directory)
		absPath, err := ResolveAbsolutePath(includePath, ctx.BaseDir)
		if err != nil {
			return "", fmt.Errorf("resolving include path %s: %w", includePath, err)
		}

		// Security check: prevent directory traversal outside the project directory
		projectRoot, err := filepath.Abs(".")
		if err != nil {
			return "", fmt.Errorf("getting project root: %w", err)
		}
		if !strings.HasPrefix(absPath, projectRoot) {
			return "", fmt.Errorf("include path %s is outside the project directory", includePath)
		}

		// Check for circular includes
		if ctx.Visited[absPath] {
			return "", fmt.Errorf("circular include detected: %s", includePath)
		}

		// Mark as visited
		ctx.Visited[absPath] = true

		// Read included file
		includedContent, err := os.ReadFile(absPath)
		if err != nil {
			return "", fmt.Errorf("reading included file %s: %w", includePath, err)
		}

		// Recursively process includes in the included file
		// Update baseDir for nested includes
		oldBaseDir := ctx.BaseDir
		ctx.BaseDir = filepath.Dir(absPath)

		processedContent, err := ProcessIncludes(string(includedContent), ctx)
		if err != nil {
			return "", err
		}

		ctx.BaseDir = oldBaseDir

		// Write processed content
		result.WriteString(processedContent)

		// Unmark for other branches (allows same file in different paths)
		delete(ctx.Visited, absPath)

		lastIndex = matchEnd
	}

	return result.String(), nil
}

func ReplacePlaceholders(content string, variables map[string]string) (string, error) {
	var missing []string

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
		missing = append(missing, varName)
		return match
	})

	if len(missing) > 0 {
		return "", fmt.Errorf("undefined variables without defaults: %v", missing)
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