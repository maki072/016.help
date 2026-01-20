# Установка Go 1.23+ на Debian 12/13

Debian 12 поставляется с устаревшей версией Go 1.19, которая не подходит для этого проекта. Необходима версия Go 1.21 или новее.

## Автоматическая установка

Скрипт `deploy.sh` автоматически устанавливает актуальную версию Go.

## Ручная установка

Если нужно установить Go вручную:

```bash
# Удалить старую версию из репозитория Debian (если установлена)
sudo apt remove golang-go

# Скачать и установить Go 1.23.6
wget https://go.dev/dl/go1.23.6.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.23.6.linux-amd64.tar.gz

# Добавить в PATH
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee -a /etc/profile
source /etc/profile

# Проверить установку
go version
```

Должно показать: `go version go1.23.6 linux/amd64`

## Для пользователя helpdesk

Убедитесь, что PATH настроен для пользователя helpdesk:

```bash
sudo -u helpdesk bash -c 'echo $PATH'
```

Если `/usr/local/go/bin` отсутствует, добавьте в `~/.bashrc`:

```bash
sudo -u helpdesk bash -c 'echo "export PATH=\$PATH:/usr/local/go/bin" >> ~/.bashrc'
```

## Проверка после установки

```bash
cd /home/helpdesk/016.help
sudo -u helpdesk go version
sudo -u helpdesk go build -o helpdesk main.go
```
