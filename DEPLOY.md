# Инструкция по развертыванию на Debian

## 1. Выгрузка в GitHub

### На локальной машине:

```bash
# Инициализация git (если еще не сделано)
git init

# Добавление удаленного репозитория
git remote add origin https://github.com/maki072/016.help.git

# Добавление всех файлов
git add .

# Коммит
git commit -m "Initial commit: Helpdesk system"

# Выгрузка в GitHub
git push -u origin main
```

Если репозиторий уже существует и есть файлы, используйте:

```bash
git pull origin main --allow-unrelated-histories
git push -u origin main
```

## 2. Развертывание на Debian сервере

### Шаг 1: Подготовка сервера

```bash
# Обновление системы
sudo apt update && sudo apt upgrade -y

# Установка необходимых пакетов
sudo apt install -y git golang-go postgresql postgresql-contrib nginx certbot python3-certbot-nginx
```

### Шаг 2: Клонирование репозитория

```bash
# Создание пользователя для приложения
sudo useradd -m -s /bin/bash helpdesk

# Переключение на пользователя
sudo su - helpdesk

# Клонирование репозитория
cd ~
git clone https://github.com/maki072/016.help.git
cd 016.help
```

### Шаг 3: Настройка PostgreSQL

```bash
# Выйти из пользователя helpdesk
exit

# Настройка PostgreSQL
sudo -u postgres psql << EOF
CREATE DATABASE helpdesk;
CREATE USER helpdesk WITH PASSWORD 'your_secure_password_here';
GRANT ALL PRIVILEGES ON DATABASE helpdesk TO helpdesk;
\q
EOF
```

### Шаг 4: Настройка приложения

```bash
# Вернуться к пользователю helpdesk
sudo su - helpdesk
cd ~/016.help

# Создание .env файла
cp env.example .env
nano .env  # Отредактируйте настройки
```

Важные настройки в `.env`:
- `DB_PASSWORD` - пароль из шага 3
- `TELEGRAM_BOT_TOKEN` - токен вашего бота
- `SESSION_SECRET` - случайная строка (можно сгенерировать: `openssl rand -hex 32`)
- `HTTP_HOST=0.0.0.0` - для прослушивания всех интерфейсов

### Шаг 5: Сборка приложения

```bash
# Установка зависимостей
go mod download

# Сборка
go build -o helpdesk main.go

# Проверка
./helpdesk  # Запустите в фоне для проверки, затем Ctrl+C
```

### Шаг 6: Создание systemd сервиса

```bash
# Выйти из пользователя helpdesk
exit

# Создание файла сервиса
sudo nano /etc/systemd/system/helpdesk.service
```

Содержимое файла:

```ini
[Unit]
Description=Helpdesk Service
After=network.target postgresql.service
Requires=postgresql.service

[Service]
Type=simple
User=helpdesk
Group=helpdesk
WorkingDirectory=/home/helpdesk/016.help
EnvironmentFile=/home/helpdesk/016.help/.env
ExecStart=/home/helpdesk/016.help/helpdesk
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=helpdesk

[Install]
WantedBy=multi-user.target
```

```bash
# Перезагрузка systemd
sudo systemctl daemon-reload

# Включение автозапуска
sudo systemctl enable helpdesk

# Запуск сервиса
sudo systemctl start helpdesk

# Проверка статуса
sudo systemctl status helpdesk

# Просмотр логов
sudo journalctl -u helpdesk -f
```

### Шаг 7: Настройка Nginx (опционально, для HTTPS)

```bash
sudo nano /etc/nginx/sites-available/helpdesk
```

Содержимое:

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

```bash
# Активация конфигурации
sudo ln -s /etc/nginx/sites-available/helpdesk /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx

# Настройка SSL (опционально)
sudo certbot --nginx -d your-domain.com
```

### Шаг 8: Настройка файрвола

```bash
# Разрешить HTTP/HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 22/tcp  # SSH
sudo ufw enable
```

## Обновление приложения

```bash
# Переключиться на пользователя helpdesk
sudo su - helpdesk
cd ~/016.help

# Остановить сервис
sudo systemctl stop helpdesk

# Обновить код
git pull origin main

# Пересобрать
go build -o helpdesk main.go

# Запустить сервис
sudo systemctl start helpdesk

# Проверить статус
sudo systemctl status helpdesk
```

## Полезные команды

```bash
# Просмотр логов
sudo journalctl -u helpdesk -n 100 -f

# Перезапуск сервиса
sudo systemctl restart helpdesk

# Проверка подключения к БД
sudo -u postgres psql -d helpdesk -c "SELECT COUNT(*) FROM organizations;"

# Проверка портов
sudo netstat -tlnp | grep 8080
```

## Устранение неполадок

### Приложение не запускается

1. Проверьте логи: `sudo journalctl -u helpdesk -n 50`
2. Проверьте .env файл: `cat /home/helpdesk/016.help/.env`
3. Проверьте права доступа: `ls -la /home/helpdesk/016.help/helpdesk`
4. Проверьте подключение к БД: `psql -U helpdesk -d helpdesk -h localhost`

### Проблемы с миграциями

```bash
# Подключиться к БД и проверить таблицы
sudo -u postgres psql -d helpdesk -c "\dt"

# Если нужно пересоздать БД (ОСТОРОЖНО: удалит все данные!)
sudo -u postgres psql << EOF
DROP DATABASE helpdesk;
CREATE DATABASE helpdesk;
GRANT ALL PRIVILEGES ON DATABASE helpdesk TO helpdesk;
\q
EOF
```

## Резервное копирование

```bash
# Создать скрипт бэкапа
sudo nano /usr/local/bin/helpdesk-backup.sh
```

```bash
#!/bin/bash
BACKUP_DIR="/home/helpdesk/backups"
DATE=$(date +%Y%m%d_%H%M%S)
mkdir -p $BACKUP_DIR

# Бэкап БД
sudo -u postgres pg_dump helpdesk > $BACKUP_DIR/db_$DATE.sql

# Бэкап файлов
tar -czf $BACKUP_DIR/uploads_$DATE.tar.gz /home/helpdesk/016.help/uploads/

# Удаление старых бэкапов (старше 7 дней)
find $BACKUP_DIR -type f -mtime +7 -delete
```

```bash
sudo chmod +x /usr/local/bin/helpdesk-backup.sh

# Добавить в cron (ежедневно в 2:00)
sudo crontab -e
# Добавить строку:
# 0 2 * * * /usr/local/bin/helpdesk-backup.sh
```
