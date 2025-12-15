package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

func TestBuildSingleFixData(t *testing.T) {
	t.Run("maps all fields correctly", func(t *testing.T) {
		req := FixRequest{
			Violation: violation.Violation{
				ID:          "test-violation-001",
				Description: "Test violation description",
				Category:    "mandatory",
				Rule: violation.Rule{
					ID:      "rule-001",
					Message: "Test rule message",
				},
			},
			Incident: violation.Incident{
				URI:        "file:///path/to/test.java",
				LineNumber: 42,
				CodeSnip:   "import javax.servlet.*;",
				Message:    "Found deprecated import",
			},
			FileContent: "package test;\nimport javax.servlet.*;",
			Language:    "java",
		}

		result := BuildSingleFixData(req)

		assert.Equal(t, "mandatory", result.Category)
		assert.Equal(t, "Test violation description", result.Description)
		assert.Equal(t, "rule-001", result.RuleID)
		assert.Equal(t, "Test rule message", result.RuleMessage)
		assert.Equal(t, "file:///path/to/test.java", result.File)
		assert.Equal(t, 42, result.Line)
		assert.Equal(t, "import javax.servlet.*;", result.CodeSnippet)
		assert.Equal(t, "package test;\nimport javax.servlet.*;", result.FileContent)
		assert.Equal(t, "java", result.Language)
		assert.Equal(t, "Found deprecated import", result.IncidentMessage)
	})

	t.Run("handles empty fields", func(t *testing.T) {
		req := FixRequest{
			Violation: violation.Violation{},
			Incident:  violation.Incident{},
		}

		result := BuildSingleFixData(req)

		assert.Empty(t, result.Category)
		assert.Empty(t, result.Description)
		assert.Empty(t, result.RuleID)
		assert.Equal(t, 0, result.Line)
		assert.Empty(t, result.Language)
	})

	t.Run("handles special characters in content", func(t *testing.T) {
		req := FixRequest{
			Violation: violation.Violation{
				Description: "Fix \"quoted\" violation",
			},
			Incident: violation.Incident{
				CodeSnip: "String msg = \"Hello\\nWorld\";",
			},
			FileContent: "// Comment with special chars: <>&\nString msg = \"test\";",
		}

		result := BuildSingleFixData(req)

		assert.Equal(t, "Fix \"quoted\" violation", result.Description)
		assert.Equal(t, "String msg = \"Hello\\nWorld\";", result.CodeSnippet)
		assert.Contains(t, result.FileContent, "special chars: <>&")
	})
}

func TestBuildBatchFixData(t *testing.T) {
	t.Run("builds batch data with multiple incidents", func(t *testing.T) {
		req := BatchRequest{
			Violation: violation.Violation{
				ID:          "batch-violation-001",
				Description: "Batch violation description",
			},
			Incidents: []violation.Incident{
				{
					URI:        "file:///path/to/file1.java",
					LineNumber: 10,
					Message:    "Issue in file1",
				},
				{
					URI:        "file:///path/to/file2.java",
					LineNumber: 20,
					Message:    "Issue in file2",
				},
			},
			FileContents: map[string]string{
				"/path/to/file1.java": "line1\nline2\nline3",
				"/path/to/file2.java": "line1\nline2\nline3",
			},
			Language: "java",
		}

		result := BuildBatchFixData(req)

		assert.Equal(t, "batch-violation-001", result.ViolationID)
		assert.Equal(t, "Batch violation description", result.Description)
		assert.Equal(t, 2, result.IncidentCount)
		assert.Equal(t, "java", result.Language)
		assert.Len(t, result.Incidents, 2)

		// Check first incident
		assert.Equal(t, 1, result.Incidents[0].Index)
		assert.Equal(t, "/path/to/file1.java", result.Incidents[0].File)
		assert.Equal(t, 10, result.Incidents[0].Line)
		assert.Equal(t, "Issue in file1", result.Incidents[0].Message)

		// Check second incident
		assert.Equal(t, 2, result.Incidents[1].Index)
		assert.Equal(t, "/path/to/file2.java", result.Incidents[1].File)
		assert.Equal(t, 20, result.Incidents[1].Line)
		assert.Equal(t, "Issue in file2", result.Incidents[1].Message)
	})

	t.Run("includes code context for incidents", func(t *testing.T) {
		fileContent := `line 1
line 2
line 3
line 4
line 5
line 6
line 7
line 8
line 9
line 10`

		req := BatchRequest{
			Violation: violation.Violation{
				ID: "test-001",
			},
			Incidents: []violation.Incident{
				{
					URI:        "file:///test.java",
					LineNumber: 5,
					Message:    "Issue at line 5",
				},
			},
			FileContents: map[string]string{
				"/test.java": fileContent,
			},
			Language: "java",
		}

		result := BuildBatchFixData(req)

		assert.Len(t, result.Incidents, 1)
		assert.NotEmpty(t, result.Incidents[0].CodeContext)
		assert.Contains(t, result.Incidents[0].CodeContext, "```java")
		assert.Contains(t, result.Incidents[0].CodeContext, ">>> line 5") // Marked line
		assert.Contains(t, result.Incidents[0].CodeContext, "line 1")     // Context before
		assert.Contains(t, result.Incidents[0].CodeContext, "line 9")     // Context after
	})

	t.Run("handles missing file content", func(t *testing.T) {
		req := BatchRequest{
			Violation: violation.Violation{
				ID: "test-001",
			},
			Incidents: []violation.Incident{
				{
					URI:        "file:///missing.java",
					LineNumber: 10,
					Message:    "Issue in missing file",
				},
			},
			FileContents: map[string]string{}, // Empty map
			Language:     "java",
		}

		result := BuildBatchFixData(req)

		assert.Len(t, result.Incidents, 1)
		assert.Empty(t, result.Incidents[0].CodeContext) // No context available
	})

	t.Run("handles empty incidents list", func(t *testing.T) {
		req := BatchRequest{
			Violation: violation.Violation{
				ID: "test-001",
			},
			Incidents:    []violation.Incident{},
			FileContents: map[string]string{},
			Language:     "java",
		}

		result := BuildBatchFixData(req)

		assert.Equal(t, 0, result.IncidentCount)
		assert.Len(t, result.Incidents, 0)
	})

	t.Run("preserves incident order", func(t *testing.T) {
		req := BatchRequest{
			Violation: violation.Violation{
				ID: "test-001",
			},
			Incidents: []violation.Incident{
				{URI: "file:///a.java", LineNumber: 1, Message: "First"},
				{URI: "file:///b.java", LineNumber: 2, Message: "Second"},
				{URI: "file:///c.java", LineNumber: 3, Message: "Third"},
			},
			FileContents: map[string]string{},
			Language:     "java",
		}

		result := BuildBatchFixData(req)

		assert.Equal(t, 1, result.Incidents[0].Index)
		assert.Equal(t, "First", result.Incidents[0].Message)
		assert.Equal(t, 2, result.Incidents[1].Index)
		assert.Equal(t, "Second", result.Incidents[1].Message)
		assert.Equal(t, 3, result.Incidents[2].Index)
		assert.Equal(t, "Third", result.Incidents[2].Message)
	})
}

func TestBuildCodeContext(t *testing.T) {
	t.Run("extracts 5 lines before and after", func(t *testing.T) {
		content := `line 1
line 2
line 3
line 4
line 5
line 6
line 7
line 8
line 9
line 10
line 11
line 12`

		result := buildCodeContext(content, 7, "java")

		// Line 7 is at index 6, so start=max(0, 7-5)=2, end=min(12, 7+5)=12
		// Should include lines 3-12 (indices 2-11)
		assert.Contains(t, result, "```java")
		assert.Contains(t, result, "line 3")     // Start of context
		assert.Contains(t, result, ">>> line 7") // Marked line
		assert.Contains(t, result, "line 12")    // End of context
		assert.NotContains(t, result, ">>> line 1")  // Line 1 should not be marked
		assert.NotContains(t, result, ">>> line 2")  // Line 2 should not be marked
		assert.NotContains(t, result, "\nline 1\n") // Line 1 as separate line
		assert.NotContains(t, result, "\nline 2\n") // Line 2 as separate line
	})

	t.Run("handles line at beginning of file", func(t *testing.T) {
		content := `line 1
line 2
line 3
line 4
line 5`

		result := buildCodeContext(content, 1, "python")

		assert.Contains(t, result, "```python")
		assert.Contains(t, result, ">>> line 1") // First line marked
		assert.Contains(t, result, "line 2")
		assert.Contains(t, result, "line 5")
		assert.NotContains(t, result, "line 0") // No negative index
	})

	t.Run("handles line at end of file", func(t *testing.T) {
		content := `line 1
line 2
line 3
line 4
line 5`

		result := buildCodeContext(content, 5, "go")

		assert.Contains(t, result, "```go")
		assert.Contains(t, result, "line 1")
		assert.Contains(t, result, ">>> line 5") // Last line marked
		assert.NotContains(t, result, "line 6")  // No extra lines
	})

	t.Run("marks correct line with arrow", func(t *testing.T) {
		content := `line 1
line 2
target line
line 4
line 5`

		result := buildCodeContext(content, 3, "javascript")

		assert.Contains(t, result, ">>> target line")
		assert.NotContains(t, result, ">>> line 1")
		assert.NotContains(t, result, ">>> line 2")
		assert.NotContains(t, result, ">>> line 4")
	})

	t.Run("handles single line file", func(t *testing.T) {
		content := "only one line"

		result := buildCodeContext(content, 1, "java")

		assert.Contains(t, result, "```java")
		assert.Contains(t, result, ">>> only one line")
	})

	t.Run("includes language in code fence", func(t *testing.T) {
		content := "test"

		javaResult := buildCodeContext(content, 1, "java")
		assert.Contains(t, javaResult, "```java")

		pythonResult := buildCodeContext(content, 1, "python")
		assert.Contains(t, pythonResult, "```python")

		goResult := buildCodeContext(content, 1, "go")
		assert.Contains(t, goResult, "```go")
	})

	t.Run("closes code fence", func(t *testing.T) {
		content := "line 1\nline 2"

		result := buildCodeContext(content, 1, "java")

		assert.Contains(t, result, "```java")
		assert.Contains(t, result, "```", "Should close code fence")
		// Should have opening and closing
		openCount := 0
		closeCount := 0
		for i := 0; i < len(result)-2; i++ {
			if result[i:i+3] == "```" {
				if i+3 < len(result) && result[i+3] != '\n' {
					openCount++
				} else {
					closeCount++
				}
			}
		}
		assert.Equal(t, 1, openCount, "Should have one opening fence")
		assert.Equal(t, 1, closeCount, "Should have one closing fence")
	})
}

func TestSplitLines(t *testing.T) {
	t.Run("splits on newlines", func(t *testing.T) {
		content := "line 1\nline 2\nline 3"

		result := splitLines(content)

		assert.Len(t, result, 3)
		assert.Equal(t, "line 1", result[0])
		assert.Equal(t, "line 2", result[1])
		assert.Equal(t, "line 3", result[2])
	})

	t.Run("handles empty string", func(t *testing.T) {
		result := splitLines("")

		assert.Len(t, result, 0)
	})

	t.Run("handles single line without newline", func(t *testing.T) {
		result := splitLines("single line")

		assert.Len(t, result, 1)
		assert.Equal(t, "single line", result[0])
	})

	t.Run("handles trailing newline", func(t *testing.T) {
		content := "line 1\nline 2\n"

		result := splitLines(content)

		assert.Len(t, result, 2)
		assert.Equal(t, "line 1", result[0])
		assert.Equal(t, "line 2", result[1])
	})

	t.Run("handles multiple consecutive newlines", func(t *testing.T) {
		content := "line 1\n\n\nline 4"

		result := splitLines(content)

		assert.Len(t, result, 4)
		assert.Equal(t, "line 1", result[0])
		assert.Equal(t, "", result[1])
		assert.Equal(t, "", result[2])
		assert.Equal(t, "line 4", result[3])
	})

	t.Run("handles Windows-style line endings", func(t *testing.T) {
		// Note: This splits on \n, so \r will be part of the line
		content := "line 1\r\nline 2\r\n"

		result := splitLines(content)

		assert.Len(t, result, 2)
		assert.Equal(t, "line 1\r", result[0]) // \r is preserved
		assert.Equal(t, "line 2\r", result[1])
	})

	t.Run("handles only newlines", func(t *testing.T) {
		content := "\n\n\n"

		result := splitLines(content)

		assert.Len(t, result, 3)
		assert.Equal(t, "", result[0])
		assert.Equal(t, "", result[1])
		assert.Equal(t, "", result[2])
	})
}

func TestMaxMin(t *testing.T) {
	t.Run("max returns larger value", func(t *testing.T) {
		assert.Equal(t, 10, max(5, 10))
		assert.Equal(t, 10, max(10, 5))
		assert.Equal(t, 5, max(5, 5))
		assert.Equal(t, 0, max(-5, 0))
		assert.Equal(t, 100, max(100, 99))
	})

	t.Run("min returns smaller value", func(t *testing.T) {
		assert.Equal(t, 5, min(5, 10))
		assert.Equal(t, 5, min(10, 5))
		assert.Equal(t, 5, min(5, 5))
		assert.Equal(t, -5, min(-5, 0))
		assert.Equal(t, 99, min(100, 99))
	})

	t.Run("handles negative numbers", func(t *testing.T) {
		assert.Equal(t, 0, max(-10, 0))
		assert.Equal(t, -5, max(-10, -5))
		assert.Equal(t, -10, min(-10, 0))
		assert.Equal(t, -10, min(-10, -5))
	})
}

func TestCodeContextEdgeCases(t *testing.T) {
	t.Run("line number beyond file length", func(t *testing.T) {
		content := "line 1\nline 2\nline 3"

		// Line 100 is beyond the file
		result := buildCodeContext(content, 100, "java")

		// Should still produce valid output without crashing
		assert.Contains(t, result, "```java")
		assert.Contains(t, result, "```")
	})

	t.Run("line number zero", func(t *testing.T) {
		content := "line 1\nline 2\nline 3"

		result := buildCodeContext(content, 0, "java")

		// Should handle gracefully
		assert.Contains(t, result, "```java")
	})

	t.Run("negative line number", func(t *testing.T) {
		content := "line 1\nline 2\nline 3"

		result := buildCodeContext(content, -5, "java")

		// Should handle gracefully without crashing
		assert.Contains(t, result, "```java")
	})
}

func TestBatchFixDataWithRealWorldScenario(t *testing.T) {
	t.Run("javax to jakarta migration", func(t *testing.T) {
		servletFile := `package com.example.web;

import javax.servlet.http.HttpServlet;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;

public class MyServlet extends HttpServlet {
    protected void doGet(HttpServletRequest request, HttpServletResponse response) {
        // Implementation
    }
}`

		filterFile := `package com.example.web;

import javax.servlet.Filter;
import javax.servlet.FilterChain;

public class MyFilter implements Filter {
    // Implementation
}`

		req := BatchRequest{
			Violation: violation.Violation{
				ID:          "javax-to-jakarta-001",
				Description: "Replace javax.servlet with jakarta.servlet",
				Category:    "mandatory",
			},
			Incidents: []violation.Incident{
				{
					URI:        "file:///src/MyServlet.java",
					LineNumber: 3,
					Message:    "Found javax.servlet.http.HttpServlet",
				},
				{
					URI:        "file:///src/MyFilter.java",
					LineNumber: 3,
					Message:    "Found javax.servlet.Filter",
				},
			},
			FileContents: map[string]string{
				"/src/MyServlet.java": servletFile,
				"/src/MyFilter.java":  filterFile,
			},
			Language: "java",
		}

		result := BuildBatchFixData(req)

		assert.Equal(t, "javax-to-jakarta-001", result.ViolationID)
		assert.Equal(t, "Replace javax.servlet with jakarta.servlet", result.Description)
		assert.Equal(t, 2, result.IncidentCount)
		assert.Equal(t, "java", result.Language)

		// Verify incidents
		assert.Equal(t, "/src/MyServlet.java", result.Incidents[0].File)
		assert.Contains(t, result.Incidents[0].CodeContext, "javax.servlet.http.HttpServlet")

		assert.Equal(t, "/src/MyFilter.java", result.Incidents[1].File)
		assert.Contains(t, result.Incidents[1].CodeContext, "javax.servlet.Filter")
	})
}
