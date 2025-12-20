package prompt

// Default templates for different providers
// These can be overridden via configuration

// getDefaultSingleFixTemplate returns the default single-fix template
// The same template works for all providers currently
func getDefaultSingleFixTemplate(provider string) *Template {
	// Provider-specific customization could go here in the future
	// For now, all providers use the same template
	return &Template{
		Name:    "single-fix-default",
		Content: defaultSingleFixContent,
	}
}

// getDefaultBatchFixTemplate returns the default batch-fix template
// The same template works for all providers currently
func getDefaultBatchFixTemplate(provider string) *Template {
	// Provider-specific customization could go here in the future
	// For now, all providers use the same template
	return &Template{
		Name:    "batch-fix-default",
		Content: defaultBatchFixContent,
	}
}

const defaultSingleFixContent = `You are a code migration assistant helping fix violations found by Konveyor static analysis.

VIOLATION DETAILS:
Category: {{.Category}}
Description: {{.Description}}
Rule: {{.RuleID}}
Rule Message: {{.RuleMessage}}

FILE LOCATION:
File: {{.File}}
Line: {{.Line}}

CURRENT CODE SNIPPET:
{{.CodeSnippet}}

FULL FILE CONTENT:
{{.FileContent}}

TASK:
Fix this violation by modifying the code. Return a JSON object with the following fields:
- "fixed_content": The complete fixed file content (entire file, not just changed lines)
- "confidence": A confidence score between 0.0 and 1.0 indicating how certain you are the fix is correct
- "explanation": A brief explanation of what was changed

Your response must be ONLY the JSON object, with no markdown code blocks or extra text.

Example response format:
{
  "fixed_content": "<complete file content here>",
  "confidence": 0.95,
  "explanation": "Replaced deprecated API call with modern equivalent"
}

CONFIDENCE SCORING GUIDELINES:
- 0.95-1.0: Simple mechanical changes (package renames, obvious API equivalents)
- 0.85-0.94: Straightforward changes with clear replacements
- 0.75-0.84: Changes requiring some context understanding
- 0.60-0.74: Complex changes with multiple valid approaches
- Below 0.60: Uncertain or requires significant domain knowledge

IMPORTANT:
- Return valid {{.Language}} code in the fixed_content field
- Ensure the fix is syntactically correct
- Preserve all other code unchanged
- Be honest about your confidence level`

const defaultBatchFixContent = `You are an expert code modernization assistant. Fix multiple occurrences of the same violation in a codebase.

VIOLATION: {{.ViolationID}}
DESCRIPTION: {{.Description}}

Fix the following {{.IncidentCount}} incident(s):

{{range .Incidents}}
INCIDENT {{.Index}}:
File: {{.File}}
Line: {{.Line}}
Issue: {{.Message}}
{{if .CodeContext}}
{{.CodeContext}}
{{end}}

{{end}}

OUTPUT FORMAT (JSON):
Return a JSON array with one object per incident. Each object must have:
- "incident_uri": The file URI (use the File path from above)
- "success": Boolean - true if fix succeeded, false if it failed
- "fixed_content": COMPLETE fixed file content (entire file, not diff) - required even if success is false
- "confidence": Confidence score 0.0-1.0
- "explanation": Brief explanation of the fix (or reason for failure if success is false)

Example response:
[
  {
    "incident_uri": "file:///path/to/file1.java",
    "success": true,
    "fixed_content": "<entire file content>",
    "confidence": 0.95,
    "explanation": "Replaced javax with jakarta imports"
  },
  {
    "incident_uri": "file:///path/to/file2.java",
    "success": true,
    "fixed_content": "<entire file content>",
    "confidence": 0.92,
    "explanation": "Updated servlet package references"
  }
]

CRITICAL REQUIREMENTS:
1. Return ONLY the JSON array (no markdown, no extra text)
2. Include "fixed_content" with the COMPLETE file (not just changes)
3. Ensure all {{.Language}} code is syntactically valid
4. Apply consistent fixes across all incidents
5. Be conservative with confidence scores

CONFIDENCE SCORING:
- 0.95-1.0: Mechanical changes (package/import renames)
- 0.85-0.94: Straightforward API replacements
- 0.75-0.84: Changes requiring context understanding
- 0.60-0.74: Complex changes with trade-offs
- Below 0.60: Uncertain or requires domain expertise`
