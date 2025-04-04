# Main Handlers

## Registration (POST)
### Принимает:
```json
{
  "email": "string",
  "password": "string",
  "phone_number": "string",
  "name": "string",
  "surname": "string"
}
```
### Возвращает:
- **200 OK** при успешной авторизации
```json
{
  "token": "jwt_token"
}
```
- **401 Unauthorized** если неверные данные
```json
{
  "error": "Invalid credentials"
}
```

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
  "token": "jwt_token"
}
```
- **401 Unauthorized** если неверные данные
```json
{
  "error": "Invalid credentials"
}
```

## UpdateProfile (PUT)
### Принимает:
```json
{
  "name": "string",
  "surname": "string",
  "age": "int",
  "email": "string",
  "phone_number": "string"
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
  "email": "string",
  "name": "string",
  "surname": "string",
  "age": "int",
  "male": "bool",
  "phone_number": "string"
}
```
- **401 Unauthorized** если токен неверный или отсутствует
```json
{
  "error": "Unauthorized"
}
?