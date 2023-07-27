import os

meli_access_token = os.environ.get('MELI_ACCESS_TOKEN')

if meli_access_token is not None:
    print("MELI_ACCESS_TOKEN:", meli_access_token)
else:
    print("MELI_ACCESS_TOKEN environment variable is not set.")
