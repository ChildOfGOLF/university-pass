# university-pass

## Запуск

Переходим в /deploy и создаем .env по примеру

```bash
   cd deploy
   cp .env.example .env
```

Поднимаем проект

```
docker-compose up -d --build
```

- Фронт: http://localhost
- Сканер: http://localhost/scanner
- Админ: http://localhost/admin.html
- Документация: http://localhost:8080/swagger/index.html

Остановить и удалить контейнеры:

```
docker compose down -v
```
