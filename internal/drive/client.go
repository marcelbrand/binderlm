package drive

import (
	"context"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// NewService creates and returns a new Google Drive v3 Service.
// If no options are passed, it automatically resolves credentials using ResolveClientOption.
func NewService(ctx context.Context, opts ...option.ClientOption) (*drive.Service, error) {
	if len(opts) == 0 {
		opt, err := ResolveClientOption(ctx)
		if err != nil {
			return nil, err
		}
		opts = append(opts, opt)
	}

	srv, err := drive.NewService(ctx, opts...)
	if err != nil {
		return nil, &AuthError{
			Message: "failed to initialize Google Drive client",
			Err:     err,
		}
	}

	return srv, nil
}
