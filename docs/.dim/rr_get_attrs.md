## request type A
{
  "type": "A",
  "layer3domain": "default",
}
## response

```
{
  "comment": "some comment"
  "created": "2024-04-26 20:22:31.000000"
  "created_by": "some_user"
  "modified": "2024-04-26 20:22:31.000000"
  "modified_by": "some_user"
  "rr": "somehost A 10.88.8.11"
  "zone": "example.com"
}
```

## request type CNAME
```
{
  "type": "CNAME",
  "name": "*.somehost.example.com.",
  "cname": "somehost.example.com."
}
```
## response

```
{
  "comment": "some comment"
  "created": "2024-04-26 20:22:31.000000"
  "created_by": "some_user"
  "modified": "2024-04-26 20:22:31.000000"
  "modified_by": "some_user"
  "rr": "*.somehost CNAME somehost.example.com."
  "zone": "example.com"
}
```
