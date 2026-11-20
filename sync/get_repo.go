package sync

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/jcalabro/atmos"
	"github.com/jcalabro/atmos/api/comatproto"
	"github.com/jcalabro/atmos/xrpc"
)

// GetRepoStream downloads a repo as a streaming CAR. The caller MUST close
// the returned ReadCloser. If since is non-empty, requests a diff from that rev.
func (c *Client) GetRepoStream(ctx context.Context, did atmos.DID, since string) (io.ReadCloser, error) {
	body, _, err := c.GetRepoStreamHost(ctx, did, since)
	return body, err
}

// GetRepoStreamHost is [GetRepoStream] that also reports the host that
// served the CAR — the final URL's host after any relay 302 redirect to
// the account's PDS. See [xrpc.Client.QueryStreamHost] for the host
// semantics (including the [*xrpc.Error] carrying Host on the error
// path). The caller MUST close the returned ReadCloser.
func (c *Client) GetRepoStreamHost(ctx context.Context, did atmos.DID, since string) (io.ReadCloser, string, error) {
	params := map[string]any{
		"did": string(did),
	}
	if since != "" {
		params["since"] = since
	}
	return c.opts.Client.QueryStreamHost(ctx, getRepoNSID, params)
}

const getRepoNSID = "com.atproto.sync.getRepo"

// GetRepoRateLimitedUntil reports when this client will next send getRepo
// to its host, or the zero time if getRepo is not parked by an exhausted
// quota. See [xrpc.Client.RateLimitedUntil].
func (c *Client) GetRepoRateLimitedUntil() time.Time {
	return c.opts.Client.RateLimitedUntil(getRepoNSID)
}

// WaitGetRepoRateLimit blocks while getRepo is parked for this client's host
// and returns how long it waited. See [xrpc.Client.WaitRateLimit].
func (c *Client) WaitGetRepoRateLimit(ctx context.Context) (time.Duration, error) {
	return c.opts.Client.WaitRateLimit(ctx, getRepoNSID)
}

// HasRecord asks the host whether it serves the record, with
// com.atproto.repo.getRecord. Only the host's own RecordNotFound answer is
// false; any other failure is an error.
//
// ALLUVIUM PATCH: not in upstream atmos v0.8.0.
func (c *Client) HasRecord(ctx context.Context, did atmos.DID, collection, rkey string) (bool, error) {
	_, err := comatproto.RepoGetRecord(ctx, c.opts.Client, "", collection, string(did), rkey)
	var xerr *xrpc.Error
	if errors.As(err, &xerr) && xerr.Name == comatproto.ErrRepoGetRecord_RecordNotFound {
		return false, nil
	}
	return err == nil, err
}
