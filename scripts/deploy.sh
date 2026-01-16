#!/bin/bash
# Скрипт для быстрого развертывания на Debian

set -e

echo "=== Helpdesk Deployment Script ==="

# Проверка прав root
if [ "$EUID" -ne 0 ]; then 
    echo "Пожалуйста, запустите скрипт с правами root (sudo)"
    exit 1
fi

# Переменные
APP_USER="helpdesk"
APP_DIR="/home/$APP_USER/016.help"
DB_NAME="helpdesk"
DB_USER="helpdesk"

echo "1. Создание пользователя..."
if ! id "$APP_USER" &>/dev/null; then
    useradd -m -s /bin/bash $APP_USER
    echo "Пользователь $APP_USER создан"
else
    echo "Пользователь $APP_USER уже существует"
fi

echo "2. Установка зависимостей..."
apt update
apt install -y git golang-go postgresql postgresql-contrib

echo "3. Настройка PostgreSQL..."
# Генерация случайного пароля
DB_PASSWORD=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-25)

sudo -u postgres psql << EOF
DO \$\$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_user WHERE usename = '$DB_USER') THEN
        CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';
    END IF;
END
\$\$;

SELECT 'CREATE DATABASE $DB_NAME'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '$DB_NAME')\gexec

GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;
EOF

echo "4. Клонирование репозитория..."
if [ ! -d "$APP_DIR" ]; then
    sudo -u $APP_USER git clone https://github.com/maki072/016.help.git $APP_DIR
else
    echo "Директория уже существует, обновление..."
    sudo -u $APP_USER bash -c "cd $APP_DIR && git pull"
fi

echo "5. Настройка .env файла..."
if [ ! -f "$APP_DIR/.env" ]; then
    sudo -u $APP_USER cp $APP_DIR/env.example $APP_DIR/.env
    # Установка пароля БД
    sudo -u $APP_USER sed -i "s/DB_PASSWORD=.*/DB_PASSWORD=$DB_PASSWORD/" $APP_DIR/.env
    # Генерация SESSION_SECRET
    SESSION_SECRET=$(openssl rand -hex 32)
    sudo -u $APP_USER sed -i "s/SESSION_SECRET=.*/SESSION_SECRET=$SESSION_SECRET/" $APP_DIR/.env
    echo "Файл .env создан. Пожалуйста, отредактируйте его:"
    echo "  nano $APP_DIR/.env"
    echo ""
    echo "ВАЖНО: Сохраните пароль БД: $DB_PASSWORD"
else
    echo "Файл .env уже существует"
fi

echo "6. Сборка приложения..."
sudo -u $APP_USER bash -c "cd $APP_DIR && go mod download && go build -o helpdesk main.go"

echo "7. Создание systemd сервиса..."
cat > /etc/systemd/system/helpdesk.service << EOF
[Unit]
Description=Helpdesk Service
After=network.target postgresql.service
Requires=postgresql.service

[Service]
Type=simple
User=$APP_USER
Group=$APP_USER
WorkingDirectory=$APP_DIR
EnvironmentFile=$APP_DIR/.env
ExecStart=$APP_DIR/helpdesk
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=helpdesk

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable helpdesk

echo "8. Запуск сервиса..."
systemctl start helpdesk

echo ""
echo "=== Развертывание завершено! ==="
echo ""
echo "Проверьте статус: sudo systemctl status helpdesk"
echo "Просмотр логов: sudo journalctl -u helpdesk -f"
echo ""
echo "ВАЖНО:"
echo "1. Отредактируйте $APP_DIR/.env и укажите TELEGRAM_BOT_TOKEN"
echo "2. Перезапустите сервис: sudo systemctl restart helpdesk"
echo "3. Пароль БД сохранен в файле .env"
