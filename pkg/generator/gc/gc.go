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
package gc

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/spf13/pflag"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/feature"
)

var gcGracePeriod time.Duration
var log = ctrl.Log.WithName("generator-gc")
var scheduler gocron.Scheduler

func init() {
	fs := pflag.NewFlagSet("gc", pflag.ExitOnError)
	fs.DurationVar(&gcGracePeriod, "generator-gc-grace-period", time.Minute*2, "Duration after which generated secrets are cleaned up after they have been flagged for gc.")
	feature.Register(feature.Feature{
		Flags: fs,
	})
	var err error
	scheduler, err = gocron.NewScheduler()
	if err != nil {
		panic(err)
	}
	scheduler.Start()
}

type Entry struct {
	Resource *apiextensions.JSON
	Impl     genv1alpha1.Generator
	State    genv1alpha1.GeneratorProviderState
}

func (e Entry) Key() string {
	h := sha256.New()
	h.Write(e.Resource.Raw)
	hash := h.Sum(e.State.Raw)
	return fmt.Sprintf("%x", hash)
}

func Enqueue(e Entry) error {
	log.V(1).Info("putting state into GC", "entry", e)
	_, err := scheduler.NewJob(
		gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(time.Now().Add(gcGracePeriod))),
		gocron.NewTask(func(entry Entry) {
			err := entry.Impl.Cleanup(context.Background(), entry.Resource, entry.State, nil, "")
			if err != nil {
				log.Error(err, "failed to cleanup generator secret", "generator", entry.Resource)
			}
		}, e))
	if err != nil {
		return err
	}
	return nil
}

func Cleanup(ctx context.Context, flaggedForGC time.Time, entry Entry, kclient client.Client, ns string) (bool, error) {
	log.V(1).Info("cleaning up generator", "entry", entry)
	if flaggedForGC.Add(gcGracePeriod).After(time.Now()) {
		log.V(1).Info("generator is not ready for cleanup", "entry", entry)
		return false, nil
	}
	err := entry.Impl.Cleanup(context.Background(), entry.Resource, entry.State, kclient, ns)
	if err != nil {
		return false, err
	}
	return true, nil
}
