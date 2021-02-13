package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Config contains configuration to create a new AWS provider.
type Config struct {
	AssumeRole string
	Region     string
	APIRetries int
}

var log = ctrl.Log.WithName("provider").WithName("aws")

// NewSession creates a new aws session based on the supported input methods.
// https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials
func NewSession(sak, aks, region, role string, stsprovider STSProvider) (*session.Session, error) {
	config := aws.NewConfig()
	sessionOpts := session.Options{
		Config: *config,
	}
	if sak != "" && aks != "" {
		sessionOpts.Config.Credentials = credentials.NewStaticCredentials(aks, sak, "")
		sessionOpts.SharedConfigState = session.SharedConfigDisable
	}
	sess, err := session.NewSessionWithOptions(sessionOpts)
	if err != nil {
		return nil, fmt.Errorf("unable to create aws session: %w", err)
	}
	if region != "" {
		log.V(1).Info("using region", "region", region)
		sess.Config.WithRegion(region)
	}

	if role != "" {
		log.V(1).Info("assuming role", "role", role)
		stsclient := stsprovider(sess)
		sess.Config.WithCredentials(stscreds.NewCredentialsWithClient(stsclient, role))
	}
	sess.Handlers.Build.PushBack(request.WithAppendUserAgent("external-secrets"))
	return sess, nil
}

type STSProvider func(*session.Session) stscreds.AssumeRoler

func DefaultSTSProvider(sess *session.Session) stscreds.AssumeRoler {
	return sts.New(sess)
}
