package pkg

import (
	"context"
	"crypto/x509"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"net/url"
	"time"
)

// CreateAndObserveGRPCConn creates a new gRPC connection and observes its conn status.
func CreateAndObserveGRPCConn(ctx context.Context, ch chan error, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	port := u.Port()
	if port == "" {
		port = "443"
	}

	if u.Scheme == "http" {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		pool, _ := x509.SystemCertPool()
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(pool, "")))
	}

	hostname := u.Hostname()
	if hostname == "" {
		return nil, errors.New("please provide URL format endpoint e.g. http(s)://<endpoint>:<port>")
	}

	address := hostname + ":" + port

	opts = append(opts, grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(100*1024*1024),
		grpc.MaxCallSendMsgSize(100*1024*1024),
	),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:    10 * time.Second,
			Timeout: 5 * time.Second,
		}),
	)

	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		return nil, err
	}

	go func() {
		var retries int
		for {
			select {
			case <-ctx.Done():
				if err = conn.Close(); err != nil {
					ch <- err
				}
				return
			default:
				state := conn.GetState()
				if state == connectivity.Ready {
					retries = 0
					time.Sleep(1 * time.Second)
					continue
				}

				if state == connectivity.TransientFailure || state == connectivity.Connecting || state == connectivity.Idle {
					if retries < 5 {
						time.Sleep(time.Duration(retries) * time.Second)
						conn.ResetConnectBackoff()
						retries++
					} else {
						conn.Close()
						conn, err = grpc.NewClient(target, opts...)
						if err != nil {
							ch <- err
						}
						retries = 0
					}
				} else if state == connectivity.Shutdown {
					conn, err = grpc.NewClient(target, opts...)
					if err != nil {
						ch <- err
					}
					retries = 0
				}

				if !conn.WaitForStateChange(ctx, state) {
					continue
				}
			}
		}
	}()

	return conn, nil
}
