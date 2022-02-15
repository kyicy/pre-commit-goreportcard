# pre-commit-goreportcard
golang https://github.com/gojp/goreportcard hook for http://pre-commit.com/


Add this to your .pre-commit-config.yaml

```yaml
- repo: https://github.com/kyicy/pre-commit-goreportcard
  rev: master
  hooks:
    - id: goreportcard
```