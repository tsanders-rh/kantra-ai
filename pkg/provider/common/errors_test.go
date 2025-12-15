package common

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnhanceAPIError_AuthenticationErrors(t *testing.T) {
	ctx := ProviderErrorContext{
		ProviderName: "Claude",
		APIKeysURL:   "https://console.anthropic.com/settings/keys",
	}

	t.Run("401 error code", func(t *testing.T) {
		err := errors.New("HTTP 401 Unauthorized")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "Claude API authentication failed")
		assert.Contains(t, enhanced.Error(), "Invalid or expired API key")
		assert.Contains(t, enhanced.Error(), "CLAUDE_API_KEY")
		assert.Contains(t, enhanced.Error(), ctx.APIKeysURL)
	})

	t.Run("unauthorized keyword", func(t *testing.T) {
		err := errors.New("Request unauthorized")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "authentication failed")
		assert.Contains(t, enhanced.Error(), "CLAUDE_API_KEY")
	})

	t.Run("invalid api key message", func(t *testing.T) {
		err := errors.New("invalid api key provided")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "authentication failed")
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		err := errors.New("UNAUTHORIZED - API KEY INVALID")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "authentication failed")
	})

	t.Run("environment variable naming", func(t *testing.T) {
		openaiCtx := ProviderErrorContext{
			ProviderName: "OpenAI",
			APIKeysURL:   "https://platform.openai.com/api-keys",
		}

		err := errors.New("401 Unauthorized")
		enhanced := EnhanceAPIError(err, openaiCtx)

		assert.Contains(t, enhanced.Error(), "OPENAI_API_KEY")
		assert.NotContains(t, enhanced.Error(), "CLAUDE_API_KEY")
	})
}

func TestEnhanceAPIError_RateLimitErrors(t *testing.T) {
	ctx := ProviderErrorContext{
		ProviderName: "Claude",
	}

	t.Run("429 error code", func(t *testing.T) {
		err := errors.New("HTTP 429 Too Many Requests")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "rate limit exceeded")
		assert.Contains(t, enhanced.Error(), "too many requests")
		assert.Contains(t, enhanced.Error(), "Wait a few minutes")
		assert.Contains(t, enhanced.Error(), "Reduce the number of violations")
		assert.Contains(t, enhanced.Error(), "Upgrade your Claude API plan")
	})

	t.Run("rate limit keyword", func(t *testing.T) {
		err := errors.New("rate limit exceeded for organization")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "rate limit exceeded")
	})

	t.Run("case insensitive", func(t *testing.T) {
		err := errors.New("RATE LIMIT ERROR")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "rate limit exceeded")
	})
}

func TestEnhanceAPIError_QuotaErrors(t *testing.T) {
	t.Run("quota without billing URL", func(t *testing.T) {
		ctx := ProviderErrorContext{
			ProviderName: "Claude",
		}

		err := errors.New("insufficient_quota: You exceeded your current quota")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "quota exceeded")
		assert.Contains(t, enhanced.Error(), "spending limit")
		assert.Contains(t, enhanced.Error(), "Check your usage and add credits")
		assert.NotContains(t, enhanced.Error(), "http") // No URL included
	})

	t.Run("quota with billing URL", func(t *testing.T) {
		ctx := ProviderErrorContext{
			ProviderName: "OpenAI",
			BillingURL:   "https://platform.openai.com/account/billing",
		}

		err := errors.New("quota exceeded")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "quota exceeded")
		assert.Contains(t, enhanced.Error(), ctx.BillingURL)
		assert.Contains(t, enhanced.Error(), "Add credits")
	})

	t.Run("quota with alternate provider", func(t *testing.T) {
		ctx := ProviderErrorContext{
			ProviderName:      "OpenAI",
			AlternateProvider: "Claude",
		}

		err := errors.New("insufficient_quota")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "quota exceeded")
		assert.Contains(t, enhanced.Error(), "--provider=claude")
	})

	t.Run("quota with billing URL and alternate provider", func(t *testing.T) {
		ctx := ProviderErrorContext{
			ProviderName:      "Claude",
			BillingURL:        "https://console.anthropic.com/settings/billing",
			AlternateProvider: "OpenAI",
		}

		err := errors.New("quota limit reached")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "quota exceeded")
		assert.Contains(t, enhanced.Error(), ctx.BillingURL)
		assert.Contains(t, enhanced.Error(), "--provider=openai")
	})
}

func TestEnhanceAPIError_TimeoutErrors(t *testing.T) {
	ctx := ProviderErrorContext{
		ProviderName: "Claude",
	}

	t.Run("timeout keyword", func(t *testing.T) {
		err := errors.New("request timeout after 60s")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "request timed out")
		assert.Contains(t, enhanced.Error(), "took too long")
		assert.Contains(t, enhanced.Error(), "Check your internet connection")
		assert.Contains(t, enhanced.Error(), "temporary issue")
	})

	t.Run("deadline exceeded", func(t *testing.T) {
		err := errors.New("context deadline exceeded")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "request timed out")
	})

	t.Run("case insensitive", func(t *testing.T) {
		err := errors.New("TIMEOUT ERROR")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "request timed out")
	})
}

func TestEnhanceAPIError_NetworkErrors(t *testing.T) {
	ctx := ProviderErrorContext{
		ProviderName: "Claude",
	}

	t.Run("connection error", func(t *testing.T) {
		err := errors.New("connection refused")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "network error")
		assert.Contains(t, enhanced.Error(), "Unable to reach the API servers")
		assert.Contains(t, enhanced.Error(), "internet connection")
		assert.Contains(t, enhanced.Error(), "firewall/proxy")
	})

	t.Run("network error", func(t *testing.T) {
		err := errors.New("network unreachable")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "network error")
	})

	t.Run("dial error", func(t *testing.T) {
		err := errors.New("dial tcp: connection timed out")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "network error")
	})
}

func TestEnhanceAPIError_ServerErrors(t *testing.T) {
	t.Run("500 error without status page", func(t *testing.T) {
		ctx := ProviderErrorContext{
			ProviderName: "Claude",
		}

		err := errors.New("HTTP 500 Internal Server Error")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "server error")
		assert.Contains(t, enhanced.Error(), "API is experiencing issues")
		assert.Contains(t, enhanced.Error(), "Wait a few minutes")
		assert.NotContains(t, enhanced.Error(), "Check status page") // No status page URL
	})

	t.Run("502 error with status page", func(t *testing.T) {
		ctx := ProviderErrorContext{
			ProviderName:  "OpenAI",
			StatusPageURL: "https://status.openai.com",
		}

		err := errors.New("HTTP 502 Bad Gateway")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "server error")
		assert.Contains(t, enhanced.Error(), "Check status page")
		assert.Contains(t, enhanced.Error(), ctx.StatusPageURL)
	})

	t.Run("503 error with alternate provider", func(t *testing.T) {
		ctx := ProviderErrorContext{
			ProviderName:      "Claude",
			AlternateProvider: "OpenAI",
		}

		err := errors.New("HTTP 503 Service Unavailable")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "server error")
		assert.Contains(t, enhanced.Error(), "--provider=openai")
		assert.Contains(t, enhanced.Error(), "If urgent")
	})

	t.Run("500 error with status page and alternate provider", func(t *testing.T) {
		ctx := ProviderErrorContext{
			ProviderName:      "OpenAI",
			StatusPageURL:     "https://status.openai.com",
			AlternateProvider: "Claude",
		}

		err := errors.New("500 server error")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "server error")
		assert.Contains(t, enhanced.Error(), ctx.StatusPageURL)
		assert.Contains(t, enhanced.Error(), "--provider=claude")
	})
}

func TestEnhanceAPIError_GenericError(t *testing.T) {
	ctx := ProviderErrorContext{
		ProviderName: "Claude",
	}

	t.Run("unknown error", func(t *testing.T) {
		err := errors.New("something went wrong")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "Claude API error")
		assert.Contains(t, enhanced.Error(), "unexpected error")
		assert.Contains(t, enhanced.Error(), "Check the error message")
		assert.Contains(t, enhanced.Error(), "Verify your API configuration")
	})

	t.Run("preserves original error", func(t *testing.T) {
		err := errors.New("custom error message")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "custom error message")
	})
}

func TestEnhanceAPIError_ErrorWrapping(t *testing.T) {
	t.Run("wraps original error for authentication", func(t *testing.T) {
		ctx := ProviderErrorContext{
			ProviderName: "Claude",
			APIKeysURL:   "https://example.com",
		}

		originalErr := errors.New("original error: 401")
		enhanced := EnhanceAPIError(originalErr, ctx)

		// Should be able to unwrap to get original error
		assert.ErrorIs(t, enhanced, originalErr)
	})

	t.Run("wraps original error for rate limit", func(t *testing.T) {
		ctx := ProviderErrorContext{ProviderName: "Claude"}
		originalErr := errors.New("original error: 429")
		enhanced := EnhanceAPIError(originalErr, ctx)

		assert.ErrorIs(t, enhanced, originalErr)
	})

	t.Run("wraps original error for quota", func(t *testing.T) {
		ctx := ProviderErrorContext{ProviderName: "Claude"}
		originalErr := errors.New("insufficient_quota")
		enhanced := EnhanceAPIError(originalErr, ctx)

		assert.ErrorIs(t, enhanced, originalErr)
	})
}

func TestEnhanceAPIError_MultipleProviders(t *testing.T) {
	t.Run("Claude provider context", func(t *testing.T) {
		ctx := ProviderErrorContext{
			ProviderName:      "Claude",
			APIKeysURL:        "https://console.anthropic.com/settings/keys",
			StatusPageURL:     "https://status.anthropic.com",
			BillingURL:        "https://console.anthropic.com/settings/billing",
			AlternateProvider: "OpenAI",
		}

		err := errors.New("401 Unauthorized")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "Claude API")
		assert.Contains(t, enhanced.Error(), "CLAUDE_API_KEY")
		assert.Contains(t, enhanced.Error(), ctx.APIKeysURL)
	})

	t.Run("OpenAI provider context", func(t *testing.T) {
		ctx := ProviderErrorContext{
			ProviderName:      "OpenAI",
			APIKeysURL:        "https://platform.openai.com/api-keys",
			StatusPageURL:     "https://status.openai.com",
			BillingURL:        "https://platform.openai.com/account/billing",
			AlternateProvider: "Claude",
		}

		err := errors.New("401 error")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "OpenAI API")
		assert.Contains(t, enhanced.Error(), "OPENAI_API_KEY")
		assert.Contains(t, enhanced.Error(), ctx.APIKeysURL)
	})

	t.Run("custom provider context", func(t *testing.T) {
		ctx := ProviderErrorContext{
			ProviderName: "CustomAI",
			APIKeysURL:   "https://customai.com/keys",
		}

		err := errors.New("unauthorized")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "CustomAI API")
		assert.Contains(t, enhanced.Error(), "CUSTOMAI_API_KEY")
	})
}

func TestEnhanceAPIError_PriorityOfErrorTypes(t *testing.T) {
	ctx := ProviderErrorContext{
		ProviderName: "Claude",
	}

	t.Run("401 takes priority over generic keywords", func(t *testing.T) {
		err := errors.New("network error: 401 unauthorized")
		enhanced := EnhanceAPIError(err, ctx)

		// Should match authentication error, not network error
		assert.Contains(t, enhanced.Error(), "authentication failed")
		assert.NotContains(t, enhanced.Error(), "network error connecting")
	})

	t.Run("429 takes priority over generic keywords", func(t *testing.T) {
		err := errors.New("timeout: 429 rate limit exceeded")
		enhanced := EnhanceAPIError(err, ctx)

		// Should match rate limit, not timeout
		assert.Contains(t, enhanced.Error(), "rate limit exceeded")
		assert.NotContains(t, enhanced.Error(), "request timed out")
	})

	t.Run("quota takes priority over timeout", func(t *testing.T) {
		err := errors.New("timeout due to insufficient_quota")
		enhanced := EnhanceAPIError(err, ctx)

		// Should match quota, not timeout
		assert.Contains(t, enhanced.Error(), "quota exceeded")
		assert.NotContains(t, enhanced.Error(), "request timed out")
	})
}

func TestContains(t *testing.T) {
	t.Run("case insensitive match", func(t *testing.T) {
		assert.True(t, contains("Hello World", "WORLD"))
		assert.True(t, contains("HELLO WORLD", "world"))
		assert.True(t, contains("HeLLo WoRLd", "llo wo"))
	})

	t.Run("exact match", func(t *testing.T) {
		assert.True(t, contains("hello", "hello"))
		assert.True(t, contains("HELLO", "HELLO"))
	})

	t.Run("partial match", func(t *testing.T) {
		assert.True(t, contains("unauthorized", "auth"))
		assert.True(t, contains("rate limit exceeded", "limit"))
	})

	t.Run("no match", func(t *testing.T) {
		assert.False(t, contains("hello", "world"))
		assert.False(t, contains("unauthorized", "quota"))
	})

	t.Run("empty substring always matches", func(t *testing.T) {
		assert.True(t, contains("hello", ""))
		assert.True(t, contains("", ""))
	})

	t.Run("empty string only matches empty substring", func(t *testing.T) {
		assert.False(t, contains("", "hello"))
	})
}

func TestEnhanceAPIError_RealWorldScenarios(t *testing.T) {
	t.Run("Claude API key error scenario", func(t *testing.T) {
		ctx := ProviderErrorContext{
			ProviderName:      "Claude",
			APIKeysURL:        "https://console.anthropic.com/settings/keys",
			StatusPageURL:     "https://status.anthropic.com",
			AlternateProvider: "OpenAI",
		}

		err := errors.New("anthropic: error, status code: 401, message: {\"error\":{\"type\":\"authentication_error\"}}")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "authentication failed")
		assert.Contains(t, enhanced.Error(), "CLAUDE_API_KEY")
		assert.Contains(t, enhanced.Error(), "https://console.anthropic.com/settings/keys")
	})

	t.Run("OpenAI rate limit scenario", func(t *testing.T) {
		ctx := ProviderErrorContext{
			ProviderName:      "OpenAI",
			APIKeysURL:        "https://platform.openai.com/api-keys",
			StatusPageURL:     "https://status.openai.com",
			AlternateProvider: "Claude",
		}

		err := errors.New("openai: error, status code: 429, message: Rate limit exceeded")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "rate limit exceeded")
		assert.Contains(t, enhanced.Error(), "Wait a few minutes")
	})

	t.Run("OpenAI quota exceeded scenario", func(t *testing.T) {
		ctx := ProviderErrorContext{
			ProviderName:      "OpenAI",
			BillingURL:        "https://platform.openai.com/account/billing",
			AlternateProvider: "Claude",
		}

		err := errors.New("insufficient_quota: You exceeded your current quota, please check your plan and billing details")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "quota exceeded")
		assert.Contains(t, enhanced.Error(), "https://platform.openai.com/account/billing")
		assert.Contains(t, enhanced.Error(), "--provider=claude")
	})

	t.Run("network connection failure scenario", func(t *testing.T) {
		ctx := ProviderErrorContext{
			ProviderName: "Claude",
		}

		err := errors.New("dial tcp: lookup api.anthropic.com: no such host")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "network error")
		assert.Contains(t, enhanced.Error(), "Unable to reach")
		assert.Contains(t, enhanced.Error(), "internet connection")
	})

	t.Run("API server outage scenario", func(t *testing.T) {
		ctx := ProviderErrorContext{
			ProviderName:      "Claude",
			StatusPageURL:     "https://status.anthropic.com",
			AlternateProvider: "OpenAI",
		}

		err := errors.New("HTTP 503 Service Unavailable: The server is temporarily unable to service your request")
		enhanced := EnhanceAPIError(err, ctx)

		assert.Contains(t, enhanced.Error(), "server error")
		assert.Contains(t, enhanced.Error(), "experiencing issues")
		assert.Contains(t, enhanced.Error(), "https://status.anthropic.com")
		assert.Contains(t, enhanced.Error(), "--provider=openai")
	})
}

func TestEnhanceAPIError_MessageFormatting(t *testing.T) {
	ctx := ProviderErrorContext{
		ProviderName: "TestProvider",
	}

	t.Run("includes provider name consistently", func(t *testing.T) {
		testCases := []struct {
			name  string
			input string
		}{
			{"auth error", "401 unauthorized"},
			{"rate limit", "429 rate limit"},
			{"quota", "insufficient_quota"},
			{"timeout", "timeout"},
			{"network", "connection refused"},
			{"server", "500 server error"},
			{"generic", "unknown error"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := errors.New(tc.input)
				enhanced := EnhanceAPIError(err, ctx)

				// All errors should mention the provider name
				assert.Contains(t, enhanced.Error(), "TestProvider")
			})
		}
	})

	t.Run("includes actionable steps", func(t *testing.T) {
		err := errors.New("401 unauthorized")
		enhanced := EnhanceAPIError(err, ctx)

		// Should have numbered steps
		assert.Contains(t, enhanced.Error(), "To fix:")
		// Check for presence of numbered items (at least one)
		hasNumberedSteps := strings.Contains(enhanced.Error(), "1.") ||
			strings.Contains(enhanced.Error(), "2.") ||
			strings.Contains(enhanced.Error(), "3.")
		assert.True(t, hasNumberedSteps, "Enhanced error should contain numbered steps")
	})

	t.Run("preserves line breaks for readability", func(t *testing.T) {
		err := errors.New("429 rate limit")
		enhanced := EnhanceAPIError(err, ctx)

		// Should have multiple newlines for formatting
		newlineCount := strings.Count(enhanced.Error(), "\n")
		assert.Greater(t, newlineCount, 3, "Enhanced error should have multiple line breaks for readability")
	})
}
