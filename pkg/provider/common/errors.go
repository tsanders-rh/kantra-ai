// Package common provides shared utilities for AI provider implementations.
package common

import (
	"fmt"
	"strings"
)

// ProviderErrorContext contains provider-specific information for error enhancement.
type ProviderErrorContext struct {
	ProviderName   string // e.g., "Claude", "OpenAI"
	APIKeysURL     string // URL to manage API keys
	StatusPageURL  string // URL to check API status
	BillingURL     string // URL for billing/usage (optional)
	AlternateProvider string // Alternative provider name for suggestions
}

// EnhanceAPIError adds helpful context to AI provider API errors.
// It detects common error patterns and provides actionable troubleshooting steps.
func EnhanceAPIError(err error, ctx ProviderErrorContext) error {
	errMsg := err.Error()

	// 401: Authentication errors
	if contains(errMsg, "401") || contains(errMsg, "unauthorized") || contains(errMsg, "invalid api key") {
		envVar := strings.ToUpper(ctx.ProviderName) + "_API_KEY"
		return fmt.Errorf("%s API authentication failed: %w\n\n"+
			"Possible causes:\n"+
			"  - Invalid or expired API key\n"+
			"  - API key revoked or deleted\n\n"+
			"To fix:\n"+
			"  1. Verify your API key at: %s\n"+
			"  2. Ensure %s is set correctly\n"+
			"  3. Try generating a new API key", ctx.ProviderName, err, ctx.APIKeysURL, envVar)
	}

	// 429: Rate limit errors
	if contains(errMsg, "429") || contains(errMsg, "rate limit") {
		msg := fmt.Sprintf("%s API rate limit exceeded: %%w\n\n"+
			"You've made too many requests in a short period.\n\n"+
			"To fix:\n"+
			"  1. Wait a few minutes and try again\n"+
			"  2. Reduce the number of violations being fixed\n"+
			"  3. Upgrade your %s API plan for higher limits", ctx.ProviderName, ctx.ProviderName)

		return fmt.Errorf(msg, err)
	}

	// Quota exceeded (OpenAI-specific pattern, but safe for all providers)
	if contains(errMsg, "insufficient_quota") || contains(errMsg, "quota") {
		msg := fmt.Sprintf("%s API quota exceeded: %%w\n\n"+
			"You've reached your account spending limit.\n\n"+
			"To fix:\n"+
			"  1. Check your usage and add credits if needed\n"+
			"  2. Upgrade your plan for higher limits", ctx.ProviderName)

		if ctx.BillingURL != "" {
			msg = fmt.Sprintf("%s API quota exceeded: %%w\n\n"+
				"You've reached your account spending limit.\n\n"+
				"To fix:\n"+
				"  1. Add credits: %s\n"+
				"  2. Upgrade your plan for higher limits", ctx.ProviderName, ctx.BillingURL)
		}

		if ctx.AlternateProvider != "" {
			msg += fmt.Sprintf("\n  3. Or use --provider=%s instead", strings.ToLower(ctx.AlternateProvider))
		}

		return fmt.Errorf(msg, err)
	}

	// Timeout errors
	if contains(errMsg, "timeout") || contains(errMsg, "deadline exceeded") {
		return fmt.Errorf("%s API request timed out: %w\n\n"+
			"The request took too long to complete.\n\n"+
			"To fix:\n"+
			"  1. Check your internet connection\n"+
			"  2. Try again - this is often a temporary issue\n"+
			"  3. If persistent, reduce file size or complexity", ctx.ProviderName, err)
	}

	// Network/connection errors
	if contains(errMsg, "connection") || contains(errMsg, "network") || contains(errMsg, "dial") {
		return fmt.Errorf("network error connecting to %s API: %w\n\n"+
			"Unable to reach the API servers.\n\n"+
			"To fix:\n"+
			"  1. Check your internet connection\n"+
			"  2. Check if your firewall/proxy is blocking the connection\n"+
			"  3. Try again in a few moments", ctx.ProviderName, err)
	}

	// Server errors (500, 502, 503)
	if contains(errMsg, "500") || contains(errMsg, "502") || contains(errMsg, "503") {
		msg := fmt.Sprintf("%s API server error: %%w\n\n"+
			"The API is experiencing issues.\n\n"+
			"To fix:\n"+
			"  1. Wait a few minutes and try again", ctx.ProviderName)

		if ctx.StatusPageURL != "" {
			msg += fmt.Sprintf("\n  2. Check status page: %s", ctx.StatusPageURL)
		}

		if ctx.AlternateProvider != "" {
			msg += fmt.Sprintf("\n  3. If urgent, try --provider=%s instead", strings.ToLower(ctx.AlternateProvider))
		}

		return fmt.Errorf(msg, err)
	}

	// Generic API error
	return fmt.Errorf("%s API error: %w\n\n"+
		"An unexpected error occurred.\n\n"+
		"To fix:\n"+
		"  1. Check the error message above for details\n"+
		"  2. Verify your API configuration\n"+
		"  3. Try again or contact support", ctx.ProviderName, err)
}

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
