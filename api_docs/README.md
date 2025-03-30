# Main Handlers

## Authorization (POST)
### Принимает:
```json
{
  "email": "string",
  "password": "string"
}
```
### Возвращает:
- **200 OK** при успешной авторизации
```json
{
  "token": "jwt_token",
  "user_id": "uuid"
}
```
- **401 Unauthorized** если неверные данные
```json
{
  "error": "Invalid credentials"
}
```

## ChangeProfile (PUT)
### Принимает:
```json
{
  "username": "string",
  "male": "bool",
  "firstName": "string",
  "lastName": "string",
  "age": "int"
}
```
### Возвращает:
- **200 OK** при успешном обновлении
```json
{
  "message": "Profile updated successfully"
}
```
- **400 Bad Request** если данные некорректны
```json
{
  "error": "Invalid input data"
}
```

## Info (GET)
### Принимает:
```json
{
  "user_id": "uuid" 
}
```

### Возвращает:
- **200 OK** при успешном запросе
```json
{
  "user_id": "uuid",
  "username": "string",
  "email": "string",
  "firstName": "string",
  "lastName": "string",
  "age": "int",
  "male": "bool"
}
```
- **401 Unauthorized** если токен неверный или отсутствует
```json
{
  "error": "Unauthorized"
}
?