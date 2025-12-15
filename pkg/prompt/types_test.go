package prompt

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_DefaultTemplates(t *testing.T) {
	t.Run("loads default Claude templates", func(t *testing.T) {
		cfg := Config{
			Provider: "claude",
		}

		templates, err := Load(cfg)
		require.NoError(t, err)
		require.NotNil(t, templates)
		assert.NotNil(t, templates.SingleFix)
		assert.NotNil(t, templates.BatchFix)
		assert.Contains(t, templates.SingleFix.Content, "code migration assistant")
		assert.Contains(t, templates.BatchFix.Content, "code modernization assistant")
	})

	t.Run("loads default OpenAI templates", func(t *testing.T) {
		cfg := Config{
			Provider: "openai",
		}

		templates, err := Load(cfg)
		require.NoError(t, err)
		require.NotNil(t, templates)
		assert.NotNil(t, templates.SingleFix)
		assert.NotNil(t, templates.BatchFix)
	})
}

func TestLoad_CustomTemplates(t *testing.T) {
	// Create temp directory for test templates
	tmpDir := t.TempDir()

	customSingleFix := "Custom single-fix template: {{.File}}"
	customBatchFix := "Custom batch-fix template: {{.ViolationID}}"

	singleFixPath := filepath.Join(tmpDir, "single-fix.txt")
	batchFixPath := filepath.Join(tmpDir, "batch-fix.txt")

	err := os.WriteFile(singleFixPath, []byte(customSingleFix), 0644)
	require.NoError(t, err)
	err = os.WriteFile(batchFixPath, []byte(customBatchFix), 0644)
	require.NoError(t, err)

	t.Run("loads custom templates from files", func(t *testing.T) {
		cfg := Config{
			Provider:      "claude",
			SingleFixPath: singleFixPath,
			BatchFixPath:  batchFixPath,
		}

		templates, err := Load(cfg)
		require.NoError(t, err)
		require.NotNil(t, templates)
		assert.Equal(t, customSingleFix, templates.SingleFix.Content)
		assert.Equal(t, customBatchFix, templates.BatchFix.Content)
	})

	t.Run("returns error for missing single-fix file", func(t *testing.T) {
		cfg := Config{
			Provider:      "claude",
			SingleFixPath: "/nonexistent/path/template.txt",
		}

		_, err := Load(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load single-fix template")
	})

	t.Run("returns error for missing batch-fix file", func(t *testing.T) {
		cfg := Config{
			Provider:     "claude",
			BatchFixPath: "/nonexistent/path/template.txt",
		}

		_, err := Load(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load batch-fix template")
	})
}

func TestLoad_LanguageSpecificTemplates(t *testing.T) {
	tmpDir := t.TempDir()

	// Create language-specific templates
	javaSingleFix := "Java template: {{.Language}}"
	javaBatchFix := "Java batch: {{.Language}}"
	pythonSingleFix := "Python template: {{.Language}}"

	javaSinglePath := filepath.Join(tmpDir, "java-single.txt")
	javaBatchPath := filepath.Join(tmpDir, "java-batch.txt")
	pythonSinglePath := filepath.Join(tmpDir, "python-single.txt")

	require.NoError(t, os.WriteFile(javaSinglePath, []byte(javaSingleFix), 0644))
	require.NoError(t, os.WriteFile(javaBatchPath, []byte(javaBatchFix), 0644))
	require.NoError(t, os.WriteFile(pythonSinglePath, []byte(pythonSingleFix), 0644))

	t.Run("loads language-specific templates", func(t *testing.T) {
		cfg := Config{
			Provider: "claude",
			LanguageTemplates: map[string]LanguagePaths{
				"java": {
					SingleFixPath: javaSinglePath,
					BatchFixPath:  javaBatchPath,
				},
				"python": {
					SingleFixPath: pythonSinglePath,
				},
			},
		}

		templates, err := Load(cfg)
		require.NoError(t, err)

		// Java templates should be loaded
		javaTmpl := templates.GetSingleFixTemplate("java")
		assert.Equal(t, javaSingleFix, javaTmpl.Content)

		javaBatch := templates.GetBatchFixTemplate("java")
		assert.Equal(t, javaBatchFix, javaBatch.Content)

		// Python single-fix should be loaded, batch should fallback to base
		pythonTmpl := templates.GetSingleFixTemplate("python")
		assert.Equal(t, pythonSingleFix, pythonTmpl.Content)

		pythonBatch := templates.GetBatchFixTemplate("python")
		assert.Equal(t, templates.BatchFix.Content, pythonBatch.Content) // Falls back to base
	})

	t.Run("returns error for missing language template file", func(t *testing.T) {
		cfg := Config{
			Provider: "claude",
			LanguageTemplates: map[string]LanguagePaths{
				"go": {
					SingleFixPath: "/nonexistent/go-template.txt",
				},
			},
		}

		_, err := Load(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load go single-fix template")
	})
}

func TestGetSingleFixTemplate(t *testing.T) {
	tmpDir := t.TempDir()

	javaSinglePath := filepath.Join(tmpDir, "java.txt")
	require.NoError(t, os.WriteFile(javaSinglePath, []byte("Java: {{.File}}"), 0644))

	cfg := Config{
		Provider: "claude",
		LanguageTemplates: map[string]LanguagePaths{
			"java": {
				SingleFixPath: javaSinglePath,
			},
		},
	}

	templates, err := Load(cfg)
	require.NoError(t, err)

	t.Run("returns language-specific template when exists", func(t *testing.T) {
		tmpl := templates.GetSingleFixTemplate("java")
		assert.Contains(t, tmpl.Content, "Java:")
	})

	t.Run("falls back to base template when language not found", func(t *testing.T) {
		tmpl := templates.GetSingleFixTemplate("python")
		assert.Equal(t, templates.SingleFix.Content, tmpl.Content)
	})

	t.Run("falls back to base template for empty language", func(t *testing.T) {
		tmpl := templates.GetSingleFixTemplate("")
		assert.Equal(t, templates.SingleFix.Content, tmpl.Content)
	})
}

func TestGetBatchFixTemplate(t *testing.T) {
	tmpDir := t.TempDir()

	javaBatchPath := filepath.Join(tmpDir, "java-batch.txt")
	require.NoError(t, os.WriteFile(javaBatchPath, []byte("Java batch: {{.ViolationID}}"), 0644))

	cfg := Config{
		Provider: "claude",
		LanguageTemplates: map[string]LanguagePaths{
			"java": {
				BatchFixPath: javaBatchPath,
			},
		},
	}

	templates, err := Load(cfg)
	require.NoError(t, err)

	t.Run("returns language-specific template when exists", func(t *testing.T) {
		tmpl := templates.GetBatchFixTemplate("java")
		assert.Contains(t, tmpl.Content, "Java batch:")
	})

	t.Run("falls back to base template when language not found", func(t *testing.T) {
		tmpl := templates.GetBatchFixTemplate("python")
		assert.Equal(t, templates.BatchFix.Content, tmpl.Content)
	})
}

func TestTemplateCompilation(t *testing.T) {
	t.Run("compiles valid template", func(t *testing.T) {
		tmpl := &Template{
			Name:    "test",
			Content: "Hello {{.File}}",
		}

		err := tmpl.compile()
		assert.NoError(t, err)
		assert.NotNil(t, tmpl.compiled)
	})

	t.Run("returns error for invalid template syntax", func(t *testing.T) {
		tmpl := &Template{
			Name:    "test",
			Content: "Hello {{.File",
		}

		err := tmpl.compile()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse template")
	})
}

func TestRenderSingleFix(t *testing.T) {
	tmpl := &Template{
		Name:    "single-fix",
		Content: "File: {{.File}}:{{.Line}}\nLanguage: {{.Language}}\n{{.FileContent}}",
	}
	require.NoError(t, tmpl.compile())

	data := SingleFixData{
		File:        "test.java",
		Line:        42,
		Language:    "java",
		FileContent: "public class Test {}",
	}

	t.Run("renders template with data", func(t *testing.T) {
		result, err := tmpl.RenderSingleFix(data)
		require.NoError(t, err)
		assert.Contains(t, result, "File: test.java:42")
		assert.Contains(t, result, "Language: java")
		assert.Contains(t, result, "public class Test {}")
	})

	t.Run("returns error when template not compiled", func(t *testing.T) {
		uncompiled := &Template{
			Name:    "test",
			Content: "{{.File}}",
		}

		_, err := uncompiled.RenderSingleFix(data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "template not compiled")
	})
}

func TestRenderBatchFix(t *testing.T) {
	tmpl := &Template{
		Name: "batch-fix",
		Content: `Violation: {{.ViolationID}}
Count: {{.IncidentCount}}
{{range .Incidents}}
  [{{.Index}}] {{.File}}:{{.Line}}
{{end}}`,
	}
	require.NoError(t, tmpl.compile())

	data := BatchFixData{
		ViolationID:   "test-001",
		IncidentCount: 2,
		Incidents: []BatchIncident{
			{Index: 1, File: "test1.java", Line: 10},
			{Index: 2, File: "test2.java", Line: 20},
		},
	}

	t.Run("renders batch template with data", func(t *testing.T) {
		result, err := tmpl.RenderBatchFix(data)
		require.NoError(t, err)
		assert.Contains(t, result, "Violation: test-001")
		assert.Contains(t, result, "Count: 2")
		assert.Contains(t, result, "[1] test1.java:10")
		assert.Contains(t, result, "[2] test2.java:20")
	})

	t.Run("returns error when template not compiled", func(t *testing.T) {
		uncompiled := &Template{
			Name:    "test",
			Content: "{{.ViolationID}}",
		}

		_, err := uncompiled.RenderBatchFix(data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "template not compiled")
	})
}

func TestComplexTemplateRendering(t *testing.T) {
	t.Run("single-fix with all variables", func(t *testing.T) {
		tmpl := &Template{
			Name: "complex-single",
			Content: `Category: {{.Category}}
Description: {{.Description}}
Rule: {{.RuleID}}
Message: {{.RuleMessage}}
File: {{.File}}:{{.Line}}
Language: {{.Language}}
Snippet: {{.CodeSnippet}}
Incident: {{.IncidentMessage}}
Content: {{.FileContent}}`,
		}
		require.NoError(t, tmpl.compile())

		data := SingleFixData{
			Category:       "mandatory",
			Description:    "Test description",
			RuleID:         "rule-001",
			RuleMessage:    "Test rule message",
			File:           "test.java",
			Line:           10,
			Language:       "java",
			CodeSnippet:    "import javax.servlet.*;",
			IncidentMessage: "Found javax import",
			FileContent:    "package test;\nimport javax.servlet.*;",
		}

		result, err := tmpl.RenderSingleFix(data)
		require.NoError(t, err)

		assert.Contains(t, result, "Category: mandatory")
		assert.Contains(t, result, "Description: Test description")
		assert.Contains(t, result, "Rule: rule-001")
		assert.Contains(t, result, "Message: Test rule message")
		assert.Contains(t, result, "File: test.java:10")
		assert.Contains(t, result, "Language: java")
		assert.Contains(t, result, "Snippet: import javax.servlet.*;")
		assert.Contains(t, result, "Incident: Found javax import")
		assert.Contains(t, result, "package test;")
	})

	t.Run("batch-fix with all variables and loops", func(t *testing.T) {
		tmpl := &Template{
			Name: "complex-batch",
			Content: `Violation: {{.ViolationID}}
Description: {{.Description}}
Language: {{.Language}}
Total: {{.IncidentCount}}
{{range .Incidents}}
Incident {{.Index}}:
  File: {{.File}}:{{.Line}}
  Message: {{.Message}}
  Context: {{.CodeContext}}
{{end}}`,
		}
		require.NoError(t, tmpl.compile())

		data := BatchFixData{
			ViolationID:   "violation-001",
			Description:   "Test violation",
			Language:      "java",
			IncidentCount: 2,
			Incidents: []BatchIncident{
				{
					Index:       1,
					File:        "File1.java",
					Line:        15,
					Message:     "Issue 1",
					CodeContext: "code context 1",
				},
				{
					Index:       2,
					File:        "File2.java",
					Line:        25,
					Message:     "Issue 2",
					CodeContext: "code context 2",
				},
			},
		}

		result, err := tmpl.RenderBatchFix(data)
		require.NoError(t, err)

		assert.Contains(t, result, "Violation: violation-001")
		assert.Contains(t, result, "Description: Test violation")
		assert.Contains(t, result, "Language: java")
		assert.Contains(t, result, "Total: 2")
		assert.Contains(t, result, "Incident 1:")
		assert.Contains(t, result, "File: File1.java:15")
		assert.Contains(t, result, "Message: Issue 1")
		assert.Contains(t, result, "Context: code context 1")
		assert.Contains(t, result, "Incident 2:")
		assert.Contains(t, result, "File: File2.java:25")
	})
}

func TestTemplateConditionals(t *testing.T) {
	t.Run("renders conditionals based on language", func(t *testing.T) {
		tmpl := &Template{
			Name: "conditional",
			Content: `{{if eq .Language "java"}}
Java code
{{else if eq .Language "python"}}
Python code
{{else}}
Other code
{{end}}`,
		}
		require.NoError(t, tmpl.compile())

		// Test Java
		javaData := SingleFixData{Language: "java"}
		result, err := tmpl.RenderSingleFix(javaData)
		require.NoError(t, err)
		assert.Contains(t, result, "Java code")
		assert.NotContains(t, result, "Python code")

		// Test Python
		pythonData := SingleFixData{Language: "python"}
		result, err = tmpl.RenderSingleFix(pythonData)
		require.NoError(t, err)
		assert.Contains(t, result, "Python code")
		assert.NotContains(t, result, "Java code")

		// Test other
		otherData := SingleFixData{Language: "go"}
		result, err = tmpl.RenderSingleFix(otherData)
		require.NoError(t, err)
		assert.Contains(t, result, "Other code")
	})
}

func TestLoad_MixedConfiguration(t *testing.T) {
	// Test scenario with base templates + language-specific overrides
	tmpDir := t.TempDir()

	baseSingle := "Base single: {{.File}}"
	baseBatch := "Base batch: {{.ViolationID}}"
	javaSingle := "Java single: {{.File}}"

	baseSinglePath := filepath.Join(tmpDir, "base-single.txt")
	baseBatchPath := filepath.Join(tmpDir, "base-batch.txt")
	javaSinglePath := filepath.Join(tmpDir, "java-single.txt")

	require.NoError(t, os.WriteFile(baseSinglePath, []byte(baseSingle), 0644))
	require.NoError(t, os.WriteFile(baseBatchPath, []byte(baseBatch), 0644))
	require.NoError(t, os.WriteFile(javaSinglePath, []byte(javaSingle), 0644))

	cfg := Config{
		Provider:      "claude",
		SingleFixPath: baseSinglePath,
		BatchFixPath:  baseBatchPath,
		LanguageTemplates: map[string]LanguagePaths{
			"java": {
				SingleFixPath: javaSinglePath,
				// Batch path omitted - should fall back to base
			},
		},
	}

	templates, err := Load(cfg)
	require.NoError(t, err)

	t.Run("uses language-specific single-fix for Java", func(t *testing.T) {
		tmpl := templates.GetSingleFixTemplate("java")
		assert.Equal(t, javaSingle, tmpl.Content)
	})

	t.Run("falls back to base batch-fix for Java", func(t *testing.T) {
		tmpl := templates.GetBatchFixTemplate("java")
		assert.Equal(t, baseBatch, tmpl.Content)
	})

	t.Run("uses base templates for non-Java languages", func(t *testing.T) {
		singleTmpl := templates.GetSingleFixTemplate("python")
		assert.Equal(t, baseSingle, singleTmpl.Content)

		batchTmpl := templates.GetBatchFixTemplate("python")
		assert.Equal(t, baseBatch, batchTmpl.Content)
	})
}
