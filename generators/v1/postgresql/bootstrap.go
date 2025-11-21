// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// Package postgresql implements the PostgreSQL bootstrap process.
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
package postgresql

import (
	"context"
	"fmt"
	"time"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	"github.com/external-secrets/external-secrets/runtime/scheduler"
	"github.com/go-logr/logr"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/gommon/log"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// Bootstrap implements the bootstrap process for PostgreSQL.
type Bootstrap struct {
	mgr    manager.Manager
	client client.Client
}

// NewBootstrap creates a new PostgreSQL bootstrap.
func NewBootstrap(client client.Client, mgr manager.Manager) *Bootstrap {
	return &Bootstrap{
		client: client,
		mgr:    mgr,
	}
}

// Start starts the bootstrap process.
func (b *Bootstrap) Start(ctx context.Context) error {
	if ok := b.mgr.GetCache().WaitForCacheSync(ctx); !ok {
		return ctx.Err()
	}

	client := b.mgr.GetClient()
	var list genv1alpha1.GeneratorStateList
	if err := client.List(ctx, &list); err != nil {
		return err
	}
	var db *pgx.Conn
	defer func() {
		if db != nil {
			err := db.Close(ctx)
			if err != nil {
				fmt.Printf("failed to close db: %v", err)
			}
		}
	}()
	for _, gs := range list.Items {
		if gs.Spec.GarbageCollectionDeadline == nil {
			log.Info("skipping generator state without garbage collection deadline")
			continue
		}

		spec, err := parseSpec(gs.Spec.Resource.Raw)
		if err != nil {
			return err
		}
		if spec.Kind != "PostgreSql" {
			// not a PostgreSql spec. skipping
			continue
		}
		if db != nil {
			err := db.Close(ctx)
			if err != nil {
				fmt.Printf("failed to close db: %v", err)
			}
		}
		db, err = newConnection(ctx, &spec.Spec, client, spec.Namespace)
		if err != nil {
			return fmt.Errorf("unable to create db connection: %w", err)
		}

		cleanupPolicy := spec.Spec.CleanupPolicy
		if cleanupPolicy != nil && cleanupPolicy.Type == genv1alpha1.IdleCleanupPolicy {
			connectionID := fmt.Sprintf(schedIDFmt, spec.UID, spec.Spec.Host, spec.Spec.Port)
			err = setupObservation(ctx, db)
			if err != nil {
				return fmt.Errorf("unable to setup observation: %w", err)
			}

			scheduler.Global().ScheduleInterval(connectionID, spec.Spec.CleanupPolicy.ActivityTrackingInterval.Duration, time.Minute, func(ctx context.Context, log logr.Logger) {
				err := triggerSessionSnapshot(ctx, &spec.Spec, b.client, gs.GetNamespace())
				if err != nil {
					log.Error(err, "failed to trigger session observation")
					return
				}
			})
		}
	}

	return nil
}
