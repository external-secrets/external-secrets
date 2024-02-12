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

package fake

import ksm "github.com/keeper-security/secrets-manager-go/core"

type MockKeeperClient struct {
	GetSecretsFn                 func([]string) ([]*ksm.Record, error)
	GetSecretByTitleFn           func(recordTitle string) (*ksm.Record, error)
	CreateSecretWithRecordDataFn func(recUID, folderUID string, recordData *ksm.RecordCreate) (string, error)
	DeleteSecretsFn              func(recrecordUids []string) (map[string]string, error)
	SaveFn                       func(record *ksm.Record) error
}

type GetSecretsMockReturn struct {
	Secrets []*ksm.Record
	Err     error
}

type GetSecretsByTitleMockReturn struct {
	Secret *ksm.Record
	Err    error
}

type CreateSecretWithRecordDataMockReturn struct {
	ID  string
	Err error
}

func (mc *MockKeeperClient) GetSecrets(filter []string) ([]*ksm.Record, error) {
	return mc.GetSecretsFn(filter)
}

func (mc *MockKeeperClient) GetSecretByTitle(recordTitle string) (*ksm.Record, error) {
	return mc.GetSecretByTitleFn(recordTitle)
}

func (mc *MockKeeperClient) CreateSecretWithRecordData(recUID, folderUID string, recordData *ksm.RecordCreate) (string, error) {
	return mc.CreateSecretWithRecordDataFn(recUID, folderUID, recordData)
}

func (mc *MockKeeperClient) DeleteSecrets(recrecordUids []string) (map[string]string, error) {
	return mc.DeleteSecretsFn(recrecordUids)
}

func (mc *MockKeeperClient) Save(record *ksm.Record) error {
	return mc.SaveFn(record)
}
