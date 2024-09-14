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

package keepersecurity

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	ksm "github.com/keeper-security/secrets-manager-go/core"
	corev1 "k8s.io/api/core/v1"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/provider/keepersecurity/fake"
	testingfake "github.com/external-secrets/external-secrets/pkg/provider/testing/fake"
)

const (
	folderID            = "a8ekf031k"
	validExistingRecord = "record0/login"
	invalidRecord       = "record5/login"
	outputRecord0       = "{\"title\":\"record0\",\"type\":\"login\",\"fields\":[{\"type\":\"login\",\"value\":[\"foo\"]},{\"type\":\"password\",\"value\":[\"bar\"]}],\"custom\":[{\"type\":\"host\",\"label\":\"host0\",\"value\":[{\"hostName\":\"mysql\",\"port\":\"3306\"}]}],\"files\":null}"
	outputRecord1       = "{\"title\":\"record1\",\"type\":\"login\",\"fields\":[{\"type\":\"login\",\"value\":[\"foo\"]},{\"type\":\"password\",\"value\":[\"bar\"]}],\"custom\":[{\"type\":\"host\",\"label\":\"host1\",\"value\":[{\"hostName\":\"mysql\",\"port\":\"3306\"}]}],\"files\":null}"
	outputRecord2       = "{\"title\":\"record2\",\"type\":\"login\",\"fields\":[{\"type\":\"login\",\"value\":[\"foo\"]},{\"type\":\"password\",\"value\":[\"bar\"]}],\"custom\":[{\"type\":\"host\",\"label\":\"host2\",\"value\":[{\"hostName\":\"mysql\",\"port\":\"3306\"}]}],\"files\":null}"
	record0             = "record0"
	record1             = "record1"
	record2             = "record2"
	LoginKey            = "login"
	PasswordKey         = "password"
	HostKeyFormat       = "host%d"
	RecordNameFormat    = "record%d"
)

func TestClientDeleteSecret(t *testing.T) {
	type fields struct {
		ksmClient SecurityClient
		folderID  string
	}
	type args struct {
		ctx       context.Context
		remoteRef v1beta1.PushSecretRemoteRef
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Delete valid secret",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					DeleteSecretsFn: func(recrecordUids []string) (map[string]string, error) {
						return map[string]string{
							record0: record0,
						}, nil
					},
					GetSecretByTitleFn: func(recordTitle string) (*ksm.Record, error) {
						return generateRecords()[0], nil
					},
				},
				folderID: folderID,
			},
			args: args{
				context.Background(),
				&v1alpha1.PushSecretRemoteRef{
					RemoteKey: validExistingRecord,
				},
			},
			wantErr: false,
		},
		{
			name: "Delete invalid secret type",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretByTitleFn: func(recordTitle string) (*ksm.Record, error) {
						return generateRecords()[1], nil
					},
				},
				folderID: folderID,
			},
			args: args{
				context.Background(),
				&v1alpha1.PushSecretRemoteRef{
					RemoteKey: validExistingRecord,
				},
			},
			wantErr: true,
		},
		{
			name: "Delete non existing secret",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretByTitleFn: func(recordTitle string) (*ksm.Record, error) {
						return nil, errors.New("failed")
					},
				},
				folderID: folderID,
			},
			args: args{
				context.Background(),
				&v1alpha1.PushSecretRemoteRef{
					RemoteKey: invalidRecord,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				ksmClient: tt.fields.ksmClient,
				folderID:  tt.fields.folderID,
			}
			if err := c.DeleteSecret(tt.args.ctx, tt.args.remoteRef); (err != nil) != tt.wantErr {
				t.Errorf("DeleteSecret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClientGetAllSecrets(t *testing.T) {
	type fields struct {
		ksmClient SecurityClient
		folderID  string
	}
	type args struct {
		ctx context.Context
		ref v1beta1.ExternalSecretFind
	}
	var path = "path_to_fail"
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[string][]byte
		wantErr bool
	}{
		{
			name: "Tags not Implemented",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{},
				folderID:  folderID,
			},
			args: args{
				ctx: context.Background(),
				ref: v1beta1.ExternalSecretFind{
					Tags: map[string]string{
						"xxx": "yyy",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Path not Implemented",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{},
				folderID:  folderID,
			},
			args: args{
				ctx: context.Background(),
				ref: v1beta1.ExternalSecretFind{
					Path: &path,
				},
			},
			wantErr: true,
		},
		{
			name: "Get secrets with matching regex",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(strings []string) ([]*ksm.Record, error) {
						return generateRecords(), nil
					},
				},
				folderID: folderID,
			},
			args: args{
				ctx: context.Background(),
				ref: v1beta1.ExternalSecretFind{
					Name: &v1beta1.FindName{
						RegExp: "record",
					},
				},
			},
			want: map[string][]byte{
				record0: []byte(outputRecord0),
				record1: []byte(outputRecord1),
				record2: []byte(outputRecord2),
			},
			wantErr: false,
		},
		{
			name: "Get 1 secret with matching regex",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(strings []string) ([]*ksm.Record, error) {
						return generateRecords(), nil
					},
				},
				folderID: folderID,
			},
			args: args{
				ctx: context.Background(),
				ref: v1beta1.ExternalSecretFind{
					Name: &v1beta1.FindName{
						RegExp: record0,
					},
				},
			},
			want: map[string][]byte{
				record0: []byte(outputRecord0),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				ksmClient: tt.fields.ksmClient,
				folderID:  tt.fields.folderID,
			}
			got, err := c.GetAllSecrets(tt.args.ctx, tt.args.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAllSecrets() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAllSecrets() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientGetSecret(t *testing.T) {
	type fields struct {
		ksmClient SecurityClient
		folderID  string
	}
	type args struct {
		ctx context.Context
		ref v1beta1.ExternalSecretDataRemoteRef
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "Get Secret with a property",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						return []*ksm.Record{generateRecords()[0]}, nil
					},
				},
				folderID: folderID,
			},
			args: args{
				ctx: context.Background(),
				ref: v1beta1.ExternalSecretDataRemoteRef{
					Key:      record0,
					Property: LoginKey,
				},
			},
			want:    []byte("foo"),
			wantErr: false,
		},
		{
			name: "Get Secret without property",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						return []*ksm.Record{generateRecords()[0]}, nil
					},
				},
				folderID: folderID,
			},
			args: args{
				ctx: context.Background(),
				ref: v1beta1.ExternalSecretDataRemoteRef{
					Key: record0,
				},
			},
			want:    []byte(outputRecord0),
			wantErr: false,
		},
		{
			name: "Get non existing secret",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						return nil, errors.New("not found")
					},
				},
				folderID: folderID,
			},
			args: args{
				ctx: context.Background(),
				ref: v1beta1.ExternalSecretDataRemoteRef{
					Key: "record5",
				},
			},
			wantErr: true,
		},
		{
			name: "Get valid secret with non existing property",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						return []*ksm.Record{generateRecords()[0]}, nil
					},
				},
				folderID: folderID,
			},
			args: args{
				ctx: context.Background(),
				ref: v1beta1.ExternalSecretDataRemoteRef{
					Key:      record0,
					Property: "invalid",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				ksmClient: tt.fields.ksmClient,
				folderID:  tt.fields.folderID,
			}
			got, err := c.GetSecret(tt.args.ctx, tt.args.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSecret() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientGetSecretMap(t *testing.T) {
	type fields struct {
		ksmClient SecurityClient
		folderID  string
	}
	type args struct {
		ctx context.Context
		ref v1beta1.ExternalSecretDataRemoteRef
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[string][]byte
		wantErr bool
	}{
		{
			name: "Get Secret with valid property",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						return []*ksm.Record{generateRecords()[0]}, nil
					},
				},
				folderID: folderID,
			},
			args: args{
				ctx: context.Background(),
				ref: v1beta1.ExternalSecretDataRemoteRef{
					Key:      record0,
					Property: LoginKey,
				},
			},
			want: map[string][]byte{
				LoginKey: []byte("foo"),
			},
			wantErr: false,
		},
		{
			name: "Get Secret without property",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						return []*ksm.Record{generateRecords()[0]}, nil
					},
				},
				folderID: folderID,
			},
			args: args{
				ctx: context.Background(),
				ref: v1beta1.ExternalSecretDataRemoteRef{
					Key: record0,
				},
			},
			want: map[string][]byte{
				LoginKey:                      []byte("foo"),
				PasswordKey:                   []byte("bar"),
				fmt.Sprintf(HostKeyFormat, 0): []byte("{\"hostName\":\"mysql\",\"port\":\"3306\"}"),
			},
			wantErr: false,
		},
		{
			name: "Get non existing secret",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						return nil, errors.New("not found")
					},
				},
				folderID: folderID,
			},
			args: args{
				ctx: context.Background(),
				ref: v1beta1.ExternalSecretDataRemoteRef{
					Key: "record5",
				},
			},
			wantErr: true,
		},
		{
			name: "Get Secret with invalid property",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						return []*ksm.Record{generateRecords()[0]}, nil
					},
				},
				folderID: folderID,
			},
			args: args{
				ctx: context.Background(),
				ref: v1beta1.ExternalSecretDataRemoteRef{
					Key:      record0,
					Property: "invalid",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				ksmClient: tt.fields.ksmClient,
				folderID:  tt.fields.folderID,
			}
			got, err := c.GetSecretMap(tt.args.ctx, tt.args.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSecretMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSecretMap() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientPushSecret(t *testing.T) {
	secretKey := "secret-key"
	type fields struct {
		ksmClient SecurityClient
		folderID  string
	}
	type args struct {
		value []byte
		data  testingfake.PushSecretData
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Invalid remote ref",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{},
				folderID:  folderID,
			},
			args: args{
				data: testingfake.PushSecretData{
					SecretKey: secretKey,
					RemoteKey: record0,
				},
				value: []byte("foo"),
			},
			wantErr: true,
		},
		{
			name: "Push new valid secret",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretByTitleFn: func(recordTitle string) (*ksm.Record, error) {
						return nil, errors.New("NotFound")
					},
					CreateSecretWithRecordDataFn: func(recUID, folderUid string, recordData *ksm.RecordCreate) (string, error) {
						return "record5", nil
					},
				},
				folderID: folderID,
			},
			args: args{
				data: testingfake.PushSecretData{
					SecretKey: secretKey,
					RemoteKey: invalidRecord,
				},
				value: []byte("foo"),
			},
			wantErr: false,
		},
		{
			name: "Push existing valid secret",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretByTitleFn: func(recordTitle string) (*ksm.Record, error) {
						return generateRecords()[0], nil
					},
					SaveFn: func(record *ksm.Record) error {
						return nil
					},
				},
				folderID: folderID,
			},
			args: args{
				data: testingfake.PushSecretData{
					SecretKey: secretKey,
					RemoteKey: validExistingRecord,
				},
				value: []byte("foo2"),
			},
			wantErr: false,
		},
		{
			name: "Push existing invalid secret",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretByTitleFn: func(recordTitle string) (*ksm.Record, error) {
						return generateRecords()[1], nil
					},
					SaveFn: func(record *ksm.Record) error {
						return nil
					},
				},
				folderID: folderID,
			},
			args: args{
				data: testingfake.PushSecretData{
					SecretKey: secretKey,
					RemoteKey: validExistingRecord,
				},
				value: []byte("foo2"),
			},
			wantErr: true,
		},
		{
			name: "Unable to push new valid secret",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretByTitleFn: func(recordTitle string) (*ksm.Record, error) {
						return nil, errors.New("NotFound")
					},
					CreateSecretWithRecordDataFn: func(recUID, folderUID string, recordData *ksm.RecordCreate) (string, error) {
						return "", errors.New("Unable to push")
					},
				},
				folderID: folderID,
			},
			args: args{
				data: testingfake.PushSecretData{
					SecretKey: secretKey,
					RemoteKey: invalidRecord,
				},
				value: []byte("foo"),
			},
			wantErr: true,
		},
		{
			name: "Unable to save existing valid secret",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretByTitleFn: func(recordTitle string) (*ksm.Record, error) {
						return generateRecords()[0], nil
					},
					SaveFn: func(record *ksm.Record) error {
						return errors.New("Unable to save")
					},
				},
				folderID: folderID,
			},
			args: args{
				data: testingfake.PushSecretData{
					SecretKey: secretKey,
					RemoteKey: validExistingRecord,
				},
				value: []byte("foo2"),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				ksmClient: tt.fields.ksmClient,
				folderID:  tt.fields.folderID,
			}
			s := &corev1.Secret{Data: map[string][]byte{secretKey: tt.args.value}}
			if err := c.PushSecret(context.Background(), s, tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("PushSecret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func generateRecords() []*ksm.Record {
	var records []*ksm.Record
	for i := 0; i < 3; i++ {
		var record ksm.Record
		if i == 0 {
			record = ksm.Record{
				Uid: fmt.Sprintf(RecordNameFormat, i),
				RecordDict: map[string]any{
					"type":      externalSecretType,
					"folderUID": folderID,
				},
			}
		} else {
			record = ksm.Record{
				Uid: fmt.Sprintf(RecordNameFormat, i),
				RecordDict: map[string]any{
					"type":      LoginType,
					"folderUID": folderID,
				},
			}
		}
		sec := fmt.Sprintf("{\"title\":\"record%d\",\"type\":\"login\",\"fields\":[{\"type\":\"login\",\"value\":[\"foo\"]},{\"type\":\"password\",\"value\":[\"bar\"]}],\"custom\":[{\"type\":\"host\",\"label\":\"host%d\",\"value\":[{\"hostName\":\"mysql\",\"port\":\"3306\"}]}]}", i, i)
		record.SetTitle(fmt.Sprintf(RecordNameFormat, i))
		record.SetStandardFieldValue(LoginKey, "foo")
		record.SetStandardFieldValue(PasswordKey, "bar")
		record.RawJson = sec
		records = append(records, &record)
	}

	return records
}
