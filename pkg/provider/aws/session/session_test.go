package session

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/stretchr/testify/assert"

	fakesm "github.com/external-secrets/external-secrets/pkg/provider/aws/secretsmanager/fake"
)

func TestSession(t *testing.T) {
	tbl := []struct {
		test              string
		aks               string
		sak               string
		region            string
		role              string
		sts               STSProvider
		expectedKeyID     string
		expectedSecretKey string
	}{
		{
			test:              "test default role provider",
			aks:               "2222",
			sak:               "1111",
			region:            "xxxxx",
			role:              "",
			sts:               DefaultSTSProvider,
			expectedSecretKey: "1111",
			expectedKeyID:     "2222",
		},
		{
			test:   "test custom sts provider",
			aks:    "1111",
			sak:    "2222",
			region: "xxxxx",
			role:   "zzzzz",
			sts: func(*session.Session) stscreds.AssumeRoler {
				return &fakesm.AssumeRoler{
					AssumeRoleFunc: func(input *sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error) {
						assert.Equal(t, *input.RoleArn, "zzzzz")
						return &sts.AssumeRoleOutput{
							AssumedRoleUser: &sts.AssumedRoleUser{
								Arn:           aws.String("1123132"),
								AssumedRoleId: aws.String("xxxxx"),
							},
							Credentials: &sts.Credentials{
								SecretAccessKey: aws.String("3333"),
								AccessKeyId:     aws.String("4444"),
								Expiration:      aws.Time(time.Now().Add(time.Hour)),
								SessionToken:    aws.String("6666"),
							},
						}, nil
					},
				}
			},
			expectedSecretKey: "3333",
			expectedKeyID:     "4444",
		},
	}
	for i := range tbl {
		row := tbl[i]
		t.Run(row.test, func(t *testing.T) {
			sess, err := New(row.sak, row.aks, row.region, row.role, row.sts)
			assert.Nil(t, err)
			creds, err := sess.Config.Credentials.Get()
			assert.Nil(t, err)
			assert.Equal(t, row.expectedKeyID, creds.AccessKeyID)
			assert.Equal(t, row.expectedSecretKey, creds.SecretAccessKey)
		})
	}
}
