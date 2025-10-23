/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/tap"
)

// LoggingUnaryInterceptor logs all RPC calls with connection details.
func LoggingUnaryInterceptor(verbose bool) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Get peer information
		p, ok := peer.FromContext(ctx)
		var peerAddr string
		if ok {
			peerAddr = p.Addr.String()
			if verbose {
				LogTLSConnectionInfo(p)
			}
		} else {
			peerAddr = "unknown"
		}

		log.Printf("[RPC] --> %s from %s", info.FullMethod, peerAddr)

		// Call the handler
		resp, err := handler(ctx, req)

		duration := time.Since(start)
		if err != nil {
			log.Printf("[RPC] <-- %s failed in %v: %v", info.FullMethod, duration, err)
		} else {
			log.Printf("[RPC] <-- %s succeeded in %v", info.FullMethod, duration)
		}

		return resp, err
	}
}

// ConnectionTapHandler logs every connection attempt with detailed information.
func ConnectionTapHandler(ctx context.Context, info *tap.Info) (context.Context, error) {
	log.Printf("[CONNECTION] New connection attempt: %+v", info)
	return ctx, nil
}

// LogTLSConnectionInfo logs detailed TLS handshake information.
func LogTLSConnectionInfo(p *peer.Peer) {
	if p == nil {
		log.Printf("[TLS] No peer information available")
		return
	}

	log.Printf("[TLS] Connection from: %s", p.Addr.String())

	authInfo := p.AuthInfo
	if authInfo == nil {
		log.Printf("[TLS] WARNING: No auth info - connection may not be using TLS")
		return
	}

	tlsInfo, ok := authInfo.(credentials.TLSInfo)
	if !ok {
		log.Printf("[TLS] WARNING: Auth info is not TLS type: %T", authInfo)
		return
	}

	state := tlsInfo.State
	log.Printf("[TLS] Handshake complete: version=0x%04x (%s), cipher=0x%04x, resumed=%v",
		state.Version, TLSVersionName(state.Version), state.CipherSuite, state.DidResume)

	if state.ServerName != "" {
		log.Printf("[TLS] SNI server name: %s", state.ServerName)
	}

	// Log peer certificates
	if len(state.PeerCertificates) > 0 {
		log.Printf("[TLS] Peer presented %d certificate(s)", len(state.PeerCertificates))
		for i, cert := range state.PeerCertificates {
			log.Printf("[TLS]   Cert %d: Subject=%s, Issuer=%s, NotBefore=%s, NotAfter=%s",
				i, cert.Subject, cert.Issuer, cert.NotBefore, cert.NotAfter)
			if len(cert.DNSNames) > 0 {
				log.Printf("[TLS]   Cert %d: DNS names=%v", i, cert.DNSNames)
			}
		}
	} else {
		log.Printf("[TLS] WARNING: No peer certificates - mTLS may not be working")
	}

	// Log verified chains
	if len(state.VerifiedChains) > 0 {
		log.Printf("[TLS] Successfully verified %d certificate chain(s)", len(state.VerifiedChains))
	} else {
		log.Printf("[TLS] WARNING: No verified certificate chains")
	}
}
