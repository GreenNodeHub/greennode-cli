# Output Formatting

The shared formatter normalizes values through JSON before rendering or applying `--query`, so queries use the same field names as JSON output (including struct JSON tags). Table/text output retains custom columns and displays large integer values exactly when supplied as integers or `json.Number`; it cannot recover precision already lost by an earlier float64 decode. JMESPath numerical operations use float64 and can round integers beyond 2^53. Invalid queries, non-JSON values, and output-write errors fail explicitly. An invalid configured shared output format is rejected; an explicit valid `--output` overrides it. AgentBase retains its separate output helper.

## Output formats

Credential-returning commands mask secret fields as `[REDACTED]` by default. Their `--show-secret` flag permits explicit disclosure in command output, not debug or error output. Set it before a mutation that returns a one-time credential; repeating the mutation may create or rotate another credential. See each service reference for supported commands and file-delivery behavior.

```bash
grn vks list-clusters --output json    # JSON (default)
grn vks list-clusters --output table   # Table
grn vks list-clusters --output text    # Tab-separated text
```

### JSON (default)

```json
{
    "items": [
        {
            "id": "k8s-xxxxx",
            "name": "my-cluster",
            "status": "ACTIVE"
        }
    ],
    "total": 1
}
```

### Table

```
id        | name       | status
----------+------------+-------
k8s-xxxxx | my-cluster | ACTIVE
```

### Text

```
k8s-xxxxx	my-cluster	ACTIVE
```

## JMESPath query

Use `--query` to filter response data with [JMESPath](https://jmespath.org/) expressions:

```bash
# Get only cluster names
grn vks list-clusters --query "items[].name"
# ["my-cluster", "prod-cluster"]

# Get cluster status
grn vks get-cluster --cluster-id k8s-xxxxx --query "status"
# "ACTIVE"

# Filter clusters by status
grn vks list-clusters --query "items[?status=='ACTIVE'].name"
# ["my-cluster"]
```
