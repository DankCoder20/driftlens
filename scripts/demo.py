#!/usr/bin/env python3
"""Disposable Kind demo. All mutations explicitly target driftlens-demo only."""
import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import time
import urllib.request
import urllib.error

ROOT = Path(__file__).resolve().parents[1]
DATA = ROOT / '.data'
CREDS = ROOT / 'credentials'
ADMIN = DATA / 'demo-admin.kubeconfig'
NAMESPACE = 'driftlens-demo'
CONTEXT = 'kind-driftlens-demo'
BASE = os.getenv('DRIFTLENS_URL', 'http://127.0.0.1:8080')

def run(*args, capture=False):
    result = subprocess.run(list(map(str, args)), cwd=ROOT, text=True, capture_output=capture)
    if result.returncode:
        raise RuntimeError(f'{args[0]} {args[1]} failed (exit {result.returncode})')
    return result.stdout.strip() if capture else ''

def kubectl(*args, capture=False):
    return run('kubectl', '--kubeconfig', ADMIN, '--context', CONTEXT, *args, capture=capture)

def token():
    value = os.getenv('DRIFTLENS_API_TOKEN', '')
    if not value and (ROOT / '.env').exists():
        for line in (ROOT / '.env').read_text().splitlines():
            if line.startswith('DRIFTLENS_API_TOKEN='):
                value = line.split('=', 1)[1].strip()
    return value

def api(path, method='GET', body=None):
    data = json.dumps(body).encode() if body is not None else None
    headers = {'Content-Type': 'application/json'}
    if token():
        headers['Authorization'] = 'Bearer ' + token()
    req = urllib.request.Request(BASE + '/api' + path, data=data, method=method, headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=60) as response:
            return json.load(response)
    except urllib.error.HTTPError as e:
        message = json.load(e).get('error', {}).get('message', 'API error')
        raise RuntimeError(f'DriftLens API: {message}') from None

def up():
    for command in ('docker', 'kind', 'kubectl', 'helm'):
        if not shutil.which(command):
            raise RuntimeError(f'{command} is required. See README prerequisites.')
    run('docker', 'info', '--format', '{{.ServerVersion}}', capture=True)
    DATA.mkdir(exist_ok=True)
    CREDS.mkdir(exist_ok=True)
    clusters = run('kind', 'get', 'clusters', capture=True).splitlines()
    if 'driftlens-demo' not in clusters:
        run('kind', 'create', 'cluster', '--name', 'driftlens-demo', '--config', ROOT / 'demo/kind/cluster.yaml', '--kubeconfig', ADMIN, '--wait', '120s')
    else:
        run('kind', 'export', 'kubeconfig', '--name', 'driftlens-demo', '--kubeconfig', ADMIN)
    kubectl('apply', '-f', ROOT / 'demo/kind/reader.yaml')
    reset()
    for name in ('api', 'frontend'):
        kubectl('-n', NAMESPACE, 'rollout', 'status', 'deployment/' + name, '--timeout=180s')
    kubectl('-n', NAMESPACE, 'rollout', 'status', 'statefulset/redis', '--timeout=180s')
    # Copy only cluster endpoint/CA, then replace the admin identity with a short-lived reader.
    raw = json.loads(kubectl('config', 'view', '--raw', '--minify', '-o', 'json', capture=True))
    reader_token = kubectl('-n', 'kube-system', 'create', 'token', 'driftlens-reader', '--duration=24h', capture=True)
    cluster_name = raw['contexts'][0]['context']['cluster']
    cfg = {'apiVersion': 'v1', 'kind': 'Config', 'clusters': raw['clusters'], 'contexts': [{'name': 'driftlens-demo', 'context': {'cluster': cluster_name, 'user': 'driftlens-reader'}}], 'current-context': 'driftlens-demo', 'users': [{'name': 'driftlens-reader', 'user': {'token': reader_token}}]}
    path = CREDS / 'demo.kubeconfig'
    path.write_text(json.dumps(cfg, indent=2))
    path.chmod(0o600)
    # For Compose, the backend reaches the Kind API via the host instead of container loopback.
    docker_cfg = json.loads(json.dumps(cfg))
    for cluster in docker_cfg['clusters']:
        cluster['cluster']['server'] = cluster['cluster']['server'].replace('127.0.0.1', 'host.docker.internal')
        cluster['cluster']['tls-server-name'] = '127.0.0.1'
    docker_path = CREDS / 'demo-docker.kubeconfig'
    docker_path.write_text(json.dumps(docker_cfg, indent=2))
    docker_path.chmod(0o600)
    print('Demo ready. Read-only kubeconfig created; token expires in approximately 24 hours.')

def connect(docker=False):
    repos = api('/repositories')
    repo = next((r for r in repos if r['name'] == 'driftlens-demo'), None)
    if not repo:
        repo = api('/repositories', 'POST', {'name': 'driftlens-demo', 'url': 'infra-repo', 'branch': ''})
    clusters = api('/clusters')
    cluster = next((c for c in clusters if c['name'] == 'driftlens-demo'), None)
    if not cluster:
        cluster = api('/clusters', 'POST', {'name': 'driftlens-demo', 'kubeconfig': 'demo-docker.kubeconfig' if docker else 'demo.kubeconfig', 'context': ''})
    api('/clusters/' + cluster['id'] + '/test', 'POST', {})
    maps = api('/mappings')
    if not any(m['name'] == 'demo-development' for m in maps):
        api('/mappings', 'POST', {'name': 'demo-development', 'repositoryId': repo['id'], 'clusterId': cluster['id'], 'chartPath': 'charts/platform', 'release': 'platform', 'namespace': NAMESPACE, 'valuesFiles': ['charts/platform/values-dev.yaml'], 'ignorePaths': []})
    print('Repository, read-only cluster and explicit environment mapping connected.')
    scan()

def scan():
    previous = api('/state')['snapshot']['id']
    api('/scans', 'POST', {})
    deadline = time.monotonic() + 320
    while time.monotonic() < deadline:
        state = api('/state')
        if not state['scanning'] and state['snapshot']['id'] != previous:
            print('Scan:', state['snapshot']['status'], state['snapshot']['summary']['classifications'])
            if state['snapshot']['errors']:
                print('Scan issues:', json.dumps(state['snapshot']['errors']))
            return state['snapshot']
        time.sleep(2)
    raise RuntimeError('Scan did not finish before timeout')

def drift():
    kubectl('-n', NAMESPACE, 'scale', 'deployment/api', '--replicas=1')
    kubectl('-n', NAMESPACE, 'delete', 'configmap/platform-config', '--ignore-not-found')
    kubectl('-n', NAMESPACE, 'create', 'configmap', 'manual-extra', '--from-literal=owner=manual')
    kubectl('-n', NAMESPACE, 'set', 'image', 'deployment/frontend', 'frontend=nginx:driftlens-intentionally-missing')
    print('Introduced replica drift, a missing ConfigMap, an extra ConfigMap, and a bad image.')

def verify(expect):
    if expect == 'drift':
        # Give the changed Deployment status time to report unavailable replicas.
        time.sleep(8)
    snap = scan()
    rows = [r for r in snap['drift'] if r['environment'] == 'demo-development']
    if not rows:
        raise AssertionError('No demo comparisons found; run demo connect first')
    if expect == 'sync':
        failures = [(r['kind'], r['name'], r['classification'], r['differences']) for r in rows if r['classification'] != 'IN_SYNC']
        if failures:
            raise AssertionError('Expected initial sync: ' + json.dumps(failures))
        print('PASS: all declared demo resources are in sync.')
        return
    def find(kind, name):
        return next(r for r in rows if r['kind'] == kind and r['name'] == name)
    deployment = find('Deployment', 'api')
    assert deployment['classification'] == 'MODIFIED'
    assert any(d['path'] == '/spec/replicas' and d['desired'] == 3 and d['actual'] == 1 for d in deployment['differences'])
    assert find('ConfigMap', 'platform-config')['classification'] == 'MISSING'
    assert find('ConfigMap', 'manual-extra')['classification'] == 'EXTRA'
    frontend = find('Deployment', 'frontend')
    assert frontend['classification'] == 'MODIFIED'
    assert any(d['path'].endswith('/image') for d in frontend['differences'])
    assert frontend['actual']['health'] == 'Degraded', 'Wait for the rollout status to become degraded, then verify again'
    print('PASS: replicas 3 → 1, missing, extra, changed image and unhealthy workload detected.')

def reset():
    run('helm', 'upgrade', '--install', 'platform', ROOT / 'demo/infra-repo/charts/platform', '--kubeconfig', ADMIN, '--kube-context', CONTEXT, '--namespace', NAMESPACE, '--create-namespace', '-f', ROOT / 'demo/infra-repo/charts/platform/values-dev.yaml')
    kubectl('-n', NAMESPACE, 'delete', 'configmap/manual-extra', '--ignore-not-found')

def down():
    run('kind', 'delete', 'cluster', '--name', 'driftlens-demo')
    print('Only the driftlens-demo Kind cluster was deleted. Remove its connections in DriftLens if desired.')

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['up', 'connect', 'scan', 'drift', 'verify', 'reset', 'down'])
    parser.add_argument('--expect', choices=['sync', 'drift'], default='drift')
    parser.add_argument('--docker', action='store_true')
    args = parser.parse_args()
    try:
        if args.action == 'connect': connect(args.docker)
        elif args.action == 'verify': verify(args.expect)
        else: globals()[args.action]()
    except (RuntimeError, AssertionError, OSError, urllib.error.URLError) as e:
        print('ERROR:', e, file=sys.stderr)
        sys.exit(1)
