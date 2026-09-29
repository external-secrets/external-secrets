/*
Copyright © The ESO Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package keepersecurity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	ksm "github.com/keeper-security/secrets-manager-go/core"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/providers/v1/keepersecurity/fake"
	testingfake "github.com/external-secrets/external-secrets/runtime/testing/fake"
)

const (
	folderID               = "a8ekf031k"
	validExistingRecord    = "record0/login"
	invalidRecord          = "record5/login"
	outputRecord0          = "{\"title\":\"record0\",\"type\":\"login\",\"fields\":[{\"type\":\"login\",\"value\":[\"foo\"]},{\"type\":\"password\",\"value\":[\"bar\"]}],\"custom\":[{\"type\":\"host\",\"label\":\"host0\",\"value\":[{\"hostName\":\"mysql\",\"port\":\"3306\"}]}],\"files\":null}"
	outputRecord1          = "{\"title\":\"record1\",\"type\":\"login\",\"fields\":[{\"type\":\"login\",\"value\":[\"foo\"]},{\"type\":\"password\",\"value\":[\"bar\"]}],\"custom\":[{\"type\":\"host\",\"label\":\"host1\",\"value\":[{\"hostName\":\"mysql\",\"port\":\"3306\"}]}],\"files\":null}"
	outputRecord2          = "{\"title\":\"record2\",\"type\":\"login\",\"fields\":[{\"type\":\"login\",\"value\":[\"foo\"]},{\"type\":\"password\",\"value\":[\"bar\"]}],\"custom\":[{\"type\":\"host\",\"label\":\"host2\",\"value\":[{\"hostName\":\"mysql\",\"port\":\"3306\"}]}],\"files\":null}"
	outputRecordWithLabels = "{\"title\":\"recordWithLabels\",\"type\":\"login\",\"fields\":[{\"type\":\"login\",\"label\":\"username\",\"value\":[\"foo\"]},{\"type\":\"password\",\"label\":\"pass\",\"value\":[\"bar\"]}],\"custom\":[{\"type\":\"host\",\"label\":\"host0\",\"value\":[{\"hostName\":\"mysql\",\"port\":\"3306\"}]}],\"files\":null}"
	record0                = "record0"
	record1                = "record1"
	record2                = "record2"
	recordWithLabels       = "recordWithLabels"
	LoginKey               = "login"
	PasswordKey            = "password"
	HostKeyFormat          = "host%d"
	RecordNameFormat       = "record%d"
	UsernameLabel          = "username"
	PassLabel              = "pass"
)

func TestClientDeleteSecret(t *testing.T) {
	type fields struct {
		ksmClient SecurityClient
		folderID  string
	}
	type args struct {
		ctx       context.Context
		remoteRef esv1.PushSecretRemoteRef
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
					GetSecretsByTitleFn: func(recordTitle string) (records []*ksm.Record, err error) {
						return generateRecords()[:1], nil
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
			name: "Delete secret with multiple matches by Name",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					DeleteSecretsFn: func(recrecordUids []string) (map[string]string, error) {
						return map[string]string{
							record0: record0,
						}, nil
					},
					GetSecretsByTitleFn: func(recordTitle string) (records []*ksm.Record, err error) {
						return []*ksm.Record{generateRecords()[0], generateRecords()[0]}, nil
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
					GetSecretsByTitleFn: func(recordTitle string) (records []*ksm.Record, err error) {
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

func TestClientDeleteSecretTargets(t *testing.T) {
	newRecord := func() *ksm.Record {
		return &ksm.Record{Uid: "record-uid", RecordDict: map[string]any{
			"type": externalSecretType,
			"fields": []any{
				map[string]any{"type": LoginType, "label": "login", "value": []any{"admin"}},
				map[string]any{"type": LoginType, "label": "username", "value": []any{"alice"}},
			},
			"custom": []any{
				map[string]any{"type": secretType, "label": "token", "value": []any{"token-value"}},
			},
		}}
	}

	t.Run("whole record", func(t *testing.T) {
		deleted := false
		client := &Client{ksmClient: &fake.MockKeeperClient{
			GetSecretsByTitleFn: func(string) ([]*ksm.Record, error) { return []*ksm.Record{newRecord()}, nil },
			DeleteSecretsFn: func([]string) (map[string]string, error) {
				deleted = true
				return nil, nil
			},
		}}
		if err := client.DeleteSecret(context.Background(), &v1alpha1.PushSecretRemoteRef{RemoteKey: record0}); err != nil {
			t.Fatal(err)
		}
		if !deleted {
			t.Fatal("expected whole-record deletion")
		}
	})

	t.Run("property field preserves sibling label", func(t *testing.T) {
		var saved *ksm.Record
		client := &Client{ksmClient: &fake.MockKeeperClient{
			GetSecretsByTitleFn: func(string) ([]*ksm.Record, error) { return []*ksm.Record{newRecord()}, nil },
			SaveFn: func(record *ksm.Record) error {
				saved = record
				return nil
			},
		}}
		if err := client.DeleteSecret(context.Background(), &v1alpha1.PushSecretRemoteRef{RemoteKey: record0, Property: "username"}); err != nil {
			t.Fatal(err)
		}
		if saved == nil || len(saved.GetFieldsByLabel("username")) != 0 || len(saved.GetFieldsByLabel("login")) != 1 {
			t.Fatal("expected username deletion to preserve login")
		}
		if strings.Contains(saved.RawJson, "username") {
			t.Fatal("expected saved JSON to omit the removed field")
		}
	})

	t.Run("property field deletes an empty record", func(t *testing.T) {
		record := &ksm.Record{Uid: "record-uid", RecordDict: map[string]any{
			"type":   externalSecretType,
			"fields": []any{map[string]any{"type": LoginType, "label": "", "value": []any{"alice"}}},
		}}
		deleted := false
		client := &Client{ksmClient: &fake.MockKeeperClient{
			GetSecretsByTitleFn: func(string) ([]*ksm.Record, error) { return []*ksm.Record{record}, nil },
			DeleteSecretsFn: func([]string) (map[string]string, error) {
				deleted = true
				return nil, nil
			},
		}}
		if err := client.DeleteSecret(context.Background(), &v1alpha1.PushSecretRemoteRef{RemoteKey: record0, Property: "login"}); err != nil {
			t.Fatal(err)
		}
		if !deleted {
			t.Fatal("expected deletion of the last managed field to delete the record")
		}
	})

	t.Run("legacy custom field deletes an empty record", func(t *testing.T) {
		record := &ksm.Record{Uid: "record-uid", RecordDict: map[string]any{
			"type":   externalSecretType,
			"custom": []any{map[string]any{"type": secretType, "label": "token", "value": []any{"value"}}},
		}}
		deleted := false
		client := &Client{ksmClient: &fake.MockKeeperClient{
			GetSecretsByTitleFn: func(string) ([]*ksm.Record, error) { return []*ksm.Record{record}, nil },
			DeleteSecretsFn: func([]string) (map[string]string, error) {
				deleted = true
				return nil, nil
			},
		}}
		if err := client.DeleteSecret(context.Background(), &v1alpha1.PushSecretRemoteRef{RemoteKey: record0 + "/token"}); err != nil {
			t.Fatal(err)
		}
		if !deleted {
			t.Fatal("expected deletion of the last legacy custom field to delete the record")
		}
	})
}

func TestClientSecretExists(t *testing.T) {
	newRecord := func() *ksm.Record {
		return &ksm.Record{RecordDict: map[string]any{
			"type":   externalSecretType,
			"fields": []any{map[string]any{"type": LoginType, "label": "login", "value": []any{"alice"}}},
			"custom": []any{map[string]any{"type": secretType, "label": "token", "value": []any{"value"}}},
		}}
	}
	for _, tt := range []struct {
		name string
		ref  *v1alpha1.PushSecretRemoteRef
		want bool
	}{
		{name: "whole record", ref: &v1alpha1.PushSecretRemoteRef{RemoteKey: record0}, want: true},
		{name: "property", ref: &v1alpha1.PushSecretRemoteRef{RemoteKey: record0, Property: "token"}, want: true},
		{name: "legacy", ref: &v1alpha1.PushSecretRemoteRef{RemoteKey: record0 + "/login"}, want: true},
		{name: "missing field", ref: &v1alpha1.PushSecretRemoteRef{RemoteKey: record0, Property: "username"}, want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{ksmClient: &fake.MockKeeperClient{
				GetSecretsByTitleFn: func(string) ([]*ksm.Record, error) { return []*ksm.Record{newRecord()}, nil },
			}}
			got, err := client.SecretExists(context.Background(), tt.ref)
			if err != nil || got != tt.want {
				t.Fatalf("SecretExists() = %v, %v; want %v, nil", got, err, tt.want)
			}
		})
	}
}

func TestClientUpdateSecret(t *testing.T) {
	newRecord := func() *ksm.Record {
		return &ksm.Record{RecordDict: map[string]any{
			"type":  externalSecretType,
			"notes": "preserve",
			"fields": []any{
				map[string]any{"type": LoginType, "label": "login", "value": []any{"old-login"}},
				map[string]any{"type": LoginType, "label": "username", "value": []any{"old-username"}},
			},
			"custom": []any{map[string]any{"type": secretType, "label": "stale", "value": []any{"old"}}},
		}}
	}
	client := &Client{ksmClient: &fake.MockKeeperClient{SaveFn: func(*ksm.Record) error { return nil }}}

	t.Run("whole record replaces stale fields", func(t *testing.T) {
		record := newRecord()
		desired := &Secret{Fields: []Field{{Type: LoginType, Label: "username", Value: []any{"new"}}}, Custom: []CustomField{{Type: secretType, Label: "token", Value: []any{"new-token"}}}}
		if err := client.updateSecret(record, desired, true); err != nil {
			t.Fatal(err)
		}
		if keeperRecordHasField(record, "login") || keeperRecordHasField(record, "stale") || !keeperRecordHasField(record, "username") || !keeperRecordHasField(record, "token") {
			t.Fatal("whole-record replacement did not reconcile fields")
		}
		if record.RecordDict["notes"] != "preserve" || strings.Contains(record.RawJson, "stale") {
			t.Fatal("whole-record replacement did not preserve metadata or refresh JSON")
		}
	})

	t.Run("targeted update adds differently labeled standard field", func(t *testing.T) {
		record := &ksm.Record{RecordDict: map[string]any{"type": externalSecretType, "fields": []any{map[string]any{"type": LoginType, "label": "login", "value": []any{"old"}}}}}
		if err := client.updateSecret(record, &Secret{Fields: []Field{{Type: LoginType, Label: "username", Value: []any{"new"}}}}, false); err != nil {
			t.Fatal(err)
		}
		if len(record.GetFieldsByLabel("login")) != 1 || len(record.GetFieldsByLabel("username")) != 1 {
			t.Fatal("targeted update overwrote a differently labeled field")
		}
	})

	t.Run("legacy update preserves the default login field", func(t *testing.T) {
		record := &ksm.Record{RecordDict: map[string]any{"type": externalSecretType, "fields": []any{map[string]any{"type": LoginType, "value": []any{"old"}}}}}
		if err := client.updateSecret(record, &Secret{Fields: []Field{{Type: LoginType, Value: []any{"new"}}}}, false); err != nil {
			t.Fatal(err)
		}
		fields := record.GetFieldsByType(LoginType)
		if len(fields) != 1 || fields[0]["label"] != nil || fields[0]["value"].([]any)[0] != "new" {
			t.Fatalf("legacy update did not update the default login field: %#v", fields)
		}
	})
}

func TestClientPushSecretSerializesRecordCreate(t *testing.T) {
	var recordJSON string
	client := &Client{
		folderID: folderID,
		ksmClient: &fake.MockKeeperClient{
			GetSecretsByTitleFn: func(string) ([]*ksm.Record, error) { return nil, nil },
			CreateSecretWithRecordDataFn: func(_, _ string, record *ksm.RecordCreate) (string, error) {
				recordJSON = record.ToJson()
				return "record-uid", nil
			},
		},
	}

	secret := &corev1.Secret{Data: map[string][]byte{
		"login":     []byte("alice"),
		"api-token": []byte("token-value"),
	}}
	data := &v1alpha1.PushSecretData{Match: v1alpha1.PushSecretMatch{
		RemoteRef: v1alpha1.PushSecretRemoteRef{RemoteKey: "record"},
	}}
	if err := client.PushSecret(context.Background(), secret, data); err != nil {
		t.Fatal(err)
	}

	var created map[string]any
	if err := json.Unmarshal([]byte(recordJSON), &created); err != nil {
		t.Fatalf("RecordCreate.ToJson() returned invalid JSON: %v", err)
	}
	fields := created["fields"].([]any)
	custom := created["custom"].([]any)
	if len(fields) != 1 || fields[0].(map[string]any)["label"] != "login" || fields[0].(map[string]any)["value"].([]any)[0] != "alice" {
		t.Fatalf("unexpected serialized standard fields: %#v", fields)
	}
	if len(custom) != 1 || custom[0].(map[string]any)["label"] != "api-token" || custom[0].(map[string]any)["value"].([]any)[0] != "token-value" {
		t.Fatalf("unexpected serialized custom fields: %#v", custom)
	}
}

func TestLegacyPushSecretCompatibility(t *testing.T) {
	client := &Client{}
	secret := &corev1.Secret{Data: map[string][]byte{"username": []byte("alice")}}
	data := &v1alpha1.PushSecretData{Match: v1alpha1.PushSecretMatch{
		SecretKey: "username",
		RemoteRef: v1alpha1.PushSecretRemoteRef{RemoteKey: "legacy/username"},
	}}

	record, err := client.buildLegacyRecord(secret, data)
	if err != nil {
		t.Fatal(err)
	}
	if len(record.Fields) != 1 || record.Fields[0].Type != LoginType || record.Fields[0].Label != "" {
		t.Fatalf("legacy username must map to the default login field: %#v", record.Fields)
	}

	target := resolvePushTarget(data.Match.RemoteRef)
	if target.fieldKey != LoginType {
		t.Fatalf("legacy username target = %#v; want default login", target)
	}
}

func TestClientGetAllSecrets(t *testing.T) {
	type fields struct {
		ksmClient SecurityClient
		folderID  string
	}
	type args struct {
		ctx context.Context
		ref esv1.ExternalSecretFind
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
				ref: esv1.ExternalSecretFind{
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
				ref: esv1.ExternalSecretFind{
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
				ref: esv1.ExternalSecretFind{
					Name: &esv1.FindName{
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
				ref: esv1.ExternalSecretFind{
					Name: &esv1.FindName{
						RegExp: record0,
					},
				},
			},
			want: map[string][]byte{
				record0: []byte(outputRecord0),
			},
			wantErr: false,
		},
		{
			name: "Get secrets with labels using matching regex",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(strings []string) ([]*ksm.Record, error) {
						return []*ksm.Record{generateRecordWithLabels()}, nil
					},
				},
				folderID: folderID,
			},
			args: args{
				ctx: context.Background(),
				ref: esv1.ExternalSecretFind{
					Name: &esv1.FindName{
						RegExp: recordWithLabels,
					},
				},
			},
			want: map[string][]byte{
				recordWithLabels: []byte(outputRecordWithLabels),
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
		ksmClient          SecurityClient
		folderID           string
		getByTitleFallback bool
	}
	type args struct {
		ctx context.Context
		ref esv1.ExternalSecretDataRemoteRef
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
		// wantNoSecretErr is asserted for every case, so a failure that is not a
		// missing record must not carry the sentinel either.
		wantNoSecretErr bool
	}{
		{
			name: "Get Secret with a property (no label)",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						return []*ksm.Record{generateRecords()[0]}, nil
					},
				},
				folderID:           folderID,
				getByTitleFallback: false,
			},
			args: args{
				ctx: context.Background(),
				ref: esv1.ExternalSecretDataRemoteRef{
					Key:      record0,
					Property: LoginKey,
				},
			},
			want:    []byte("foo"),
			wantErr: false,
		},
		{
			name: "Get Secret without property (no label)",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						return []*ksm.Record{generateRecords()[0]}, nil
					},
				},
				folderID:           folderID,
				getByTitleFallback: false,
			},
			args: args{
				ctx: context.Background(),
				ref: esv1.ExternalSecretDataRemoteRef{
					Key: record0,
				},
			},
			want:    []byte(outputRecord0),
			wantErr: false,
		},
		{
			name: "Get Secret with a property using label",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						return []*ksm.Record{generateRecordWithLabels()}, nil
					},
				},
				folderID:           folderID,
				getByTitleFallback: false,
			},
			args: args{
				ctx: context.Background(),
				ref: esv1.ExternalSecretDataRemoteRef{
					Key:      recordWithLabels,
					Property: UsernameLabel,
				},
			},
			want:    []byte("foo"),
			wantErr: false,
		},
		{
			name: "Get Secret with a property using type when label exists",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						return []*ksm.Record{generateRecordWithLabels()}, nil
					},
				},
				folderID:           folderID,
				getByTitleFallback: false,
			},
			args: args{
				ctx: context.Background(),
				ref: esv1.ExternalSecretDataRemoteRef{
					Key:      recordWithLabels,
					Property: LoginKey, // Try to access by type when label exists
				},
			},
			wantErr: true, // Should fail because label takes precedence
		},
		{
			name: "Get Secret without property (with labels)",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						return []*ksm.Record{generateRecordWithLabels()}, nil
					},
				},
				folderID:           folderID,
				getByTitleFallback: false,
			},
			args: args{
				ctx: context.Background(),
				ref: esv1.ExternalSecretDataRemoteRef{
					Key: recordWithLabels,
				},
			},
			want:    []byte(outputRecordWithLabels),
			wantErr: false,
		},
		{
			name: "Get secret with multiple matches by ID",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						return []*ksm.Record{generateRecords()[0], generateRecords()[0]}, nil
					},
				},
				folderID:           folderID,
				getByTitleFallback: false,
			},
			args: args{
				ctx: context.Background(),
				ref: esv1.ExternalSecretDataRemoteRef{
					Key: record0,
				},
			},
			want:    []byte(outputRecord0),
			wantErr: false,
		},
		{
			name: "Get secret by ID",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						return []*ksm.Record{generateRecords()[0], generateRecords()[0]}, nil
					},
				},
				folderID:           folderID,
				getByTitleFallback: false,
			},
			args: args{
				ctx: context.Background(),
				ref: esv1.ExternalSecretDataRemoteRef{
					Key: record0,
				},
			},
			want:    []byte(outputRecord0),
			wantErr: false,
		},
		{
			name: "Get secret by ID (with fallback)",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						return []*ksm.Record{generateRecords()[0], generateRecords()[0]}, nil
					},
				},
				folderID:           folderID,
				getByTitleFallback: true,
			},
			args: args{
				ctx: context.Background(),
				ref: esv1.ExternalSecretDataRemoteRef{
					Key: record0,
				},
			},
			want:    []byte(outputRecord0),
			wantErr: false,
		},
		{
			name: "Get non existing secret with client error",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						return nil, errors.New("not found")
					},
				},
				folderID:           folderID,
				getByTitleFallback: false,
			},
			args: args{
				ctx: context.Background(),
				ref: esv1.ExternalSecretDataRemoteRef{
					Key: "record5",
				},
			},
			wantErr: true,
		},
		{
			name: "Get non existing secret",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						return []*ksm.Record{}, nil
					},
				},
				folderID:           folderID,
				getByTitleFallback: false,
			},
			args: args{
				ctx: context.Background(),
				ref: esv1.ExternalSecretDataRemoteRef{
					Key: "record5",
				},
			},
			wantErr:         true,
			wantNoSecretErr: true,
		},
		{
			name: "Get non existing secret with fallback",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						return []*ksm.Record{}, nil
					},
					GetSecretsByTitleFn: func(recordTitle string) ([]*ksm.Record, error) {
						return []*ksm.Record{}, nil
					},
				},
				folderID:           folderID,
				getByTitleFallback: true,
			},
			args: args{
				ctx: context.Background(),
				ref: esv1.ExternalSecretDataRemoteRef{
					Key: "record5",
				},
			},
			wantErr:         true,
			wantNoSecretErr: true,
		},
		{
			name: "Get valid secret with non existing property",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						return []*ksm.Record{generateRecords()[0]}, nil
					},
				},
				folderID:           folderID,
				getByTitleFallback: false,
			},
			args: args{
				ctx: context.Background(),
				ref: esv1.ExternalSecretDataRemoteRef{
					Key:      record0,
					Property: "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "Get secret by name",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						// Return empty list to trigger name lookup
						return []*ksm.Record{}, nil
					},
					GetSecretsByTitleFn: func(recordTitle string) ([]*ksm.Record, error) {
						return []*ksm.Record{generateRecords()[0]}, nil
					},
				},
				folderID:           folderID,
				getByTitleFallback: true,
			},
			args: args{
				ctx: context.Background(),
				ref: esv1.ExternalSecretDataRemoteRef{
					Key: record0,
				},
			},
			want:    []byte(outputRecord0),
			wantErr: false,
		},
		{
			name: "Get secret by name with multiple matches",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						// Return empty list to trigger name lookup
						return []*ksm.Record{}, nil
					},
					GetSecretsByTitleFn: func(recordTitle string) ([]*ksm.Record, error) {
						return []*ksm.Record{generateRecords()[0], generateRecords()[0]}, nil
					},
				},
				folderID:           folderID,
				getByTitleFallback: true,
			},
			args: args{
				ctx: context.Background(),
				ref: esv1.ExternalSecretDataRemoteRef{
					Key: record0,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				ksmClient:          tt.fields.ksmClient,
				folderID:           tt.fields.folderID,
				getByTitleFallback: tt.fields.getByTitleFallback,
			}
			got, err := c.GetSecret(tt.args.ctx, tt.args.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if isNoSecret := errors.Is(err, esv1.NoSecretErr); isNoSecret != tt.wantNoSecretErr {
				t.Errorf("GetSecret() errors.Is(err, NoSecretErr) = %v, want %v (err = %v)", isNoSecret, tt.wantNoSecretErr, err)
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
		ref esv1.ExternalSecretDataRemoteRef
	}
	tests := []struct {
		name            string
		fields          fields
		args            args
		want            map[string][]byte
		wantErr         bool
		wantNoSecretErr bool
	}{
		{
			name: "Get Secret with valid property (no label)",
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
				ref: esv1.ExternalSecretDataRemoteRef{
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
			name: "Get Secret without property (no label)",
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
				ref: esv1.ExternalSecretDataRemoteRef{
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
			name: "Get Secret with valid property using label",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						return []*ksm.Record{generateRecordWithLabels()}, nil
					},
				},
				folderID: folderID,
			},
			args: args{
				ctx: context.Background(),
				ref: esv1.ExternalSecretDataRemoteRef{
					Key:      recordWithLabels,
					Property: UsernameLabel,
				},
			},
			want: map[string][]byte{
				UsernameLabel: []byte("foo"),
			},
			wantErr: false,
		},
		{
			name: "Get Secret without property (with labels)",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						return []*ksm.Record{generateRecordWithLabels()}, nil
					},
				},
				folderID: folderID,
			},
			args: args{
				ctx: context.Background(),
				ref: esv1.ExternalSecretDataRemoteRef{
					Key: recordWithLabels,
				},
			},
			want: map[string][]byte{
				UsernameLabel:                 []byte("foo"),
				PassLabel:                     []byte("bar"),
				fmt.Sprintf(HostKeyFormat, 0): []byte("{\"hostName\":\"mysql\",\"port\":\"3306\"}"),
			},
			wantErr: false,
		},
		{
			// The API call itself fails here, so this must not be reported as a
			// missing record.
			name: "Get secret when the API call fails",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						return nil, errors.New("keeper API unavailable")
					},
				},
				folderID: folderID,
			},
			args: args{
				ctx: context.Background(),
				ref: esv1.ExternalSecretDataRemoteRef{
					Key: "record5",
				},
			},
			wantErr: true,
		},
		{
			name: "Get non existing secret",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
						return []*ksm.Record{}, nil
					},
				},
				folderID: folderID,
			},
			args: args{
				ctx: context.Background(),
				ref: esv1.ExternalSecretDataRemoteRef{
					Key: "record5",
				},
			},
			wantErr:         true,
			wantNoSecretErr: true,
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
				ref: esv1.ExternalSecretDataRemoteRef{
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
			if isNoSecret := errors.Is(err, esv1.NoSecretErr); isNoSecret != tt.wantNoSecretErr {
				t.Errorf("GetSecretMap() errors.Is(err, NoSecretErr) = %v, want %v (err = %v)", isNoSecret, tt.wantNoSecretErr, err)
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
					GetSecretsByTitleFn: func(recordTitle string) (records []*ksm.Record, err error) {
						return generateRecords()[0:0], nil
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
					GetSecretsByTitleFn: func(recordTitle string) (records []*ksm.Record, err error) {
						return generateRecords()[0:1], nil
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
			name: "Unable to push new valid secret with multiple matches by Name",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsByTitleFn: func(recordTitle string) (records []*ksm.Record, err error) {
						return []*ksm.Record{generateRecords()[0], generateRecords()[0]}, nil
					},
				},
				folderID: folderID,
			},
			args: args{
				data: testingfake.PushSecretData{
					SecretKey: secretKey,
					RemoteKey: validExistingRecord,
				},
				value: []byte("foo"),
			},
			wantErr: true,
		},
		{
			name: "Unable to push new valid secret",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsByTitleFn: func(recordTitle string) (records []*ksm.Record, err error) {
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
			name: "Push new secret fails without folderID",
			fields: fields{
				ksmClient: &fake.MockKeeperClient{
					GetSecretsByTitleFn: func(recordTitle string) (records []*ksm.Record, err error) {
						return generateRecords()[0:0], nil
					},
				},
				folderID: "",
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
					GetSecretsByTitleFn: func(recordTitle string) (records []*ksm.Record, err error) {
						return generateRecords()[0:1], nil
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
	records := make([]*ksm.Record, 0, 3)
	for i := range 3 {
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
		sec := fmt.Sprintf(
			"{\"title\":\"record%d\",\"type\":\"login\",\"fields\":[{\"type\":\"login\",\"value\":[\"foo\"]},{\"type\":\"password\",\"value\":[\"bar\"]}],\"custom\":[{\"type\":\"host\",\"label\":\"host%d\",\"value\":[{\"hostName\":\"mysql\",\"port\":\"3306\"}]}]}",
			i,
			i,
		)
		record.SetTitle(fmt.Sprintf(RecordNameFormat, i))
		record.SetStandardFieldValue(LoginKey, "foo")
		record.SetStandardFieldValue(PasswordKey, "bar")
		record.RawJson = sec
		records = append(records, &record)
	}

	return records
}

func generateRecordWithLabels() *ksm.Record {
	record := ksm.Record{
		Uid: recordWithLabels,
		RecordDict: map[string]any{
			"type":      externalSecretType,
			"folderUID": folderID,
		},
	}
	// Fields with labels - using label as key
	sec := fmt.Sprintf(
		"{\"title\":%q,\"type\":\"login\",\"fields\":[{\"type\":\"login\",\"label\":%q,\"value\":[\"foo\"]},{\"type\":\"password\",\"label\":%q,\"value\":[\"bar\"]}],\"custom\":[{\"type\":\"host\",\"label\":\"host0\",\"value\":[{\"hostName\":\"mysql\",\"port\":\"3306\"}]}]}",
		recordWithLabels,
		UsernameLabel,
		PassLabel,
	)
	record.SetTitle(recordWithLabels)
	record.SetStandardFieldValue(LoginKey, "foo")
	record.SetStandardFieldValue(PasswordKey, "bar")
	record.RawJson = sec
	return &record
}
