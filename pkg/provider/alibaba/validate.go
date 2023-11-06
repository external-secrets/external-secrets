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
package alibaba
 
import (
	//"context"
	//"encoding/json"
	"fmt"

	 openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	 kmssdk "github.com/alibabacloud-go/kms-20160120/v3/client"
	 util "github.com/alibabacloud-go/tea-utils/v2/service"
	 credential "github.com/aliyun/credentials-go/credentials"
	 	"github.com/avast/retry-go/v4"
		"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	 kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils"
)
// type KeyManagementService struct {
// 	Client SMInterfacetype KeyManagementService struct {
// 	Client SMInterface
// 	Config *openapi.Config
// // }
// 	Config *openapi.Config
// // }
// type SMInterface interface {
// 	GetSecretValue(ctx context.Context, request *kmssdk.GetSecretValueRequest) (*kmssdk.GetSecretValueResponseBody, error)
// 	Endpoint() string
// }

func (kms *KeyManagementService) ValidateStore(store esv1beta1.GenericStore) error {
	
		storeSpec := store.GetSpec()
		  if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.Kubernetes == nil {
			return fmt.Errorf("no store type or wrong store type")
		  }
		
		 alibabaSpec := storeSpec.Provider.Alibaba
	   
		 regionID := alibabaSpec.RegionID
	   
		 if regionID == "" {
		  return fmt.Errorf("missing alibaba region")
		 }
		 
			
		 if alibabaSpec.Auth.rrsa != nil {
	    	if alibabaSpec.Auth.rrsa.ServiceAccountRef != nil {
			   if err := utils.ValidateReferentServiceAccountSelector(store, *alibabaSpec.Auth.rrsa.ServiceAccountRef); err != nil {
			   return fmt.Errorf(errInvalidKubeSA, err)
		    }
		  }
		 
		  accessKeyID := alibabaSpec.Auth.SecretRef.AccessKeyID
		  err := utils.ValidateSecretSelector(store, accessKeyID)
		   if err != nil {
			   return err
		  }
	   
		 if alibabaSpec.Auth.rrsa.oidcProviderArn == "" {
		  return fmt.Errorf("missing alibaba oidc-provider-arn")
		  }
		 if alibabaSpec.Auth.rrsa.oidcTokenFilePath == "" {
		  return fmt.Errorf("missing alibaba oidc-token-file-path")
		  }
		  if alibabaSpec.Auth.rrsa.roleArn == "" {
		   return fmt.Errorf("missing alibaba role-arn")
		  }
		  if alibabaSpec.Auth.rrsa.sessionName == "" {
		   return fmt.Errorf("missing alibaba session-name")
		  }  
		 
		  return nil
		 }
		return kms.validateStoreAuth(store)
}
	   
	
   