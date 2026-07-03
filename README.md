# university-pass

## Запуск

Внутри ./backend 

```
docker compose up -d
```

Стартует бд с одним пользователем и бэкенд на порту 8080

http://localhost:8080/auth/login

Принимает POST запрос с телом:

```
{
    "email": "student1@uni.com",
    "password": "password123",
    "device_id": "12345-ababab-test"
}
```

Пример ответа:

```
{
    "secret_key": "HQZD35IMGBKSSESVBJZZM4MOFDEW5ONI"
}
```

secret_key при каждом новом логине новый

Остановить контейнеры:
```
docker compose down -v
```