import hashlib
import json
import os
import sys
import boto3

p=json.load(sys.stdin);r=p['repository'];a=p['artifact'];key=p['key']
s=boto3.client('s3',endpoint_url=r['endpoint'],region_name=r.get('region') or 'us-east-1',aws_access_key_id=p['credentials']['accessKey'],aws_secret_access_key=p['credentials']['secretKey'])
s.upload_file(a['file'],r['bucket'],key,ExtraArgs={'Metadata':{'sha256':a['sha256']}})
# HEAD/ETag alone cannot establish byte integrity for multipart objects.
response=s.get_object(Bucket=r['bucket'],Key=key)
h=hashlib.sha256()
for chunk in iter(lambda:response['Body'].read(1024*1024),b''):h.update(chunk)
response['Body'].close()
if h.hexdigest()!=a['sha256']:raise RuntimeError('remote backup checksum mismatch')
report={'verified':True,'bucket':r['bucket'],'key':key,'sha256':h.hexdigest(),'versionId':response.get('VersionId')}
with open(os.path.join(os.path.dirname(a['file']),'repository-report.json'),'w') as f:json.dump(report,f)
