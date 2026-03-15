# -*- mode: Python -*-

kubectl_cmd = "kubectl"

# verify kubectl command exists
if str(local("command -v " + kubectl_cmd + " || true", quiet = True)) == "":
    fail("Required command '" + kubectl_cmd + "' not found in PATH")

# set defaults
settings = {
    "debug": {
        "enabled": False,
    },
}

# merge default settings with user defined settings
tilt_file = "./tilt-settings.yaml" if os.path.exists("./tilt-settings.yaml") else "./tilt-settings.json"
settings.update(read_yaml(
    tilt_file,
    default = {},
))

# set up the development environment

# Split the YAML into CRDs and other resources
objects = decode_yaml_stream(read_file('bin/deploy/manifests/external-secrets.yaml'))

crds = []
other_resources = []

for o in objects:
    if o.get('kind') == 'CustomResourceDefinition':
        crds.append(o)
    else:
        other_resources.append(o)

# Process deployments for development
for o in other_resources:
    if o.get('kind') == 'Deployment' and o.get('metadata').get('name') in ['external-secrets-cert-controller', 'external-secrets', 'external-secrets-webhook']:
        o['spec']['template']['spec']['containers'][0]['securityContext'] = {'runAsNonRoot': False, 'readOnlyRootFilesystem': False}
        o['spec']['template']['spec']['containers'][0]['imagePullPolicy'] = 'Always'
        if settings.get('debug').get('enabled') and o.get('metadata').get('name') == 'external-secrets':
            o['spec']['template']['spec']['containers'][0]['ports'] = [{'containerPort': 30000}]

# Create the directory
local('mkdir -p .tilt-tmp')

# Apply CRDs with server-side apply (handles large CRDs)
if crds:
    crd_yaml = encode_yaml_stream(crds)
    local('cat > .tilt-tmp/external-secrets-crds.yaml', stdin=crd_yaml)
    local_resource(
        'apply-crds',
        'kubectl apply --server-side -f .tilt-tmp/external-secrets-crds.yaml',
        deps=['bin/deploy/manifests/external-secrets.yaml']
    )

# Use regular k8s_yaml for deployments (Tilt will handle image substitution)
if other_resources:
    deployments_yaml = encode_yaml_stream(other_resources)
    local('cat > .tilt-tmp/external-secrets-deployments.yaml', stdin=deployments_yaml)
    k8s_yaml('.tilt-tmp/external-secrets-deployments.yaml')

load('ext://restart_process', 'docker_build_with_restart')

# enable hot reloading by doing the following:
# - locally build the whole project
# - create a docker imagine using tilt's hot-swap wrapper
# - push that container to the local tilt registry
# Once done, rebuilding now should be a lot faster since only the relevant
# binary is rebuilt and the hot swat wrapper takes care of the rest.
gcflags = ''
if settings.get('debug').get('enabled'):
    gcflags = '-N -l'

buildtags = settings.get('buildtags', 'all_providers')

local_resource(
    'external-secret-binary',
    "CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags '{buildtags}' -gcflags '{gcflags}' -v -o bin/external-secrets ./".format(buildtags=buildtags, gcflags=gcflags),
    deps = [
        "main.go",
        "go.mod",
        "go.sum",
        "apis",
        "cmd",
        "pkg",
    ],
)

# Build the docker image for our controller. We use a specific Dockerfile
# since tilt can't run on a scratch container.
# `only` here is important, otherwise, the container will get updated
# on _any_ file change. We only want to monitor the binary.
# If debugging is enabled, we switch to a different docker file using
# the delve port.
entrypoint = ['/external-secrets']
dockerfile = 'tilt.dockerfile'
if settings.get('debug').get('enabled'):
    k8s_resource('external-secrets', port_forwards=[
        port_forward(30000, 30000, 'debugger'),
    ])
    entrypoint = ['/dlv', '--listen=:30000', '--api-version=2', '--continue=true', '--accept-multiclient=true', '--headless=true', 'exec', '/external-secrets', '--']
    dockerfile = 'tilt.debug.dockerfile'

docker_build_with_restart(
    'ghcr.io/external-secrets/external-secrets',
    '.',
    dockerfile = dockerfile,
    entrypoint = entrypoint,
    only=[
      './bin',
    ],
    live_update = [
        sync('./bin/external-secrets', '/external-secrets'),
    ],
)
