/*
Copyright © 2022 ESO Maintainer Team

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
package shell

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	pb "github.com/external-secrets/external-secrets/pkg/plugin/grpc"
	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/yaml"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type Server struct {
	pb.UnimplementedSecretsClientServer
	provider esapi.Provider
	scheme   *runtime.Scheme
	log      logr.Logger
}

func init() {
	ctrllog.SetLogger(zap.New())
}

func RunServer(provider esapi.Provider) error {
	log := ctrl.Log.WithName("provider")
	providerName, ok := esapi.GetProviderNameByType(provider)
	if !ok {
		return errors.New("could not get provider name by type")
	}

	scheme := runtime.NewScheme()
	esapi.AddToScheme(scheme)
	clientgoscheme.AddToScheme(scheme)

	pluginServer := &Server{
		provider: provider,
		scheme:   scheme,
		log:      log,
	}
	sockAddr := fmt.Sprintf("/var/run/eso/provider/sockets/%s.sock", providerName)
	_ = os.Remove(sockAddr)
	lis, err := net.Listen("unix", sockAddr)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	defer lis.Close()
	defer os.Remove(sockAddr)
	s := grpc.NewServer()

	go func() {
		c := make(chan os.Signal, 1) // we need to reserve to buffer size 1, so the notifier are not blocked
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)

		<-c
		log.Info("stopping grpc server")
		s.GracefulStop()
	}()

	pb.RegisterSecretsClientServer(s, pluginServer)
	log.Info("server listening ", "addr", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Error(err, "failed to serve")
		return err
	}
	return nil
}

func (s *Server) GetSecret(ctx context.Context, req *pb.GetSecretRequest) (*pb.GetSecretReply, error) {
	store, err := s.decodeStore(req.Store)
	if err != nil {
		return nil, err
	}
	kubeClient, err := s.getClient(req.Objects)
	if err != nil {
		return nil, err
	}
	s.log.Info("GetSecret() request", "namespace", req.Namespace, "name", store.GetObjectMeta().Name)
	secretsClient, err := s.provider.NewClient(ctx, store, kubeClient, req.Namespace)
	if err != nil {
		return nil, err
	}
	secret, err := secretsClient.GetSecret(ctx, remoteRef(req.RemoteRef))
	s.log.Info("GetSecret() response", "namespace", req.Namespace, "name", store.GetObjectMeta().Name, "secret", secret, "err", err)
	if err != nil {
		return &pb.GetSecretReply{
			Error: err.Error(),
		}, nil
	}
	return &pb.GetSecretReply{
		Secret: secret,
	}, nil
}

func (s *Server) GetSecretMap(ctx context.Context, req *pb.GetSecretMapRequest) (*pb.GetSecretMapReply, error) {
	store, err := s.decodeStore(req.Store)
	if err != nil {
		return nil, err
	}
	s.log.Info("GetSecretMap()", "namespace", req.Namespace, "name", store.GetObjectMeta().Name)
	kubeClient, err := s.getClient(req.Objects)
	if err != nil {
		return nil, err
	}
	secretsClient, err := s.provider.NewClient(ctx, store, kubeClient, req.Namespace)
	if err != nil {
		return nil, err
	}
	secret, err := secretsClient.GetSecretMap(ctx, remoteRef(req.RemoteRef))
	if err != nil {
		return &pb.GetSecretMapReply{
			Error: err.Error(),
		}, nil
	}
	return &pb.GetSecretMapReply{
		Data: secret,
	}, nil
}

func (s *Server) GetAllSecrets(ctx context.Context, req *pb.GetAllSecretsRequest) (*pb.GetAllSecretsReply, error) {
	store, err := s.decodeStore(req.Store)
	if err != nil {
		return nil, err
	}
	s.log.Info("GetAllSecrets()", "namespace", req.Namespace, "name", store.GetObjectMeta().Name)
	kubeClient, err := s.getClient(req.Objects)
	if err != nil {
		return nil, err
	}
	secretsClient, err := s.provider.NewClient(ctx, store,
		kubeClient, req.Namespace)
	if err != nil {
		return nil, err
	}
	secret, err := secretsClient.GetAllSecrets(ctx, externalSecretFind(req.RemoteRef))
	if err != nil {
		return &pb.GetAllSecretsReply{
			Error: err.Error(),
		}, nil
	}
	return &pb.GetAllSecretsReply{
		Data: secret,
	}, nil
}

func (s *Server) PushSecret(ctx context.Context, req *pb.PushSecretRequest) (*pb.PushSecretReply, error) {
	store, err := s.decodeStore(req.Store)
	if err != nil {
		return nil, err
	}
	s.log.Info("PushSecret()", "namespace", req.Namespace, "name", store.GetObjectMeta().Name)
	kubeClient, err := s.getClient(req.Objects)
	if err != nil {
		return nil, err
	}
	secretsClient, err := s.provider.NewClient(ctx, store, kubeClient, req.Namespace)
	if err != nil {
		return nil, err
	}
	err = secretsClient.PushSecret(ctx, req.Secret, pushRemoteRef(req.RemoteRef))
	if err != nil {
		return &pb.PushSecretReply{
			Error: err.Error(),
		}, nil
	}
	return &pb.PushSecretReply{}, nil
}

func (s *Server) DeleteSecret(ctx context.Context, req *pb.DeleteSecretRequest) (*pb.DeleteSecretReply, error) {
	store, err := s.decodeStore(req.Store)
	if err != nil {
		return nil, err
	}
	s.log.Info("DeleteSecret()", "namespace", req.Namespace, "name", store.GetObjectMeta().Name)
	kubeClient, err := s.getClient(req.Objects)
	if err != nil {
		return nil, err
	}
	secretsClient, err := s.provider.NewClient(ctx, store, kubeClient, req.Namespace)
	if err != nil {
		return nil, err
	}
	err = secretsClient.DeleteSecret(ctx, pushRemoteRef(req.RemoteRef))
	if err != nil {
		return &pb.DeleteSecretReply{
			Error: err.Error(),
		}, nil
	}
	return &pb.DeleteSecretReply{}, nil
}

func externalSecretFind(ref *pb.ExternalSecretFind) esapi.ExternalSecretFind {
	find := esapi.ExternalSecretFind{
		Tags:               ref.Tags,
		ConversionStrategy: esapi.ExternalSecretConversionStrategy(ref.ConversionStrategy),
		DecodingStrategy:   esapi.ExternalSecretDecodingStrategy(ref.GetDecodingStrategy()),
	}

	if ref.Path != "" {
		find.Path = &ref.Path
	}
	if ref.FindNameRegexp != "" {
		find.Name = &esapi.FindName{
			RegExp: ref.FindNameRegexp,
		}
	}
	return find
}

func pushRemoteRef(ref *pb.PushRemoteRef) esapi.PushRemoteRef {
	return esv1alpha1.PushSecretRemoteRef{
		RemoteKey: ref.RemoteKey,
		Property:  ref.Property,
	}
}

func remoteRef(ref *pb.RemoteRef) esapi.ExternalSecretDataRemoteRef {
	return esapi.ExternalSecretDataRemoteRef{
		Key:                ref.Key,
		MetadataPolicy:     esapi.ExternalSecretMetadataPolicy(ref.MetadataPolicy),
		Property:           ref.Property,
		Version:            ref.Version,
		ConversionStrategy: esapi.ExternalSecretConversionStrategy(ref.ConversionStrategy),
		DecodingStrategy:   esapi.ExternalSecretDecodingStrategy(ref.DecodingStrategy),
	}
}

func (s *Server) decodeStore(data []byte) (esapi.GenericStore, error) {
	obj, gvk, err := serializer.NewCodecFactory(s.scheme).UniversalDeserializer().Decode(data, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to decode store data: %w", err)
	}
	switch gvk.Kind {
	case esapi.SecretStoreKind:
		ss, ok := obj.(*esapi.SecretStore)
		if !ok {
			return nil, fmt.Errorf("unable to convert SecretStore object")
		}
		return ss, nil
	case esapi.ClusterSecretStoreKind:
		css, ok := obj.(*esapi.ClusterSecretStore)
		if !ok {
			return nil, fmt.Errorf("unable to convert SecretStore object")
		}
		return css, nil
	}
	return nil, errors.New("unexpected store data")
}

func (s *Server) decodeObjects(data []byte) ([]client.Object, error) {
	s.log.Info("decoded data", "data", data)
	decode := serializer.NewCodecFactory(s.scheme).UniversalDeserializer().Decode
	reader := yaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))
	var objects []client.Object
	for {
		buf, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				s.log.Error(err, "found EOF")
				break
			}
			s.log.Error(err, "unable to read buf")
			return nil, err
		}
		s.log.Error(err, "decoding buf", "buf", buf)
		obj, gvk, err := decode(buf, nil, nil)
		if err != nil {
			return nil, err
		}
		s.log.Info("decoded object", "object", obj)

		switch item := obj.(type) {
		case *corev1.Secret:
			objects = append(objects, item)
		case *corev1.ConfigMap:
			objects = append(objects, item)
		default:
			return nil, fmt.Errorf("unexpected object type: %s", gvk)
		}
	}
	return objects, nil
}

func (s *Server) getClient(data []byte) (client.Client, error) {
	objects, err := s.decodeObjects(data)
	if err != nil {
		return nil, err
	}
	restCfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, err
	}
	// TODO: do not log secrets
	s.log.Info("creating client with cached objects", "objects", objects)
	kubeClient, err := client.New(restCfg, client.Options{
		Cache: &client.CacheOptions{
			Reader: fake.NewClientBuilder().WithScheme(s.scheme).WithObjects(objects...).Build(),
		},
		Scheme: s.scheme,
	})
	if err != nil {
		return nil, err
	}
	return kubeClient, nil
}
