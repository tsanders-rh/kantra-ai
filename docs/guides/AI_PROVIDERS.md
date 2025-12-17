# AI Providers Guide

kantra-ai supports **50+ LLM providers** through a combination of native implementations and OpenAI-compatible APIs.

## Quick Comparison

| Provider | Speed | Cost | Quality | Privacy | Best For |
|----------|-------|------|---------|---------|----------|
| Claude | Medium | $$$ | Excellent | Cloud | Production use, highest quality |
| GPT-4 | Medium | $$$$ | Excellent | Cloud | Production use |
| Groq | Very Fast | $-$$ | Good | Cloud | Fast iteration, testing |
| Ollama | Fast | Free | Good | Local | Privacy, offline, cost-sensitive |
| Together | Fast | $-$$ | Good | Cloud | Open source models |
| OpenRouter | Varies | Varies | Varies | Cloud | Model exploration |

---

## Native Providers

### Claude (Anthropic) - Recommended

Best quality for code fixes with batch processing support.

**Setup:**
```bash
export ANTHROPIC_API_KEY=sk-ant-...
```

**Usage:**
```bash
./kantra-ai remediate \
  --provider=claude \
  --model=claude-sonnet-4-20250514
```

**Available Models:**
- `claude-sonnet-4-20250514` (default) - Best balance of speed/quality
- `claude-opus-4-20250514` - Highest quality, slower
- `claude-3-5-sonnet-20241022` - Previous generation

**Features:**
- ✅ Batch processing (50-80% cost savings)
- ✅ Plan generation
- ✅ High quality code fixes
- ✅ Long context windows (200k tokens)

**Pricing (per 1M tokens):**
- Input: $3.00
- Output: $15.00

**Best For:**
- Production migrations
- Complex refactoring
- High-quality results

---

### OpenAI

High-quality fixes with GPT-4 and GPT-3.5 models.

**Setup:**
```bash
export OPENAI_API_KEY=sk-...
```

**Usage:**
```bash
./kantra-ai remediate \
  --provider=openai \
  --model=gpt-4
```

**Available Models:**
- `gpt-4` - Highest quality
- `gpt-4-turbo` - Faster GPT-4
- `gpt-3.5-turbo` - Cheaper, faster, lower quality

**Features:**
- ✅ Batch processing (50-80% cost savings)
- ⚠️ Plan generation not yet supported
- ✅ High quality code fixes

**Pricing (per 1M tokens):**
- GPT-4: $30.00 input / $60.00 output
- GPT-3.5 Turbo: $0.50 input / $1.50 output

**Best For:**
- Production migrations (if not using Claude)
- Organizations already using OpenAI

---

## OpenAI-Compatible Providers

These providers use the OpenAI API format, making them easy to integrate.

### Groq - Ultra-Fast Inference

Fastest inference speeds with free tier available.

**Setup:**
```bash
export OPENAI_API_KEY=gsk_...
```

**Usage:**
```bash
./kantra-ai remediate \
  --provider=groq \
  --model=llama-3.1-70b-versatile
```

**Available Models:**
- `llama-3.1-70b-versatile` - Best for code (recommended)
- `llama-3.1-8b-instant` - Faster, lower quality
- `mixtral-8x7b-32768` - Alternative option
- `gemma-7b-it` - Smaller model

**Features:**
- ✅ Extremely fast inference
- ✅ Free tier with rate limits
- ✅ Batch processing
- ⚠️ Plan generation not yet supported

**Pricing:**
- Free tier: 30 requests/minute
- Paid tier: Very affordable

**Best For:**
- Fast iteration during development
- Testing and experimentation
- Cost-sensitive projects

---

### Ollama - Local Models

Run models locally for free and complete privacy.

**Setup:**
```bash
ollama serve  # No API key needed
```

**Usage:**
```bash
./kantra-ai remediate \
  --provider=ollama \
  --model=codellama
```

**Popular Models:**
- `codellama` - Meta's code-focused model (recommended)
- `llama3.1` - General purpose, good for code
- `deepseek-coder` - Specialized for code
- `qwen2.5-coder` - Another code specialist

**Install Models:**
```bash
ollama pull codellama
ollama pull llama3.1
ollama pull deepseek-coder
```

**Features:**
- ✅ 100% free
- ✅ Complete privacy (runs locally)
- ✅ No API rate limits
- ✅ Batch processing
- ⚠️ Slower than cloud providers
- ⚠️ Plan generation not yet supported

**Requirements:**
- 8GB+ RAM (16GB recommended)
- GPU optional but recommended

**Best For:**
- Privacy-sensitive codebases
- Offline development
- Zero API costs
- Learning and experimentation

---

### Together AI - Open Source Models

Wide selection of open source models at competitive prices.

**Setup:**
```bash
export OPENAI_API_KEY=...  # Together AI API key
```

**Usage:**
```bash
./kantra-ai remediate \
  --provider=together \
  --model=meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo
```

**Popular Models:**
- `meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo` - Best quality
- `meta-llama/Meta-Llama-3.1-8B-Instruct-Turbo` - Faster
- `codellama/CodeLlama-34b-Instruct-hf` - Code-focused
- `mistralai/Mixtral-8x7B-Instruct-v0.1` - Alternative

**Features:**
- ✅ Many open source models
- ✅ Competitive pricing
- ✅ Batch processing
- ⚠️ Plan generation not yet supported

**Pricing:**
- $0.20-$0.90 per 1M tokens (varies by model)

**Best For:**
- Open source model preference
- Cost optimization
- Model experimentation

---

### Anyscale

Ray-powered inference with Llama models.

**Setup:**
```bash
export OPENAI_API_KEY=...  # Anyscale API key
```

**Usage:**
```bash
./kantra-ai remediate \
  --provider=anyscale \
  --model=meta-llama/Meta-Llama-3.1-70B-Instruct
```

**Features:**
- ✅ Ray-based scaling
- ✅ Llama 3.1 models
- ✅ Batch processing

**Best For:**
- Ray ecosystem users
- Scalable deployments

---

### Perplexity AI - Online Context

Can search online for migration guidance during fixes.

**Setup:**
```bash
export OPENAI_API_KEY=pplx-...
```

**Usage:**
```bash
./kantra-ai remediate \
  --provider=perplexity \
  --model=llama-3.1-sonar-large-128k-online
```

**Available Models:**
- `llama-3.1-sonar-large-128k-online` - With web search
- `llama-3.1-sonar-large-128k-chat` - Without search

**Features:**
- ✅ Can search web for migration docs
- ✅ Useful for newer frameworks
- ✅ Batch processing

**Best For:**
- Migrations with limited documentation
- Newer frameworks and libraries

---

### OpenRouter - 100+ Models

Access to 100+ models through one API with automatic fallbacks.

**Setup:**
```bash
export OPENAI_API_KEY=sk-or-...
```

**Usage:**
```bash
./kantra-ai remediate \
  --provider=openrouter \
  --model=meta-llama/llama-3.1-70b-instruct
```

**Features:**
- ✅ Access to 100+ models
- ✅ Automatic fallbacks
- ✅ Unified billing
- ✅ Batch processing

**Best For:**
- Model exploration and comparison
- Fallback redundancy

---

### LM Studio - Local GUI

Run models locally with a user-friendly GUI.

**Setup:**
```bash
# Start LM Studio and load a model from the GUI
```

**Usage:**
```bash
./kantra-ai remediate \
  --provider=lmstudio \
  --model=local-model
```

**Features:**
- ✅ User-friendly GUI
- ✅ Local execution
- ✅ Model management
- ✅ Free

**Best For:**
- Users preferring GUIs over CLI
- Local model experimentation

---

## Custom OpenAI-Compatible APIs

Use any OpenAI-compatible API by setting the base URL.

**Via Config File:**

```yaml
# .kantra-ai.yaml
provider:
  name: openai
  base-url: https://your-custom-api.com/v1
  model: your-model
```

**Via Environment:**

```bash
export OPENAI_API_KEY=your-key
export OPENAI_API_BASE=https://your-custom-api.com/v1

./kantra-ai remediate \
  --provider=openai \
  --model=your-model
```

**Examples:**
- vLLM deployments
- Text Generation Inference
- FastChat
- LocalAI
- Any custom OpenAI-compatible server

---

## Batch Processing Support

All providers support batch processing for 50-80% cost reduction and 70-90% faster execution.

**How it works:**
- Groups similar violations together
- Processes up to 10 incidents in a single API call
- Runs 4 batches in parallel by default

**Performance:**
```
Without batching: 100 violations × $0.10 = $10.00, ~50 minutes
With batching:     100 violations ÷ 10 × $0.10 = $1.00, ~8 minutes
Savings:          $9.00 (90% cost reduction), 42 minutes (84% faster)
```

**Configuration:**
```bash
./kantra-ai remediate \
  --batch-size=10 \
  --batch-parallelism=4
```

See [Batch Processing Design](../design/BATCH_PROCESSING_DESIGN.md) for details.

---

## Plan Generation Support

| Provider | Plan Generation | Status |
|----------|----------------|---------|
| Claude | ✅ Yes | Full support |
| OpenAI | ⚠️ Planned | Coming soon |
| Others | ⚠️ Future | Not yet supported |

Currently, only Claude supports AI-powered plan generation. Other providers can execute existing plans.

---

## Cost Estimation

Typical costs per violation (with batch processing):

| Provider | Simple Fix | Medium Fix | Complex Fix |
|----------|-----------|------------|-------------|
| Claude Sonnet 4 | $0.002-$0.01 | $0.01-$0.03 | $0.03-$0.10 |
| GPT-4 | $0.02-$0.10 | $0.10-$0.30 | $0.30-$1.00 |
| GPT-3.5 Turbo | $0.001-$0.01 | $0.01-$0.03 | $0.03-$0.10 |
| Groq | Free-$0.01 | Free-$0.03 | $0.01-$0.10 |
| Ollama | $0 | $0 | $0 |
| Together AI | $0.001-$0.01 | $0.01-$0.03 | $0.03-$0.10 |

Use `--dry-run` to get exact cost estimates before applying fixes.

---

## Choosing a Provider

### For Production Use

**Recommendation:** Claude Sonnet 4

**Reasons:**
- Highest quality code fixes
- Best at understanding context
- Batch processing for cost savings
- Plan generation support

**Command:**
```bash
./kantra-ai plan \
  --provider=claude \
  --model=claude-sonnet-4-20250514
```

---

### For Development/Testing

**Recommendation:** Groq or Ollama

**Groq** (if you have internet):
```bash
./kantra-ai remediate \
  --provider=groq \
  --model=llama-3.1-70b-versatile
```

**Ollama** (if you want local/offline):
```bash
ollama serve
./kantra-ai remediate \
  --provider=ollama \
  --model=codellama
```

---

### For Cost Optimization

**Recommendation:** Ollama (free) or Together AI (cheap cloud)

**Ollama** (zero cost):
```bash
ollama pull codellama
./kantra-ai remediate \
  --provider=ollama \
  --model=codellama
```

**Together AI** (low cost, cloud):
```bash
./kantra-ai remediate \
  --provider=together \
  --model=meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo
```

---

### For Privacy/Security

**Recommendation:** Ollama or LM Studio (100% local)

**Ollama:**
```bash
ollama serve
./kantra-ai remediate \
  --provider=ollama \
  --model=codellama
```

**Benefits:**
- No data leaves your machine
- Works offline
- No API keys needed
- Complete control

---

### For Model Exploration

**Recommendation:** OpenRouter

**Why:**
- Access 100+ models
- Single API key
- Easy model switching
- Automatic fallbacks

**Usage:**
```bash
# Try Llama 3.1
./kantra-ai remediate --provider=openrouter --model=meta-llama/llama-3.1-70b-instruct

# Try Mixtral
./kantra-ai remediate --provider=openrouter --model=mistralai/mixtral-8x7b-instruct

# Try Claude (via OpenRouter)
./kantra-ai remediate --provider=openrouter --model=anthropic/claude-3.5-sonnet
```

---

## Configuration Examples

### Claude (Default)

```yaml
# .kantra-ai.yaml
provider:
  name: claude
  model: claude-sonnet-4-20250514
```

### OpenAI with Custom Model

```yaml
provider:
  name: openai
  model: gpt-4-turbo
```

### Groq with Fast Model

```yaml
provider:
  name: groq
  model: llama-3.1-70b-versatile
```

### Ollama with CodeLlama

```yaml
provider:
  name: ollama
  model: codellama
  base-url: http://localhost:11434/v1  # Default Ollama URL
```

### Custom OpenAI-Compatible API

```yaml
provider:
  name: openai
  base-url: https://your-api.com/v1
  model: your-model
```

---

## Troubleshooting

### Rate Limit Errors

**Problem:** API rate limit exceeded

**Solutions:**
1. Reduce parallelism:
   ```bash
   --batch-parallelism=2
   ```
2. Use a different provider (e.g., Ollama has no rate limits)
3. Wait and retry with `--resume`

### Authentication Errors

**Problem:** Invalid API key

**Solutions:**
1. Check API key is correct:
   ```bash
   echo $ANTHROPIC_API_KEY
   echo $OPENAI_API_KEY
   ```
2. Verify key has necessary permissions
3. Try regenerating the key

### Low-Quality Fixes

**Problem:** AI generates poor fixes

**Solutions:**
1. Use a higher-quality model (Claude > GPT-4 > Llama)
2. Enable confidence filtering:
   ```bash
   --enable-confidence --on-low-confidence=skip
   ```
3. Use custom prompts optimized for your tech stack

### Slow Performance

**Problem:** Fixes taking too long

**Solutions:**
1. Use a faster provider (Groq, Ollama)
2. Increase parallelism:
   ```bash
   --batch-parallelism=8
   ```
3. Use batch processing (enabled by default)

---

## See Also

- [Usage Examples](./USAGE_EXAMPLES.md) - Provider-specific examples
- [CLI Reference](./CLI_REFERENCE.md) - Complete flag reference
- [Quick Start](./QUICKSTART.md) - Getting started guide
