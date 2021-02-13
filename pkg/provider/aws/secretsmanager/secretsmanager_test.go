package secretsmanager

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	awssm "github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

var _ = Describe("SSM", func() {

	It("Should create an Client using environment variables", func() {
		sm := &SecretsManager{}
		os.Setenv("AWS_SECRET_ACCESS_KEY", "1111")
		os.Setenv("AWS_ACCESS_KEY_ID", "2222")
		defer os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		defer os.Unsetenv("AWS_ACCESS_KEY_ID")
		smi, err := sm.New(context.Background(), &esv1alpha1.SecretStore{
			Spec: esv1alpha1.SecretStoreSpec{
				Provider: &esv1alpha1.SecretStoreProvider{
					// defaults
					AWSSM: &esv1alpha1.AWSSMProvider{},
				},
			},
		}, k8sClient, "example-ns")
		Expect(err).ToNot(HaveOccurred())
		Expect(smi).ToNot(BeNil())

		creds, err := sm.session.Config.Credentials.Get()
		Expect(err).ToNot(HaveOccurred())
		Expect(creds.AccessKeyID).To(Equal("2222"))
		Expect(creds.SecretAccessKey).To(Equal("1111"))
	})

	It("Should create an Client using environment variables and assume a role", func() {
		sts := &FakeSTS{
			AssumeRoleFunc: func(input *sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error) {
				Expect(*input.RoleArn).To(Equal("my-awesome-role"))
				return &sts.AssumeRoleOutput{
					AssumedRoleUser: &sts.AssumedRoleUser{
						Arn:           aws.String("1123132"),
						AssumedRoleId: aws.String("xxxxx"),
					},
					Credentials: &sts.Credentials{
						AccessKeyId:     aws.String("3333"),
						SecretAccessKey: aws.String("4444"),
						Expiration:      aws.Time(time.Now().Add(time.Hour)),
						SessionToken:    aws.String("6666"),
					},
				}, nil
			},
		}
		sm := &SecretsManager{
			stsProvider: func(se *session.Session) stscreds.AssumeRoler {
				// check if the correct temporary credentials were used
				creds, err := se.Config.Credentials.Get()
				Expect(err).ToNot(HaveOccurred())
				Expect(creds.AccessKeyID).To(Equal("2222"))
				Expect(creds.SecretAccessKey).To(Equal("1111"))
				return sts
			},
		}
		os.Setenv("AWS_SECRET_ACCESS_KEY", "1111")
		os.Setenv("AWS_ACCESS_KEY_ID", "2222")
		defer os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		defer os.Unsetenv("AWS_ACCESS_KEY_ID")
		smi, err := sm.New(context.Background(), &esv1alpha1.SecretStore{
			Spec: esv1alpha1.SecretStoreSpec{
				Provider: &esv1alpha1.SecretStoreProvider{
					// do assume role!
					AWSSM: &esv1alpha1.AWSSMProvider{
						Role: "my-awesome-role",
					},
				},
			},
		}, k8sClient, "example-ns")
		Expect(err).ToNot(HaveOccurred())
		Expect(smi).ToNot(BeNil())

		creds, err := sm.session.Config.Credentials.Get()
		Expect(err).ToNot(HaveOccurred())
		Expect(creds.AccessKeyID).To(Equal("3333"))
		Expect(creds.SecretAccessKey).To(Equal("4444"))
	})

	It("GetSecret should return the correct values", func() {
		fake := &FakeSM{}
		p := &SecretsManager{
			client: fake,
		}
		for i, row := range []struct {
			apiInput       *awssm.GetSecretValueInput
			apiOutput      *awssm.GetSecretValueOutput
			rr             esv1alpha1.ExternalSecretDataRemoteRef
			apiErr         error
			expectError    string
			expectedSecret string
		}{
			{
				// good case: default version is set
				// key is passed in, output is sent back
				apiInput: &awssm.GetSecretValueInput{
					SecretId:     aws.String("/baz"),
					VersionStage: aws.String("AWSCURRENT"),
				},
				rr: esv1alpha1.ExternalSecretDataRemoteRef{
					Key: "/baz",
				},
				apiOutput: &awssm.GetSecretValueOutput{
					SecretString: aws.String("RRRRR"),
				},
				apiErr:         nil,
				expectError:    "",
				expectedSecret: "RRRRR",
			},
			{
				// good case: extract property
				apiInput: &awssm.GetSecretValueInput{
					SecretId:     aws.String("/baz"),
					VersionStage: aws.String("AWSCURRENT"),
				},
				rr: esv1alpha1.ExternalSecretDataRemoteRef{
					Key:      "/baz",
					Property: "/shmoo",
				},
				apiOutput: &awssm.GetSecretValueOutput{
					SecretString: aws.String(`{"/shmoo": "bang"}`),
				},
				apiErr:         nil,
				expectError:    "",
				expectedSecret: "bang",
			},
			{
				// bad case: missing property
				apiInput: &awssm.GetSecretValueInput{
					SecretId:     aws.String("/baz"),
					VersionStage: aws.String("AWSCURRENT"),
				},
				rr: esv1alpha1.ExternalSecretDataRemoteRef{
					Key:      "/baz",
					Property: "DOES NOT EXIST",
				},
				apiOutput: &awssm.GetSecretValueOutput{
					SecretString: aws.String(`{"/shmoo": "bang"}`),
				},
				apiErr:         nil,
				expectError:    "has no property",
				expectedSecret: "",
			},
			{
				// bad case: extract property failure due to invalid json
				apiInput: &awssm.GetSecretValueInput{
					SecretId:     aws.String("/baz"),
					VersionStage: aws.String("AWSCURRENT"),
				},
				rr: esv1alpha1.ExternalSecretDataRemoteRef{
					Key:      "/baz",
					Property: "/shmoo",
				},
				apiOutput: &awssm.GetSecretValueOutput{
					SecretString: aws.String(`------`),
				},
				apiErr:         nil,
				expectError:    "unable to unmarshal secret",
				expectedSecret: "",
			},
			{
				// should pass version
				apiInput: &awssm.GetSecretValueInput{
					SecretId:     aws.String("/foo/bar"),
					VersionStage: aws.String("1234"),
				},
				rr: esv1alpha1.ExternalSecretDataRemoteRef{
					Key:     "/foo/bar",
					Version: "1234",
				},
				apiOutput: &awssm.GetSecretValueOutput{
					SecretString: aws.String("FOOBA!"),
				},
				apiErr:         nil,
				expectError:    "",
				expectedSecret: "FOOBA!",
			},
			{
				// should return err
				apiInput: &awssm.GetSecretValueInput{
					SecretId:     aws.String("/foo/bar"),
					VersionStage: aws.String("AWSCURRENT"),
				},
				rr: esv1alpha1.ExternalSecretDataRemoteRef{
					Key: "/foo/bar",
				},
				apiOutput:   &awssm.GetSecretValueOutput{},
				apiErr:      fmt.Errorf("oh no"),
				expectError: "oh no",
			},
		} {
			fake.WithValue(row.apiInput, row.apiOutput, row.apiErr)
			out, err := p.GetSecret(context.Background(), row.rr)
			if !ErrorContains(err, row.expectError) {
				GinkgoT().Errorf("[%d] unexpected error: %s, expected: '%s'", i, err.Error(), row.expectError)
			}
			if string(out) != row.expectedSecret {
				GinkgoT().Errorf("[%d] unexpected secret: expected %s, got %s", i, row.expectedSecret, string(out))
			}
		}
	})

	It("GetSecretMap should return correct values", func() {
		fake := &FakeSM{}
		p := &SecretsManager{
			client: fake,
		}
		for i, row := range []struct {
			apiInput     *awssm.GetSecretValueInput
			apiOutput    *awssm.GetSecretValueOutput
			rr           esv1alpha1.ExternalSecretDataRemoteRef
			expectedData map[string]string
			apiErr       error
			expectError  string
		}{
			{
				// good case: default version & deserialization
				apiInput: &awssm.GetSecretValueInput{
					SecretId:     aws.String("/baz"),
					VersionStage: aws.String("AWSCURRENT"),
				},
				apiOutput: &awssm.GetSecretValueOutput{
					SecretString: aws.String(`{"foo":"bar"}`),
				},
				rr: esv1alpha1.ExternalSecretDataRemoteRef{
					Key: "/baz",
				},
				expectedData: map[string]string{
					"foo": "bar",
				},
				apiErr:      nil,
				expectError: "",
			},
			{
				// bad case: api error returned
				apiInput: &awssm.GetSecretValueInput{
					SecretId:     aws.String("/baz"),
					VersionStage: aws.String("AWSCURRENT"),
				},
				apiOutput: &awssm.GetSecretValueOutput{
					SecretString: aws.String(`{"foo":"bar"}`),
				},
				rr: esv1alpha1.ExternalSecretDataRemoteRef{
					Key: "/baz",
				},
				expectedData: map[string]string{
					"foo": "bar",
				},
				apiErr:      fmt.Errorf("some api err"),
				expectError: "some api err",
			},
			{
				// bad case: invalid json
				apiInput: &awssm.GetSecretValueInput{
					SecretId:     aws.String("/baz"),
					VersionStage: aws.String("AWSCURRENT"),
				},
				apiOutput: &awssm.GetSecretValueOutput{
					SecretString: aws.String(`-----------------`),
				},
				rr: esv1alpha1.ExternalSecretDataRemoteRef{
					Key: "/baz",
				},
				expectedData: map[string]string{},
				apiErr:       nil,
				expectError:  "unable to unmarshal secret",
			},
		} {
			fake.WithValue(row.apiInput, row.apiOutput, row.apiErr)
			out, err := p.GetSecretMap(context.Background(), row.rr)
			if !ErrorContains(err, row.expectError) {
				GinkgoT().Errorf("[%d] unexpected error: %s, expected: '%s'", i, err.Error(), row.expectError)
			}
			if cmp.Equal(out, row.expectedData) {
				GinkgoT().Errorf("[%d] unexpected secret data: expected %#v, got %#v", i, row.expectedData, out)
			}
		}
	})
})

func ErrorContains(out error, want string) bool {
	if out == nil {
		return want == ""
	}
	if want == "" {
		return false
	}
	return strings.Contains(out.Error(), want)
}

type FakeSM struct {
	valFn func(*awssm.GetSecretValueInput) (*awssm.GetSecretValueOutput, error)
}

func (sm *FakeSM) GetSecretValue(in *awssm.GetSecretValueInput) (*awssm.GetSecretValueOutput, error) {
	return sm.valFn(in)
}

func (sm *FakeSM) WithValue(in *awssm.GetSecretValueInput, val *awssm.GetSecretValueOutput, err error) {
	sm.valFn = func(paramIn *awssm.GetSecretValueInput) (*awssm.GetSecretValueOutput, error) {
		if !cmp.Equal(paramIn, in) {
			return nil, fmt.Errorf("unexpected test argument")
		}
		return val, err
	}
}

type FakeSTS struct {
	AssumeRoleFunc func(*sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error)
}

func (f *FakeSTS) AssumeRole(input *sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error) {
	return f.AssumeRoleFunc(input)
}
