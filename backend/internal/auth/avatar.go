package auth

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
)

func firstNonEmptyClaim(claims map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := claims[key].(string)
		if ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func avatarFromClaims(claims map[string]any) string {
	for _, key := range []string{"picture", "avatar", "avatar_url"} {
		if avatar := normalizeAvatar(claims[key]); avatar != "" {
			return avatar
		}
	}
	return ""
}

func normalizeAvatar(raw any) string {
	if attributes, ok := raw.(map[string]any); ok {
		for _, key := range []string{"url", "src", "data", "base64"} {
			if avatar := normalizeAvatar(attributes[key]); avatar != "" {
				return avatar
			}
		}
		return ""
	}

	value, ok := raw.(string)
	if !ok || len(value) > 1400000 {
		return ""
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	if address, err := url.Parse(value); err == nil &&
		(address.Scheme == "http" || address.Scheme == "https") && address.Host != "" {
		return value
	}

	if strings.HasPrefix(value, "data:image/") {
		return normalizeDataImage(value)
	}

	bytes, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return ""
	}

	mimeType := http.DetectContentType(bytes)
	if !allowedAvatarType(mimeType) {
		return ""
	}

	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(bytes)
}

func normalizeDataImage(value string) string {
	parts := strings.SplitN(value, ",", 2)
	if len(parts) != 2 || !strings.HasSuffix(strings.ToLower(parts[0]), ";base64") {
		return ""
	}

	mimeType := strings.ToLower(strings.TrimPrefix(strings.Split(parts[0], ";")[0], "data:"))
	if !allowedAvatarType(mimeType) {
		return ""
	}

	if _, err := base64.StdEncoding.DecodeString(parts[1]); err != nil {
		return ""
	}
	return value
}

func allowedAvatarType(mimeType string) bool {
	switch mimeType {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}
