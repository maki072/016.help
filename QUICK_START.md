# Быстрый старт

## Локальная разработка

```bash
# 1. Клонировать репозиторий
git clone https://github.com/maki072/016.help.git
cd 016.help

# 2. Запустить PostgreSQL через Docker
docker-compose up -d

# 3. Создать .env файл
cp env.example .env
# Отредактировать .env и указать TELEGRAM_BOT_TOKEN

# 4. Запустить приложение
go run main.go
```

## Развертывание на Debian (автоматическое)

```bash
# На сервере с Debian
sudo bash <(curl -s https://raw.githubusercontent.com/maki072/016.help/main/scripts/deploy.sh)
```

Или вручную:

```bash
# 1. Клонировать скрипт развертывания
wget https://raw.githubusercontent.com/maki072/016.help/main/scripts/deploy.sh
chmod +x deploy.sh
sudo ./deploy.sh

# 2. Отредактировать .env
sudo nano /home/helpdesk/016.help/.env
# Указать TELEGRAM_BOT_TOKEN

# 3. Перезапустить сервис
sudo systemctl restart helpdesk
```

## Развертывание на Debian (ручное)

См. подробную инструкцию в [DEPLOY.md](DEPLOY.md)

## Первый вход

1. Откройте `http://your-server:8080` или `http://your-domain.com`
2. Войдите как администратор:
   - Email: `admin@example.com`
   - Пароль: `admin123`
3. **СРОЧНО смените пароль!**

## Полезные команды

```bash
# Просмотр логов
sudo journalctl -u helpdesk -f

# Перезапуск
sudo systemctl restart helpdesk

# Статус
sudo systemctl status helpdesk
```
