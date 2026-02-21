package sqlite

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

func parseDSN(dsn string) (string, error) {
	if !strings.HasPrefix(dsn, "sqlite://") {
		return "", fmt.Errorf("invalid sqlite DSN scheme, expected sqlite://")
	}

	rest := strings.TrimPrefix(dsn, "sqlite://")

	if rest == ":memory:" {
		return ":memory:", nil
	}

	if strings.HasPrefix(rest, "/") {
		return rest, nil
	}

	if strings.HasPrefix(rest, "./") {
		return rest, nil
	}

	if strings.Contains(rest, "?") {
		parts := strings.SplitN(rest, "?", 2)
		path := parts[0]
		query := parts[1]

		unescaped, err := url.PathUnescape(path)
		if err != nil {
			return "", fmt.Errorf("unescaping path: %w", err)
		}
		path = unescaped

		if !filepath.IsAbs(path) && !strings.HasPrefix(path, "./") {
			path = "./" + path
		}
		return path + "?" + query, nil
	}

	unescaped, err := url.PathUnescape(rest)
	if err != nil {
		return "", fmt.Errorf("unescaping path: %w", err)
	}
	rest = unescaped

	if !filepath.IsAbs(rest) {
		rest = "./" + rest
	}

	return rest, nil
}
