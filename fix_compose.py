import yaml

class MyDumper(yaml.SafeDumper):
    def increase_indent(self, flow=False, indentless=False):
        return super(MyDumper, self).increase_indent(flow, False)

with open('docker-compose.yml', 'r') as f:
    data = yaml.safe_load(f)

for svc_name, svc in data['services'].items():
    if 'restart' not in svc:
        svc['restart'] = 'unless-stopped'
    
    if 'ports' in svc:
        if svc_name == 'postgres':
            svc['ports'] = ['127.0.0.1:5433:5432']
        elif svc_name == 'redis':
            svc['ports'] = ['127.0.0.1:6380:6379']
        elif svc_name in ['api-gateway', 'n8n']:
            pass
        else:
            del svc['ports']

data['services']['umkm-frontend'] = {
    'build': {
        'context': './frontend/umkm-web',
        'dockerfile': 'Dockerfile'
    },
    'container_name': 'wch-umkm-frontend',
    'ports': ['80:80'],
    'restart': 'unless-stopped',
    'depends_on': ['api-gateway']
}

with open('docker-compose.yml', 'w') as f:
    yaml.dump(data, f, Dumper=MyDumper, default_flow_style=False, sort_keys=False)

