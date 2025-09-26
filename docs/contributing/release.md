ESO and the ESO Helm Chart have two distinct lifecycles and can be released independently. Helm Chart releases are named `external-secrets-x.y.z`.

The external-secrets project is released on a as-needed basis. Feel free to open a issue to request a release.

## Release ESO

When doing a release it's best to start with  with the ["Create Release" issue template](https://github.com/external-secrets/external-secrets/issues/new?assignees=&labels=area%2Frelease&projects=&template=create_release.md&title=Release+x.y), it has a checklist to go over.

⚠️ Note: when releasing multiple versions, make sure to first release the "old" version, then the newer version.
Otherwise the `latest` documentation will point to the older version. Also avoid to release both versions at the same time to avoid race conditions in the CI pipeline (updating docs, GitHub Release, helm chart release).

1. Run `Create Release` Action to create a new release, pass in the desired version number to release.
    1. choose the right `branch` to execute the action: use `main` when creating a new release.
    2. ⚠️ make sure that CI on the relevant branch has completed the docker build/push jobs. Otherwise an old image will be promoted.
2. GitHub Release, Changelog will be created by the `release.yml` workflow which also promotes the container image.
3. update Helm Chart, see below

## Release Helm Chart

1. Update `version` and/or `appVersion` in `Chart.yaml` and run `make helm.docs helm.update.appversion helm.test.update docs.update test.crds.update`
2. push to branch and open pr
3. run `/ok-to-test-managed` commands for all cloud providers
4. merge PR if everything is green
5. CI picks up the new chart version and creates a new GitHub Release for it

The following things are updated with those commands:
1. Update helm docs
2. Update the apiVersion in the snapshots for the helm tests
3. Update all the helm tests with potential added values
4. Update the stability docs with the latest minor version if exists
5. Update the CRD conformance tests

The branch to create this release should be `release-chart-x.y.z`. Though be aware that release branches are _immutable_.
This means that if there is anything that needs to be fixed, a new branch will need to be created.

Also, keep an eye on `main` so nothing is merged while the chart branch is running the e2e tests. If that happens,
the chart PR CANNOT be merged because we don't allow not up-to-date pull requests to be merged. And you can't update
because the branch is immutable.
