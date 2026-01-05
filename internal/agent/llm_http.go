package agent

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

func readHTTPResponse(resp *http.Response, provider string) ([]byte, error) {
	if resp.Body == nil {
		return nil, fmt.Errorf("%s response empty", provider)
	}
	data, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if readErr != nil {
		err := fmt.Errorf("%s response read: %w", provider, readErr)
		if closeErr != nil {
			return nil, errors.Join(err, fmt.Errorf("%s response close: %w", provider, closeErr))
		}
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("%s error: %s", provider, string(data))
		if closeErr != nil {
			return nil, errors.Join(err, fmt.Errorf("%s response close: %w", provider, closeErr))
		}
		return nil, err
	}
	if closeErr != nil {
		return nil, fmt.Errorf("%s response close: %w", provider, closeErr)
	}
	return data, nil
}
