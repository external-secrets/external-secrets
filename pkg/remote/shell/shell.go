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
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	pb "github.com/external-secrets/external-secrets/pkg/plugin/grpc"
	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type Server struct {
	pb.UnimplementedSecretsClientServer
	provider   esapi.Provider
	scheme     *runtime.Scheme
	kubeClient client.Client
	log        logr.Logger
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
	restCfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return err
	}
	kubeClient, err := client.New(restCfg, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return err
	}

	pluginServer := &Server{
		provider:   provider,
		scheme:     scheme,
		kubeClient: kubeClient,
		log:        log,
	}
	sockAddr := fmt.Sprintf("/tmp/eso-%s.sock", providerName)
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
	s.log.Info("GetSecret()", "namespace", req.Namespace, "name", store.GetObjectMeta().Name)

	secretsClient, err := s.provider.NewClient(ctx, store, s.kubeClient, req.Namespace)
	if err != nil {
		return nil, err
	}
	secret, err := secretsClient.GetSecret(ctx, remoteRef(req.RemoteRef))
	if err != nil {
		// TODO: handle NoSecret error on the client side
		return &pb.GetSecretReply{
			Error: err.Error(),
		}, nil
	}
	return &pb.GetSecretReply{
		Secret: secret,
	}, nil
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
