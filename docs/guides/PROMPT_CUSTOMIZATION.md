# Prompt Customization Guide

This guide covers how to customize AI prompts in kantra-ai for optimized migration results.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Template System](#template-system)
- [Template Variables](#template-variables)
- [Language-Specific Templates](#language-specific-templates)
- [Example Templates](#example-templates)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

kantra-ai uses AI prompts to guide how violations are fixed. The default prompts work well for general migrations, but you can customize them for:

- **Technology-specific migrations**: Java (javax→jakarta), Python (2→3), etc.
- **Company coding standards**: Enforce specific patterns or styles
- **Domain-specific guidance**: Security, performance, compliance requirements
- **Cost optimization**: Shorter prompts for simple fixes, detailed prompts for complex changes

### Why Customize Prompts?

**Better Results**:
- AI understands your specific migration context
- More consistent fixes across your codebase
- Higher confidence scores for familiar patterns

**Cost Savings**:
- Shorter, focused prompts reduce token usage
- Better guidance reduces retry attempts

**Compliance**:
- Enforce company coding standards
- Include security best practices
- Add regulatory requirements

## Quick Start

### Step 1: Create Prompt Directory

```bash
mkdir -p prompts
```

### Step 2: Create a Custom Template

Create `prompts/java-jakarta.txt`:

```
You are a Java migration expert specializing in javax → jakarta EE migrations.

VIOLATION: {{.RuleID}}
Description: {{.Description}}

FILE: {{.File}}:{{.Line}}

CURRENT CODE:
{{.CodeSnippet}}

MIGRATION TASK:
1. Replace javax.* imports with jakarta.* equivalents
2. Update annotations package references
3. Ensure Jakarta EE 9+ compatibility
4. Preserve formatting and comments

FULL FILE CONTENT:
{{.FileContent}}

RESPONSE FORMAT (JSON only):
{
  "fixed_content": "<complete fixed file>",
  "confidence": 0.0-1.0,
  "explanation": "<brief description of changes>"
}

CONFIDENCE GUIDELINES:
- 0.95-1.0: Simple package rename
- 0.85-0.94: Straightforward API update
- 0.75-0.84: Multiple related changes
- Below 0.75: Complex or uncertain changes
```

### Step 3: Configure kantra-ai

Add to `.kantra-ai.yaml`:

```yaml
prompts:
  language-templates:
    java:
      single-fix: ./prompts/java-jakarta.txt
```

### Step 4: Test It

```bash
./kantra-ai remediate \
  --analysis=output.yaml \
  --input=./src \
  --dry-run
```

Review the fixes to ensure your custom prompt is working as expected.

## Template System

### Template Types

kantra-ai uses two types of prompts:

1. **Single-fix templates**: Fix one incident at a time
   - Used for most violations
   - Gets full file context
   - Most precise fixes

2. **Batch-fix templates**: Fix multiple incidents together
   - Used when `--batch-size > 1`
   - Processes up to 10 incidents in one call
   - 50-80% cost savings

### Template Hierarchy

Templates are selected in this order:

```
1. Language-specific template (if exists)
   ├─ single-fix: ./prompts/java-fix.txt
   └─ batch-fix: ./prompts/java-batch.txt

2. Base template (if exists)
   ├─ single-fix: ./prompts/base-fix.txt
   └─ batch-fix: ./prompts/base-batch.txt

3. Built-in default (always available)
   ├─ Default single-fix prompt
   └─ Default batch-fix prompt
```

### Configuration Structure

```yaml
prompts:
  # Base templates (used as fallback for all languages)
  single-fix-template: ./prompts/base-fix.txt
  batch-fix-template: ./prompts/base-batch.txt

  # Language-specific overrides
  language-templates:
    java:
      single-fix: ./prompts/java-fix.txt
      batch-fix: ./prompts/java-batch.txt
    python:
      single-fix: ./prompts/python-fix.txt
      batch-fix: ./prompts/python-batch.txt
    go:
      single-fix: ./prompts/go-fix.txt
      # batch-fix omitted → falls back to base-batch.txt
```

## Template Variables

### Single-Fix Template Variables

Available in templates for fixing individual incidents:

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `{{.Category}}` | string | Violation category | `mandatory`, `optional`, `potential` |
| `{{.Description}}` | string | Violation description | `Replace javax.servlet with jakarta.servlet` |
| `{{.RuleID}}` | string | Rule identifier | `javax-to-jakarta-001` |
| `{{.RuleMessage}}` | string | Rule message | `javax.servlet has been replaced by jakarta.servlet` |
| `{{.File}}` | string | File path | `src/main/java/Controller.java` |
| `{{.Line}}` | int | Line number | `15` |
| `{{.CodeSnippet}}` | string | Code at violation | `import javax.servlet.*;` |
| `{{.FileContent}}` | string | Full file content | `package com.example;\n\nimport...` |
| `{{.Language}}` | string | Programming language | `java`, `python`, `go`, `javascript` |
| `{{.IncidentMessage}}` | string | Specific incident message | `Found use of javax.servlet.HttpServlet` |

### Batch-Fix Template Variables

Available in templates for fixing multiple incidents:

| Variable | Type | Description |
|----------|------|-------------|
| `{{.ViolationID}}` | string | Violation ID (same for all incidents in batch) |
| `{{.Description}}` | string | Violation description |
| `{{.IncidentCount}}` | int | Number of incidents in this batch |
| `{{.Language}}` | string | Programming language |
| `{{.Incidents}}` | array | Array of incidents (see below) |

**Incident Array Fields** (`{{.Incidents}}`):

| Field | Type | Description |
|-------|------|-------------|
| `{{.Index}}` | int | 1-based index |
| `{{.File}}` | string | File path |
| `{{.Line}}` | int | Line number |
| `{{.Message}}` | string | Incident message |
| `{{.CodeContext}}` | string | Code context around the incident |

### Template Syntax

kantra-ai uses Go's `text/template` syntax:

```
Basic substitution:
{{.Variable}}

Conditionals:
{{if eq .Language "java"}}
  This is Java code
{{else if eq .Language "python"}}
  This is Python code
{{else}}
  Other language
{{end}}

Loops (for batch templates):
{{range .Incidents}}
  Incident {{.Index}}: {{.File}}:{{.Line}}
  {{.CodeContext}}
{{end}}

Comments (not included in output):
{{/* This is a comment */}}
```

## Language-Specific Templates

### Supported Languages

kantra-ai automatically detects language from file extensions:

| Language | Extensions | Variable Value |
|----------|------------|----------------|
| Java | `.java` | `java` |
| Python | `.py` | `python` |
| Go | `.go` | `go` |
| JavaScript | `.js` | `javascript` |
| TypeScript | `.ts` | `typescript` |
| XML | `.xml` | `xml` |
| YAML | `.yaml`, `.yml` | `yaml` |
| Properties | `.properties` | `properties` |

### Creating Language-Specific Templates

**Example: Java Template** (`prompts/java-fix.txt`):

```
You are a Java migration expert.

{{.RuleMessage}}

FILE: {{.File}}:{{.Line}}
SNIPPET: {{.CodeSnippet}}

JAVA-SPECIFIC GUIDELINES:
- Follow Java naming conventions (camelCase, PascalCase)
- Maintain proper imports ordering
- Preserve JavaDoc comments
- Ensure thread safety where applicable

{{.FileContent}}

JSON response required.
```

**Example: Python Template** (`prompts/python-fix.txt`):

```
You are a Python migration expert.

{{.RuleMessage}}

FILE: {{.File}}:{{.Line}}
SNIPPET: {{.CodeSnippet}}

PYTHON-SPECIFIC GUIDELINES:
- Follow PEP 8 style guide
- Maintain proper indentation (4 spaces)
- Preserve docstrings
- Use type hints where beneficial

{{.FileContent}}

JSON response required.
```

## Example Templates

### Minimal Template

For simple, low-cost migrations:

```
Fix this {{.Language}} code violation:

{{.Description}}

File: {{.File}}:{{.Line}}
Code:
{{.CodeSnippet}}

Full file:
{{.FileContent}}

Return JSON: {"fixed_content": "...", "confidence": 0.0-1.0, "explanation": "..."}
```

### Security-Focused Template

For migrations with security requirements:

```
You are a security-focused code migration expert.

VIOLATION: {{.RuleID}}
{{.Description}}

FILE: {{.File}}:{{.Line}}

SECURITY REQUIREMENTS:
1. Validate all inputs
2. Use parameterized queries (no SQL injection)
3. Sanitize outputs (no XSS)
4. Follow principle of least privilege
5. No hardcoded credentials

CURRENT CODE:
{{.CodeSnippet}}

FULL FILE:
{{.FileContent}}

Fix the violation while maintaining or improving security posture.

JSON response with confidence score and security impact assessment.
```

### Company Standards Template

For enforcing company-specific patterns:

```
You are a code migration expert for ACME Corporation.

ACME CODING STANDARDS:
- All classes must have Javadoc headers
- Use ACME logging framework (com.acme.logging.Logger)
- Error handling: All exceptions must extend AcmeException
- Performance: Use ACME connection pooling

VIOLATION: {{.RuleID}}
{{.Description}}

FILE: {{.File}}:{{.Line}}
CODE: {{.CodeSnippet}}

FULL FILE:
{{.FileContent}}

Fix while adhering to ACME standards.

JSON: {"fixed_content": "...", "confidence": 0.0-1.0, "explanation": "..."}
```

### Batch Template Example

For processing multiple incidents efficiently:

```
You are a {{.Language}} migration expert. Fix {{.IncidentCount}} related violations.

VIOLATION: {{.ViolationID}}
Description: {{.Description}}

INCIDENTS TO FIX:
{{range .Incidents}}
[{{.Index}}/{{$.IncidentCount}}] {{.File}}:{{.Line}}
Issue: {{.Message}}
Context:
{{.CodeContext}}

{{end}}

Fix all {{.IncidentCount}} incidents consistently.

Return JSON array:
[
  {
    "incident_uri": "{{.Incidents.0.File}}:{{.Incidents.0.Line}}",
    "success": true,
    "fixed_content": "...",
    "confidence": 0.0-1.0,
    "explanation": "..."
  },
  ...
]
```

## Best Practices

### 1. Start with Base Template

Begin with a base template, then add language-specific overrides:

```yaml
prompts:
  single-fix-template: ./prompts/base-fix.txt  # Used by all languages
  language-templates:
    java:
      single-fix: ./prompts/java-fix.txt        # Java overrides base
```

### 2. Keep Prompts Focused

**Bad** (too generic):
```
Fix this code. Make it better. Consider performance, security, and best practices.
```

**Good** (specific):
```
Replace deprecated javax.servlet imports with jakarta.servlet equivalents.
Preserve all existing functionality.
```

### 3. Include Examples

AI performs better with concrete examples:

```
EXAMPLE TRANSFORMATION:
Before: import javax.servlet.HttpServlet;
After:  import jakarta.servlet.HttpServlet;
```

### 4. Specify Response Format Clearly

Always specify JSON format and required fields:

```
REQUIRED JSON RESPONSE:
{
  "fixed_content": "<entire file content>",
  "confidence": <0.0 to 1.0>,
  "explanation": "<what changed>"
}

DO NOT include markdown code blocks (```json).
ONLY return the JSON object.
```

### 5. Set Confidence Guidelines

Help AI provide accurate confidence scores:

```
CONFIDENCE SCORING:
- 0.95-1.0: Mechanical change (package rename)
- 0.85-0.94: Straightforward (obvious API equivalent)
- 0.75-0.84: Requires understanding (multiple changes)
- 0.60-0.74: Complex (architectural impact)
- Below 0.60: Uncertain (needs expert review)
```

### 6. Test Incrementally

1. Start with one language
2. Test on small sample (--dry-run)
3. Review results
4. Refine prompt
5. Expand to more languages

### 7. Monitor Confidence Scores

Track confidence scores to measure prompt effectiveness:

```bash
# Enable confidence filtering to see scores
./kantra-ai remediate \
  --enable-confidence \
  --analysis=output.yaml \
  --input=./src
```

Low confidence scores may indicate:
- Prompt is too generic
- Missing domain-specific guidance
- Complex violations needing manual review

## Troubleshooting

### Problem: AI Returns Wrong Format

**Symptom**: Errors like "failed to parse JSON"

**Solution**: Make response format very explicit:

```
CRITICAL: Your response must be ONLY the JSON object.
DO NOT include:
- Markdown code blocks (```json)
- Explanatory text before or after
- Multiple JSON objects

CORRECT:
{"fixed_content": "...", "confidence": 0.95, "explanation": "..."}

INCORRECT:
```json
{"fixed_content": "..."}
```
```

### Problem: Low Confidence Scores

**Symptom**: Many fixes skipped due to low confidence

**Solutions**:

1. **Add domain expertise** to prompt:
   ```
   CONTEXT: This is a Spring Boot application migrating from Java EE 8 to Jakarta EE 9.
   ```

2. **Include examples**:
   ```
   EXAMPLE: javax.persistence → jakarta.persistence
   EXAMPLE: javax.servlet → jakarta.servlet
   ```

3. **Set clearer guidelines**:
   ```
   For simple package renames: confidence should be 0.95+
   For API changes with documentation: confidence should be 0.85+
   ```

### Problem: Inconsistent Fixes

**Symptom**: Similar violations fixed differently

**Solution**: Use batch processing with consistent examples:

```yaml
prompts:
  batch-fix-template: ./prompts/consistent-batch.txt
```

```
Process all {{.IncidentCount}} incidents CONSISTENTLY using the same pattern:

PATTERN:
1. Find javax.* import
2. Replace with jakarta.* equivalent
3. Verify no other changes needed

Apply this pattern to ALL incidents identically.
```

### Problem: Template Not Loading

**Symptom**: Errors like "failed to load template"

**Checks**:

1. **Verify file path** is correct (relative to working directory):
   ```bash
   ls -la ./prompts/java-fix.txt
   ```

2. **Check YAML syntax**:
   ```yaml
   prompts:
     language-templates:
       java:  # Correct indentation
         single-fix: ./prompts/java-fix.txt
   ```

3. **Check file permissions**:
   ```bash
   chmod 644 ./prompts/java-fix.txt
   ```

### Problem: Template Variables Not Substituting

**Symptom**: Output contains `{{.File}}` instead of actual filename

**Solution**: Check template syntax:

```
WRONG: {{File}}         # Missing dot
RIGHT: {{.File}}        # Correct

WRONG: {{ .File }}      # Extra spaces
RIGHT: {{.File}}        # No spaces (or consistent spacing)
```

### Problem: Batch Template Not Working

**Symptom**: Batch fixes failing or producing wrong output

**Check**:

1. **Verify batch is enabled**:
   ```bash
   ./kantra-ai remediate --batch-size=10
   ```

2. **Check incident loop syntax**:
   ```
   CORRECT:
   {{range .Incidents}}
     File: {{.File}}
   {{end}}

   WRONG:
   {{range Incidents}}    # Missing dot
   {{range .Incident}}    # Wrong field name
   ```

3. **Access parent fields in loop**:
   ```
   {{range .Incidents}}
     Violation: {{$.ViolationID}}    # Use $ for parent context
     File: {{.File}}                  # Use . for current incident
   {{end}}
   ```

## Advanced Topics

### Conditional Prompts

Customize prompts based on violation type:

```
{{if eq .Category "mandatory"}}
PRIORITY: HIGH - This is a mandatory migration requirement.
{{else if eq .Category "optional"}}
PRIORITY: MEDIUM - This is an optional improvement.
{{else}}
PRIORITY: LOW - This is a potential issue.
{{end}}
```

### Dynamic Confidence Thresholds

Suggest different confidence thresholds per complexity:

```
CONFIDENCE REQUIREMENTS:
{{if eq .Language "java"}}
- Simple imports: 0.95+
- API changes: 0.85+
- Architecture: 0.75+
{{else if eq .Language "python"}}
- Syntax changes: 0.90+
- Library updates: 0.80+
{{end}}
```

### Multi-Language Projects

Create a single template for multi-language projects:

```
You are a migration expert for {{.Language}} code.

{{if eq .Language "java"}}
JAVA GUIDELINES:
- camelCase naming
- JavaDoc comments
{{else if eq .Language "python"}}
PYTHON GUIDELINES:
- snake_case naming
- PEP 8 compliance
{{else if eq .Language "javascript"}}
JAVASCRIPT GUIDELINES:
- Prefer const/let over var
- Use arrow functions
{{end}}

FILE: {{.File}}:{{.Line}}
{{.FileContent}}
```

## Resources

- **Example Templates**: [.kantra-ai.example.yaml](../../.kantra-ai.example.yaml)
- **Built-in Defaults**: [pkg/prompt/defaults.go](../../pkg/prompt/defaults.go)
- **Template Syntax**: [Go text/template documentation](https://pkg.go.dev/text/template)
- **Main Documentation**: [README.md](../../README.md)

## Questions?

For issues or questions about prompt customization:

1. Check [Troubleshooting](#troubleshooting) section above
2. Review [example templates](#example-templates)
3. Open an issue on [GitHub](https://github.com/tsanders-rh/kantra-ai/issues)
