package main

import (
	"context"
	"flag"
	"log"
	"reflect"
	"time"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	pb "github.com/external-secrets/external-secrets/pkg/plugin/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	addr = flag.String("addr", "unix:///tmp/plugin.sock", "the address to connect to")
)

func main() {
	flag.Parse()
	reflector()
	// Set up a connection to the server.
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewSecretsClientClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	res, err := c.GetSecret(ctx, &pb.GetSecretRequest{
		RemoteRef: &pb.RemoteRef{
			Key: "foo",
		},
	})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("secret=%s, err=%s", string(res.Secret), res.Error)
}

func reflector() {

	ns := "bar"
	prov := &v1beta1.SecretStoreProvider{
		Kubernetes: &v1beta1.KubernetesProvider{
			Auth: v1beta1.KubernetesAuth{
				Token: &v1beta1.TokenAuth{
					BearerToken: v1.SecretKeySelector{
						Name: "brr",
						Key:  "fart",
					},
				},
				ServiceAccount: &v1.ServiceAccountSelector{
					Name:      "ccccc",
					Namespace: &ns,
					Audiences: nil,
				},
			},
			Server: v1beta1.KubernetesServer{
				URL:      "asdasda",
				CABundle: []byte{1, 23, 4, 1, 231, 23, 1},
				CAProvider: &v1beta1.CAProvider{
					Type:      v1beta1.CAProviderTypeConfigMap,
					Name:      "ca",
					Key:       "ca.crt",
					Namespace: &ns,
				},
			},
		},
		Vault: &v1beta1.VaultProvider{
			Auth: v1beta1.VaultAuth{
				TokenSecretRef: &v1.SecretKeySelector{
					Name:      "foo",
					Namespace: &ns,
					Key:       "Baz",
				},
				Kubernetes: &v1beta1.VaultKubernetesAuth{
					ServiceAccountRef: &v1.ServiceAccountSelector{
						Name:      "kfoo",
						Namespace: &ns,
						Audiences: []string{"bzzzzing"},
					},
				},
			},
		},
	}

	res := &ItResult{}

	iterate(prov, res)

	log.Printf("=== RESULTS: %#v", res)
}

type ItResult struct {
	SecretKeySelectors      []v1.SecretKeySelector
	ServiceAccountSelectors []v1.ServiceAccountSelector
	CAProviders             []v1beta1.CAProvider
}

func iterate(data interface{}, res *ItResult) {
	log.Printf("iterate: %#v %#v", reflect.ValueOf(data).Interface(), res)
	if reflect.ValueOf(data).Kind() == reflect.Slice {
		d := reflect.ValueOf(data)
		for i := 0; i < d.Len(); i++ {
			val := d.Index(i)
			analyse(val, res)
			iterate(reflect.Indirect(d.Index(i)).Interface(), res)
		}
	} else if reflect.ValueOf(data).Kind() == reflect.Map {
		d := reflect.ValueOf(data)
		for _, k := range d.MapKeys() {
			typeOfValue := reflect.TypeOf(d.MapIndex(k).Interface()).Kind()
			if typeOfValue == reflect.Map || typeOfValue == reflect.Slice {
				val := d.MapIndex(k)
				analyse(val, res)
				iterate(reflect.Indirect(val).Interface(), res)
			} else {
				log.Printf("val not map or slice: %#v", typeOfValue)
			}
		}
	} else if reflect.ValueOf(data).Kind() == reflect.Pointer {
		originalValue := reflect.ValueOf(data).Elem()
		if !originalValue.IsValid() {
			return
		}
		iterate(reflect.Indirect(originalValue).Interface(), res)
	} else if reflect.ValueOf(data).Kind() == reflect.Struct {
		v := reflect.ValueOf(data)
		for _, f := range reflect.VisibleFields(v.Type()) {
			val := v.FieldByIndex(f.Index)
			log.Printf("checking field %s ftype=%s type=%s kind=%d|%s", f.Name, f.Type, val.Type(), val.Type().Kind(), val.Type().Kind())
			analyse(val, res)
			vv := reflect.Indirect(val)
			if vv.IsValid() {
				iterate(vv.Interface(), res)
			}
		}
	}
}

func analyse(val reflect.Value, res *ItResult) {
	log.Printf("analyse: %#v %#v", val, res)
	if val.Kind() == reflect.Pointer {
		originalValue := val.Elem()
		if !originalValue.IsValid() {
			return
		}
		analyse(reflect.Indirect(originalValue), res)
		return
	}

	secretSelT := reflect.TypeOf(v1.SecretKeySelector{})
	if val.Type().AssignableTo(secretSelT) {
		sel := val.Interface().(v1.SecretKeySelector)
		res.SecretKeySelectors = append(res.SecretKeySelectors, sel)
		return
	}

	serviceAccSelT := reflect.TypeOf(v1.ServiceAccountSelector{})
	if val.Type().AssignableTo(serviceAccSelT) {
		sel := val.Interface().(v1.ServiceAccountSelector)
		res.ServiceAccountSelectors = append(res.ServiceAccountSelectors, sel)
		return
	}

	caProviderT := reflect.TypeOf(v1beta1.CAProvider{})
	if val.Type().AssignableTo(caProviderT) {
		sel := val.Interface().(v1beta1.CAProvider)
		res.CAProviders = append(res.CAProviders, sel)
		return
	}

	// TODO: add more types that are of interest...
}
