# Helpdesk System

Простая Helpdesk система на Go для малой нагрузки с поддержкой мультитенантности, Telegram бота и интеграцией с Google Calendar.

## Возможности

- ✅ Мультитенантность (разные организации)
- ✅ Telegram bot для общения с клиентами
- ✅ Веб-интерфейс для администраторов и агентов
- ✅ Интеграция с Google Calendar
- ✅ Локальное хранение файлов
- ✅ Простая архитектура без сложных зависимостей

## Требования

- Go 1.21+
- PostgreSQL 12+
- Docker и Docker Compose (опционально)
- Telegram Bot Token
- Google OAuth2 credentials (опционально)

## Установка

### 1. Клонирование и настройка

```bash
git clone <repository>
cd helpdesk
```

### 2. Настройка окружения

Создайте файл `.env` на основе `env.example`:

```bash
cp env.example .env
```

Отредактируйте `.env` и укажите:
- Параметры подключения к БД
- `TELEGRAM_BOT_TOKEN` - токен вашего Telegram бота
- `GOOGLE_CLIENT_ID` и `GOOGLE_CLIENT_SECRET` (опционально)
- `SESSION_SECRET` - случайная строка для сессий

### 3. Запуск PostgreSQL

С помощью Docker Compose:

```bash
docker-compose up -d
```

Или используйте существующий PostgreSQL сервер.

### 4. Установка зависимостей

```bash
go mod download
```

### 5. Запуск приложения

```bash
go run main.go
```

Приложение будет доступно по адресу `http://localhost:8080`

## Первый запуск

После первого запуска создается:
- База данных с таблицами
- Организация по умолчанию (ID=1)
- Администратор: `admin@example.com` / `admin123`

**⚠️ ВАЖНО:** Смените пароль администратора после первого входа!

## Структура проекта

```
helpdesk/
├── main.go                 # Точка входа
├── internal/
│   ├── auth/              # Аутентификация и сессии
│   ├── bot/               # Telegram bot
│   ├── calendar/          # Google Calendar интеграция
│   ├── db/                # Работа с БД
│   ├── handlers/          # HTTP handlers
│   └── models/            # Модели данных
├── migrations/            # SQL миграции
├── templates/             # HTML шаблоны
├── uploads/               # Загруженные файлы
├── docker-compose.yml     # Docker Compose конфигурация
└── README.md
```

## Использование

### Telegram Bot

1. Создайте бота через [@BotFather](https://t.me/BotFather)
2. Получите токен и укажите его в `.env`
3. Клиенты могут отправлять сообщения боту для создания тикетов
4. Ответьте на сообщение бота, чтобы добавить комментарий к тикету

### Веб-интерфейс

1. Откройте `http://localhost:8080`
2. Войдите как администратор или агент
3. Просматривайте и управляйте тикетами
4. Назначайте тикеты агентам
5. Обновляйте статусы тикетов

### Google Calendar

1. Создайте OAuth2 credentials в [Google Cloud Console](https://console.cloud.google.com/)
2. Добавьте redirect URI: `http://localhost:8080/auth/google/callback`
3. Укажите `GOOGLE_CLIENT_ID` и `GOOGLE_CLIENT_SECRET` в `.env`
4. Войдите в веб-интерфейс и перейдите на `/auth/google` для авторизации

## API Endpoints

### Публичные
- `GET /login` - Страница входа
- `POST /login` - Авторизация
- `GET /logout` - Выход

### Защищенные
- `GET /dashboard` - Дашборд с тикетами
- `GET /ticket/{id}` - Просмотр тикета
- `POST /ticket/message` - Добавить сообщение
- `POST /ticket/status` - Изменить статус
- `POST /ticket/assign` - Назначить агента
- `GET /auth/google` - Авторизация Google Calendar
- `GET /auth/google/callback` - Callback для OAuth

## Роли пользователей

- **admin** - Полный доступ ко всем функциям
- **agent** - Может работать с тикетами, назначать и изменять статусы
- **customer** - Может создавать тикеты и добавлять сообщения

## Развертывание на Debian 12/13

### 1. Установка зависимостей

```bash
sudo apt update
sudo apt install -y postgresql postgresql-contrib golang-go
```

### 2. Настройка PostgreSQL

```bash
sudo -u postgres psql
CREATE DATABASE helpdesk;
CREATE USER helpdesk WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE helpdesk TO helpdesk;
\q
```

### 3. Сборка приложения

```bash
go build -o helpdesk main.go
```

### 4. Запуск как сервис

Создайте файл `/etc/systemd/system/helpdesk.service`:

```ini
[Unit]
Description=Helpdesk Service
After=network.target postgresql.service

[Service]
Type=simple
User=helpdesk
WorkingDirectory=/opt/helpdesk
ExecStart=/opt/helpdesk/helpdesk
Restart=always
EnvironmentFile=/opt/helpdesk/.env

[Install]
WantedBy=multi-user.target
```

Запустите сервис:

```bash
sudo systemctl enable helpdesk
sudo systemctl start helpdesk
```

## Безопасность

- ⚠️ Смените пароль администратора по умолчанию
- ⚠️ Используйте сильный `SESSION_SECRET`
- ⚠️ Настройте HTTPS в production
- ⚠️ Ограничьте доступ к БД
- ⚠️ Регулярно обновляйте зависимости

## Лицензия

MIT

## Поддержка

При возникновении проблем проверьте логи приложения и убедитесь, что:
- PostgreSQL запущен и доступен
- Все переменные окружения установлены
- Telegram bot token валиден
- Порты 8080 и 5432 свободны
