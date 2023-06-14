package remote

import (
	"context"
	"errors"
	"log"
	"reflect"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type AggregateResult struct {
	SecretKeySelectors      []v1.SecretKeySelector
	ServiceAccountSelectors []v1.ServiceAccountSelector
	CAProviders             []esapi.CAProvider
}

func aggregateObjects(ctx context.Context, store esapi.GenericStore, kube client.Client, namespace string) ([]byte, error) {
	if store == nil {
		return nil, errors.New("store can not be nil")
	}
	res := &AggregateResult{}
	storeSpec := store.GetSpec()
	iterate(storeSpec.Provider, res)

	var objects []byte
	for _, sel := range res.SecretKeySelectors {
		var secret corev1.Secret
		key := types.NamespacedName{
			Name:      sel.Name,
			Namespace: namespace,
		}
		if store.GetKind() == esapi.ClusterSecretStoreKind && sel.Namespace != nil {
			key.Namespace = *sel.Namespace
		}
		err := kube.Get(ctx, key, &secret)
		if err != nil {
			return nil, err
		}
		secret.SetGroupVersionKind(schema.GroupVersionKind{
			Kind:    "Secret",
			Version: "v1",
		})
		secretBytes, err := yaml.Marshal(secret)
		if err != nil {
			return nil, err
		}
		objects = append(objects, secretBytes...)
		objects = append(objects, []byte("\n---\n")...)
	}

	return objects, nil
}

func iterate(data interface{}, res *AggregateResult) {
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
			analyse(val, res)
			vv := reflect.Indirect(val)
			if vv.IsValid() {
				iterate(vv.Interface(), res)
			}
		}
	}
}

func analyse(val reflect.Value, res *AggregateResult) {
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

	caProviderT := reflect.TypeOf(esapi.CAProvider{})
	if val.Type().AssignableTo(caProviderT) {
		sel := val.Interface().(esapi.CAProvider)
		res.CAProviders = append(res.CAProviders, sel)
		return
	}

	// TODO: add more types that are of interest...
}
