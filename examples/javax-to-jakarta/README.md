# Example: javax.servlet → jakarta.servlet

This is a simple test case for validating AI-powered remediation.

## The Violation

Java EE was rebranded to Jakarta EE, requiring all `javax.*` packages to be renamed to `jakarta.*`.

**Before:**
```java
import javax.servlet.http.HttpServlet;
```

**After:**
```java
import jakarta.servlet.http.HttpServlet;
```

## Testing

```bash
# Run from kantra-ai root directory

# Dry run first
./kantra-ai remediate \
  --analysis=./examples/javax-to-jakarta/output.yaml \
  --input=./examples/javax-to-jakarta \
  --provider=claude \
  --dry-run

# Actually fix it
./kantra-ai remediate \
  --analysis=./examples/javax-to-jakarta/output.yaml \
  --input=./examples/javax-to-jakarta \
  --provider=claude

# Compare with expected result
diff ./examples/javax-to-jakarta/src/UserServlet.java \
     ./examples/javax-to-jakarta/expected/UserServlet.java
```

## Expected Outcome

✅ **Success**: AI should correctly replace all `javax.servlet.*` imports with `jakarta.servlet.*`

**Success criteria:**
- All 4 javax.servlet imports replaced
- Code still compiles (if you have Java installed)
- No other code changed
- Cost: ~$0.05-0.10
