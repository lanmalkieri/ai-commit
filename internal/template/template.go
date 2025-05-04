package template

import (
	"embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed templates
var templateFS embed.FS

// LoadAndExecuteTemplate loads and executes a template with the given diff data
func LoadAndExecuteTemplate(templateName string, diffData string) (string, error) {
	// Construct the template path
	templatePath := fmt.Sprintf("templates/%s.tmpl", templateName)
	
	// Read the template file
	templateContent, err := templateFS.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to load template '%s': %w", templateName, err)
	}
	
	// Parse the template
	tmpl, err := template.New("commit").Parse(string(templateContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}
	
	// Define data to pass to the template
	data := struct {
		Diff string
	}{
		Diff: diffData,
	}
	
	// Execute the template
	var builder strings.Builder
	if err := tmpl.Execute(&builder, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	
	return builder.String(), nil
}