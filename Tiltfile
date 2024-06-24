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

# Update the root security group. Tilt requires root access to update the
# running process.
objects = decode_yaml_stream(read_file('bin/deploy/manifests/external-secrets.yaml'))
for o in objects:
    if o.get('kind') == 'Deployment' and o.get('metadata').get('name') in ['external-secrets-cert-controller', 'external-secrets', 'external-secrets-webhook']:
        o['spec']['template']['spec']['containers'][0]['securityContext'] = {'runAsNonRoot': False, 'readOnlyRootFilesystem': False}
        o['spec']['template']['spec']['containers'][0]['imagePullPolicy'] = 'Always'
        if settings.get('debug').get('enabled') and o.get('metadata').get('name') == 'external-secrets':
            o['spec']['template']['spec']['containers'][0]['ports'] = [{'containerPort': 30000}]


updated_install = encode_yaml_stream(objects)

# Apply the updated yaml to the cluster.
k8s_yaml(updated_install, allow_duplicates = True)

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

local_resource(
    'external-secret-binary',
    "CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -gcflags '{gcflags}' -v -o bin/external-secrets ./".format(gcflags=gcflags),
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
