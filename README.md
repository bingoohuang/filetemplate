# file-template

provide http  and cli api for config file overwrite and reload

## usage

```bash
$ curl -H "Content-type: application/json" "http://localhost:3003/file" -d '{"filename": "/tmp/my.cnf","content": "this is bingoohuang"}'
{"code":0,"message":"OK"}
```
