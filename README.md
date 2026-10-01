# skillswap_lab_sem3

SkillSwap — обмен навыками. Командный проект курса «Микросервисы на Go».

## Проверки

Те же шаги запускает CI (`.github/workflows/ci.yml`) на каждый PR и push в `main`:

```bash
gofmt -l .        # должен ничего не вывести
go vet ./...
go build ./...
go test -race ./...
```

PR мержится только при зелёном CI и approve от участника, который не автор кода.
