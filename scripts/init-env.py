from pathlib import Path
import secrets
root = Path(__file__).resolve().parents[1]
path = root / '.env'
if path.exists():
    print('.env already exists; left unchanged.')
else:
    path.write_text('DRIFTLENS_API_TOKEN=' + secrets.token_urlsafe(32) + '\n')
    path.chmod(0o600)
    print('Created .env with a random local API token. Read this file to sign in.')
(root / 'credentials').mkdir(exist_ok=True)
(root / '.data').mkdir(exist_ok=True)
