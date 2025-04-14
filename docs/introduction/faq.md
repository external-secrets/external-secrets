## Can I manually trigger a secret refresh?

You can trigger a secret refresh by using kubectl or any other kubernetes api client.
You just need to change an annotation, label or the spec of the resource:

```
kubectl annotate es my-es force-sync=$(date +%s) --overwrite
```

## How do I know when my secret was last synced?


The last synchronization timestamp of an ExternalSecret can be retrieved from the field `refreshTime`. 

```
kubectl get es my-external-secret -o yaml | grep refreshTime
  refreshTime: "2022-05-21T23:02:47Z"
```

The interval can be changed by the `spec.refreshInterval` in the ExternalSecret.

## How do I know when the status of my secret changed the last time?

Every ExternalSecret resource contains a status condition that indicates whether a secret was successfully synchronized, along with the timestamp of the last status change of the ExternalSecret (e.g. from SecretSyncedError to SecretSynced). This can be obtained from the field `lastTransitionTime`:

```
kubectl get es my-external-secret -o yaml | grep condition -A 5
  conditions:
  - lastTransitionTime: "2022-05-21T21:02:47Z"
    message: Secret was synced
    reason: SecretSynced
    status: "True"
    type: Ready
```

## Differences to csi-secret-store
Please take a look at this [issue comment here](https://github.com/external-secrets/external-secrets/issues/478#issuecomment-964413129).

## How do I debug an external-secret that doesn't sync?

First, check the status of the ExternalSecret resource using `kubectl describe`. That displays the status conditions as well as recent events.
You should expect a status condition with `Type=Ready`, `Status=True`. Further you shouldn't see any events with `Type=Warning`. Read carefully if they exist.

```
kubectl describe es my-external-secret
[...]
Status:
  Conditions:
    Last Transition Time:   2022-05-21T21:02:47Z
    Message:                Secret was synced
    Reason:                 SecretSynced
    Status:                 True
    Type:                   Ready
  Refresh Time:             2022-05-21T21:06:47Z
  Synced Resource Version:  1-5c833527afd7ba3f426cb0082ee7e083
Events:
  Type     Reason        Age                  From              Message
  ----     ------        ----                 ----              -------
  Warning  UpdateFailed  4m12s                external-secrets  secrets "yyyyyyy" already exists
  Normal   Updated       12s (x4 over 3m12s)  external-secrets  Updated Secret
```

If everything looks good you should check the corresponding secret store resource that is referenced from an ExternalSecret. Again, use `kubectl describe` to show status conditions and events and look for warning signs as described above.

In an ideally, the store should be validated and Ready.

```
kubectl describe css kubernetes
[...]
Status:
  Conditions:
    Last Transition Time:  2022-05-21T21:02:47Z
    Message:               store validated
    Reason:                Valid
    Status:                True
    Type:                  Ready
Events:
  Type    Reason  Age                From                  Message
  ----    ------  ----               ----                  -------
  Normal  Valid   52s (x4 over 10m)  cluster-secret-store  store validated
  Normal  Valid   52s (x4 over 10m)  cluster-secret-store  store validated
```

If everything looks normal so far, please go ahead and ensure that the created secret has the expected value. Also, take a look at the logs of the controller.

## Upgrading from KES to ESO

Migrating from KES to ESO is quite tricky! There is a tool we built to help users out available [here](https://github.com/external-secrets/kes-to-eso), and there is a small migration procedure.

There are some incompatibilities between KES to ESO, and while the tool tries to cover most of them, some of them will require manual intervention. We recommend to first convert the manifest files, and actually see if the tool provides a warning about any file needed to be changed.
Beware that the tool points the SecretStores to use KES Service Account, so you'll also need to tweak that if you plan to uninstall KES after the upgrade.



