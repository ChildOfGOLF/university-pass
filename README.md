# university-pass

## Запуск

Внутри в корне

```
docker-compose up --build
```

Стартует бд с одним пользователем, гостем, админом и бэкенд на порту 8080

На http://localhost:8081/swagger/index.html доступна документация

Остановить и удалить контейнеры:

```
docker compose down -v
```
