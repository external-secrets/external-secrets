package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	pb "github.com/external-secrets/external-secrets/pkg/plugin/grpc"
	"google.golang.org/grpc"
)

// Client is a small wrapper to map ESO SecretsClient to gRPC calls
type Client struct {
	store     esv1beta1.GenericStore
	namespace string

	conn       *grpc.ClientConn
	grpcClient pb.SecretsClientClient
}

func (s *Client) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	storeBytes, err := json.Marshal(s.store)
	if err != nil {
		return nil, err
	}
	res, err := s.grpcClient.GetSecret(ctx, &pb.GetSecretRequest{
		Store:     storeBytes,
		Namespace: s.namespace,
		RemoteRef: &pb.RemoteRef{
			Key:                ref.Key,
			Property:           ref.Property,
			Version:            ref.Version,
			MetadataPolicy:     string(ref.MetadataPolicy),
			ConversionStrategy: string(ref.ConversionStrategy),
			DecodingStrategy:   string(ref.DecodingStrategy),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("unable to rpc: %w", err)
	}
	log.Printf("rpc secret=%s, err=%s", string(res.Secret), res.Error)
	return res.Secret, nil
}

func (s *Client) PushSecret(ctx context.Context, value []byte, remoteRef esv1beta1.PushRemoteRef) error {
	storeBytes, err := json.Marshal(s.store)
	if err != nil {
		return err
	}
	res, err := s.grpcClient.PushSecret(ctx, &pb.PushSecretRequest{
		Store:     storeBytes,
		Namespace: s.namespace,
		Secret:    value,
		RemoteRef: &pb.PushRemoteRef{
			RemoteKey: remoteRef.GetRemoteKey(),
			Property:  remoteRef.GetProperty(),
		},
	})
	if err != nil {
		return fmt.Errorf("unable to rpc: %w", err)
	}
	if res.Error != "" {
		return fmt.Errorf("rpc error: %s", res.Error)
	}
	return nil
}

func (s *Client) DeleteSecret(ctx context.Context, remoteRef esv1beta1.PushRemoteRef) error {
	storeBytes, err := json.Marshal(s.store)
	if err != nil {
		return err
	}
	res, err := s.grpcClient.DeleteSecret(ctx, &pb.DeleteSecretRequest{
		Store:     storeBytes,
		Namespace: s.namespace,
		RemoteRef: &pb.PushRemoteRef{
			RemoteKey: remoteRef.GetRemoteKey(),
			Property:  remoteRef.GetProperty(),
		},
	})
	if err != nil {
		return fmt.Errorf("unable to rpc: %w", err)
	}
	if res.Error != "" {
		return fmt.Errorf("rpc error: %s", res.Error)
	}
	return nil
}

func (s *Client) Validate() (esv1beta1.ValidationResult, error) {
	return esv1beta1.ValidationResultUnknown, nil
}

func (s *Client) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	storeBytes, err := json.Marshal(s.store)
	if err != nil {
		return nil, err
	}
	res, err := s.grpcClient.GetSecretMap(ctx, &pb.GetSecretMapRequest{
		Store:     storeBytes,
		Namespace: s.namespace,
		RemoteRef: &pb.RemoteRef{
			Key:                ref.Key,
			Property:           ref.Property,
			Version:            ref.Version,
			MetadataPolicy:     string(ref.MetadataPolicy),
			ConversionStrategy: string(ref.ConversionStrategy),
			DecodingStrategy:   string(ref.DecodingStrategy),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("unable to rpc: %w", err)
	}
	if res.Error != "" {
		return nil, fmt.Errorf("rpc error: %s", res.Error)
	}
	return res.Data, nil
}

func (s *Client) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	storeBytes, err := json.Marshal(s.store)
	if err != nil {
		return nil, err
	}

	findRef := &pb.ExternalSecretFind{
		Tags:               ref.Tags,
		ConversionStrategy: string(ref.ConversionStrategy),
		DecodingStrategy:   string(ref.DecodingStrategy),
	}
	if ref.Path != nil {
		findRef.Path = *ref.Path
	}
	if ref.Name != nil {
		findRef.FindNameRegexp = ref.Name.RegExp
	}
	res, err := s.grpcClient.GetAllSecrets(ctx, &pb.GetAllSecretsRequest{
		Store:     storeBytes,
		Namespace: s.namespace,
		RemoteRef: findRef,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to rpc: %w", err)
	}
	if res.Error != "" {
		return nil, fmt.Errorf("rpc error: %s", res.Error)
	}
	return res.Data, nil
}

func (s *Client) Close(ctx context.Context) error {
	return s.conn.Close()
}
